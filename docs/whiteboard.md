# Rancangan Whiteboard (W0)

Status: rancangan, belum ada kode. Ditulis 2026-11-24 untuk revisi besar dari Manajer: Whiteboard setara Milanote atau whiteboard Canva, dengan tema NVM. Dokumen ini menjadi acuan tahap W1 sampai W4. Keputusan di sini boleh diubah, tetapi perubahan dicatat di bagian "Riwayat keputusan" di akhir.

Fakta dari kode yang dipakai sebagai dasar:

- CSP di `frontend/nginx-security-headers.conf`: `script-src 'self'`, `style-src 'self'` (tanpa `unsafe-inline`), `img-src 'self' blob:` plus Drive, `connect-src 'self'`.
- Cookie sesi `nvm_session` HttpOnly, SameSite=Lax. `sameOrigin()` di `backend/internal/httpapi/protect.go` hanya memeriksa metode selain GET, HEAD, OPTIONS.
- Lampiran disimpan `storage.Store` di `attachments/<2 huruf>/<uuid>`. Job harian `maintenance` menghapus berkas di folder itu yang id-nya tidak ada di `ListAttachmentFileIDs` (UNION `task_attachments` dan `activity_attachments`).
- Lampiran task maksimal 50 MB (`task.MaxAttachmentSize`), pemeriksaan isi berkas di `attachment.Detect`.
- `deploy/sql/10-grants.sql` memberi DELETE per tabel, saat ini delapan tabel.
- `pg_trgm` sudah terpasang (migrasi `000002`). Migrasi terakhir `000022`.
- Satu container backend, satu proses Go. Publik lewat Cloudflare Tunnel ke nginx frontend.
- Token warna di `frontend/src/style.css`, termasuk pasangan latar dan teks status (`requested`, `progress`, `ready`, `completed`), terang dan gelap.

## Ringkasan keputusan

| No | Hal | Keputusan |
|---|---|---|
| 1 | Model data | Satu pohon `wb_boards` (folder dan papan), kartu di `wb_cards` dengan isi JSONB per jenis, `wb_connectors`, `wb_comments`, `wb_files`, `wb_activity`, `wb_link_previews`. Tempat sampah memakai kolom `deleted_at` dan `deleted_batch`, bukan tabel terpisah. `10-grants.sql` berubah. |
| 2 | Simpan dan konflik | Server sumber kebenaran. Per kartu dengan `version`, posisi, ukuran, dan warna digabung per field, isi dijaga 409. Kunci lunak per kartu saat mengetik, teks pemegang tampil langsung di layar orang lain. CRDT ditolak untuk sekarang. |
| 3 | Kolaborasi langsung | WebSocket untuk kehadiran (avatar, kursor bernama, redup saat diam), siapa mengubah apa (bingkai warna, label, gerakan seret 25 per detik), kunci, dan siaran. Perubahan data tetap lewat REST. Hub di memori proses backend. Kasus tepi dan penanda "Tersimpan" atau "Terputus, menyambung ulang" dirinci. |
| 4 | Frontend | DOM dengan CSS transform, culling di luar layar. TipTap 3 (MIT) hanya untuk kartu yang sedang diedit. Ekspor PNG dengan `html-to-image` (MIT), PDF dirakit sendiri dari JPEG. |
| 5 | Keamanan | Pratinjau tautan diambil server dengan pemeriksaan IP di dialer. Teks berformat disimpan sebagai JSON ProseMirror tervalidasi, dirender tanpa `v-html`. SVG ditolak. 50 MB. |
| 6 | Berkas | `storage.Store` yang sama, tabel `wb_files` masuk `ListAttachmentFileIDs`. Ekspor Drive: menunggu Manajer, bawaan tidak ikut. |
| 7 | Performa | 500 kartu: satu permintaan muat papan, render hanya yang terlihat, drag tanpa render ulang Vue. |
| 8 | Aksesibilitas | Kartu fokus dengan roving tabindex, pindah dengan Shift+panah, tampilan Daftar sebagai alternatif kanvas. |
| 9 | Tahap | W1 fondasi dan catatan, W2 kolaborasi langsung dan edit lanjutan, W3 berkas, tautan, kolom, penghubung, komentar, W4 template, ekspor, pencarian isi, poles. |

## 1. Model data dan migrasi

Semua tabel memakai awalan `wb_` supaya mudah dikenali di grants, job, dan backup. Id memakai UUID (`gen_random_uuid()`), karena kartu dibuat di browser saat offline sebentar dan saat tempel banyak kartu, dan id dari klien harus bisa dipakai langsung tanpa menunggu server. Server tetap memvalidasi bahwa id belum dipakai.

### Tabel

**`wb_boards`**: folder dan papan dalam satu pohon.

| Kolom | Tipe | Catatan |
|---|---|---|
| `id` | uuid PK | |
| `kind` | text CHECK `folder` atau `board` | Folder tidak punya kanvas, hanya berisi folder dan papan. Papan punya kanvas. |
| `parent_id` | uuid NULL FK `wb_boards` | NULL berarti di Beranda Whiteboard. Papan di dalam papan memakai kolom ini juga. |
| `title` | text, 1 sampai 120 karakter | |
| `color` | text NULL | Nama token, bukan kode warna. |
| `is_template` | boolean | Template adalah papan biasa yang ditandai (W4). |
| `seq` | bigint | Naik setiap ada perubahan apa pun di papan ini; dasar sinkron ulang setelah reconnect. |
| `version` | int | Optimistic locking untuk judul, warna, pindah induk. |
| `created_by`, `created_at`, `updated_by`, `updated_at` | | |
| `deleted_at`, `deleted_by`, `deleted_batch` | NULL | Tempat sampah. |

CHECK `parent_id <> id`. Siklus (papan dipindah ke dalam turunannya sendiri) dicek di Go dengan CTE rekursif di dalam transaksi yang sama, karena CHECK tidak bisa melihat baris lain. Kedalaman maksimal 8, supaya breadcrumb tetap terbaca di ponsel.

Papan di dalam papan tampil sebagai kartu jenis `board` di kanvas induknya, seperti Milanote. Kartu itu hanya menyimpan posisi dan `ref_board_id`; judul diambil dari papan anak. Papan di dalam folder tampil di daftar folder, bukan di kanvas.

**`wb_cards`**

| Kolom | Tipe | Catatan |
|---|---|---|
| `id` | uuid PK | |
| `board_id` | uuid FK | |
| `kind` | text CHECK | `note`, `heading`, `checklist`, `image`, `file`, `link`, `column`, `board` |
| `x`, `y`, `w`, `h` | double precision | Koordinat dunia dalam piksel pada zoom 100%. `w` dan `h` dibatasi 40 sampai 4000. |
| `z` | double precision | Urutan tumpuk. Bilangan pecahan supaya "bawa ke depan" cukup mengubah satu baris. |
| `column_id` | uuid NULL FK `wb_cards` | Kartu yang ditumpuk di kartu kolom. Bila terisi, `x` dan `y` diabaikan. |
| `col_order` | double precision NULL | Urutan di dalam kolom, pecahan dengan alasan yang sama dengan `z`. |
| `color` | text NULL | Nama token. |
| `content` | jsonb | Isi per jenis, divalidasi di Go (lihat bawah). Maksimal 64 KB. |
| `plain` | text | Teks polos dari `content`, diisi Go saat simpan, untuk pencarian. |
| `file_id` | uuid NULL FK `wb_files` | Untuk `image` dan `file`. |
| `ref_board_id` | uuid NULL FK `wb_boards` | Untuk `board`. |
| `version` | int | Naik setiap perubahan. |
| `content_version` | int | Naik hanya bila `content` berubah; dipakai aturan konflik di bagian 2. |
| `seq` | bigint | Nilai `wb_boards.seq` saat kartu terakhir berubah. |
| audit dan tempat sampah | | Sama dengan `wb_boards`. |

Isi `content` per jenis:

- `note`: `{ "doc": <dokumen ProseMirror> }`.
- `heading`: `{ "text": "...", "level": 1 | 2 }`.
- `checklist`: `{ "title": "...", "items": [{ "id": "<uuid>", "text": "...", "done": false }] }`, maksimal 200 butir.
- `image`: `{ "caption": "..." }` plus `file_id`.
- `file`: `{ "caption": "..." }` plus `file_id`.
- `link`: `{ "url": "https://...", "preview_id": "<id wb_link_previews>" | null }`.
- `column`: `{ "title": "..." }`.
- `board`: `{}` plus `ref_board_id`.

Kenapa JSONB, bukan tabel per jenis: delapan jenis kartu dengan field berbeda, sedangkan semua jenis berbagi geometri, warna, tempat sampah, dan siaran. Tabel per jenis berarti delapan query untuk memuat satu papan dan delapan jalur hapus dan pulihkan. Validasinya pindah ke Go, yang memang sudah tempat validasi lain (`apperr.Invalid`).

**`wb_connectors`**: garis dan panah.

| Kolom | Catatan |
|---|---|
| `id`, `board_id` | |
| `from_card_id`, `to_card_id` | FK `wb_cards`, harus di papan yang sama (dicek Go). |
| `style` | jsonb: `{ "start": "none" \| "arrow", "end": "none" \| "arrow", "line": "solid" \| "dashed", "color": "<token>", "label": "..." }` |
| `version`, `seq`, audit, tempat sampah | |

**`wb_comments`**: `id`, `card_id`, `user_id`, `body` (teks polos, 1 sampai 2000 karakter), `created_at`, `edited_at`, `deleted_at`, `deleted_by`. Teks polos, bukan teks berformat: cukup untuk diskusi dan menghapus satu jalur XSS.

**`wb_files`**: `id` (sama dengan nama berkas di storage), `board_id`, `kind` (`image` atau `file`), `original_name`, `mime_type`, `size_bytes`, `checksum`, `width`, `height`, `has_thumb`, `uploaded_by`, `created_at`. Tabel sendiri, bukan `task_attachments`, karena `task_attachments.task_id` wajib dan aturan hapusnya ikut task.

**`wb_link_previews`**: `id`, `url_hash` (UNIQUE, sha256 URL ternormalisasi), `url`, `final_url`, `title`, `description`, `site_name`, `image_file_id` (FK `wb_files`, NULL), `status` (`ok`, `blocked`, `failed`), `fetched_at`. Cache 7 hari.

**`wb_activity`**: riwayat per papan. `id` bigserial, `board_id`, `user_id`, `action`, `entity` (`board`, `card`, `connector`, `comment`), `entity_id`, `summary` jsonb, `created_at`. Perubahan geometri tidak dicatat satu per satu: seret dan ubah ukuran dari orang yang sama di papan yang sama digabung menjadi satu baris per 5 menit ("Rina memindahkan 6 kartu"). Tanpa penggabungan, satu sesi merapikan papan menghasilkan ratusan baris.

### Tempat sampah

Kolom `deleted_at`, `deleted_by`, `deleted_batch` di `wb_boards`, `wb_cards`, dan `wb_connectors`. Satu aksi hapus (mis. 12 kartu terpilih, atau satu folder beserta seluruh isinya) memakai satu `deleted_batch`, dan Tempat sampah menampilkan satu baris per batch: "Rina menghapus folder Kampanye Desember (3 papan, 140 kartu)". Pulihkan mengembalikan seluruh batch sekaligus.

Alternatif yang ditolak: tabel `wb_trash` berisi salinan JSON. Pulihkan dari salinan harus menulis ulang baris dan semua relasinya (berkas, penghubung, komentar), sementara soft delete cukup mengosongkan tiga kolom. Pola `is_deleted` juga sudah dipakai task dan lampiran.

Aturan pulihkan:

- Bila induknya masih ada: kembali ke tempat semula.
- Bila papan induk sebuah kartu juga di Tempat sampah: ditolak dengan pesan "Pulihkan dulu papan <judul>". Memulihkan papan diam-diam akan mengejutkan orang yang menghapusnya.
- Bila folder induk sebuah papan sudah dihapus permanen: papan kembali ke Beranda Whiteboard dan pesannya menyebut itu.
- Penghubung yang salah satu ujungnya masih di Tempat sampah tetap tersembunyi sampai kedua ujungnya pulih.

Hapus permanen hanya oleh job harian `maintenance` (sudah dipanggil cron pukul 03.30 WIB): batch yang `deleted_at`-nya lebih dari 30 hari dihapus dengan `DELETE`, anak lebih dulu. Baris `wb_files` ikut terhapus, lalu berkas di disk dibersihkan oleh pembersih berkas yatim yang sudah ada pada run berikutnya (setelah `orphanGrace`). Tidak ada tombol "Hapus permanen sekarang" di W1 sampai W4, karena itu satu-satunya aksi yang tidak bisa dibatalkan.

### Indeks

- `wb_boards (parent_id, title) WHERE deleted_at IS NULL`: isi folder dan papan anak.
- `wb_cards (board_id) WHERE deleted_at IS NULL`: muat papan.
- `wb_cards (board_id, seq)`: sinkron ulang setelah reconnect, termasuk kartu yang baru dihapus.
- `wb_cards (column_id, col_order) WHERE column_id IS NOT NULL AND deleted_at IS NULL`.
- `wb_connectors (board_id) WHERE deleted_at IS NULL`, `wb_connectors (from_card_id)`, `wb_connectors (to_card_id)`.
- `wb_comments (card_id, created_at)`.
- `wb_activity (board_id, created_at DESC)`.
- `deleted_batch` dan `deleted_at` di ketiga tabel, `WHERE deleted_at IS NOT NULL`: daftar Tempat sampah dan job hapus permanen tanpa memindai baris hidup.
- GIN `gin_trgm_ops` pada `wb_boards.title` dan `wb_cards.plain`, `WHERE deleted_at IS NULL`: pencarian.
- `wb_link_previews (url_hash)` UNIQUE.

FK dari `wb_cards.file_id`, `ref_board_id`, `column_id`, dan penghubung diindeks juga, karena hapus permanen menghapus induk dan Postgres memindai anak lewat FK.

### Migrasi dan grants

- W1: `000023_whiteboard` (boards, cards, activity, kolom tempat sampah, indeks). W2 tanpa migrasi. W3: `000024_whiteboard_media` (files, link_previews, connectors, comments, kolom `file_id`, `ref_board_id`, `column_id`). W4: hanya bila template butuh kolom tambahan. Setiap migrasi dengan `up` dan `down`.
- `deploy/sql/10-grants.sql` **berubah**: `nvm_internal_app` butuh DELETE pada `wb_boards`, `wb_cards`, `wb_connectors`, `wb_comments`, `wb_files`, dan `wb_link_previews`, karena job `maintenance` berjalan sebagai role aplikasi dan menghapus permanen. `wb_activity` tidak diberi DELETE sampai Manajer memutuskan masa simpannya. Daftar DELETE di runbook berubah dari delapan menjadi 12 (W1) lalu 14 (W3). `deploy/rilis.sh` sudah menjalankan grants setiap ada migrasi.
- `ListAttachmentFileIDs` wajib ditambah `UNION ALL SELECT id::text FROM wb_files` di migrasi yang sama dengan tabelnya. Tanpa itu, job harian menghapus semua berkas Whiteboard sebagai berkas yatim (pelajaran revisi 24).
- `testdb.Pool` menambah tabel baru ke daftar TRUNCATE.

## 2. Strategi simpan dan konflik

Sasaran dari Manajer: realtime penuh setara Google Sheets, Milanote, atau Canva. Semua orang tahu siapa yang ada di papan dan siapa sedang mengubah apa, dan tidak ada perubahan yang hilang atau saling menimpa diam-diam.

### Prinsip: server sumber kebenaran

- Setiap perubahan yang tersimpan lewat REST: hak akses (Admin hanya baca), validasi isi, dan versi diperiksa di server di dalam transaksi, baru kemudian disiarkan ke yang lain. Klien tidak pernah menyiarkan perubahan data langsung ke klien lain.
- Yang boleh lewat WebSocket tanpa disimpan hanya hal sementara: kursor, pilihan, gerakan seret yang sedang berlangsung, dan draf teks pemegang kunci. Server tetap memeriksa pengirimnya (bukan Admin, memegang kunci kartu itu, isi draf lolos validasi skema) sebelum meneruskannya.
- Setiap perubahan tersimpan tercatat di `wb_activity` (siapa, apa, kapan, kartu mana). Gerakan geometri beruntun dari orang yang sama digabung menjadi satu baris per 5 menit, dengan daftar id kartu dan rentang waktunya, jadi tetap terlacak tanpa ratusan baris. Edit isi dicatat satu baris per sesi edit (dari kunci diambil sampai dilepas).

### Patch per field

- `PATCH /api/whiteboard/cards/:id` berisi `{ base_version, base_content_version, fields }`, dan `fields` hanya memuat field yang benar-benar berubah. Banyak kartu sekaligus (seret 30 kartu, rata kiri, tempel) lewat `POST /api/whiteboard/boards/:id/patch` dalam satu transaksi.
- Posisi, ukuran, urutan tumpuk, warna, dan kolom digabung **per field** dengan last-write-wins: satu orang mengganti warna dan orang lain menggeser kartu yang sama, keduanya tersimpan. Bila dua orang mengubah field yang sama, yang terakhir sampai ke server menang, dan yang kalah melihat hasil akhirnya lewat siaran. Supaya tidak diam-diam, klien yang patch-nya ditimpa dalam 2 detik oleh orang lain menampilkan label sebentar di kartu: "Dipindah Nala".
- Isi (`content`) dijaga `base_content_version`: patch dengan versi isi basi ditolak 409. Dengan kunci lunak (di bawah) ini hampir tidak pernah terjadi; bila terjadi (mis. kunci kedaluwarsa saat koneksi putus lama), ketikan tidak dibuang: klien membuat kartu salinan di sebelahnya dan menampilkan "Kartu ini sudah diubah Nala. Ketikanmu disimpan di kartu baru di sebelahnya."
- Checklist: centang dan teks butir memakai patch per butir (`items[<id>]`), jadi dua orang mencentang butir berbeda tidak saling menimpa, dan mencentang tidak butuh kunci.
- Setiap perubahan yang diterima menaikkan `wb_boards.seq`, menulis `seq` ke baris, lalu disiarkan sebagai event berisi baris lengkap. Klien menerapkan event yang `version`-nya lebih tinggi dari yang dipegang dan mengabaikan yang lebih lama, sehingga urutan sampai yang acak tidak mengembalikan keadaan lama.

### Teks bersamaan: kunci lunak atau CRDT

**Pilihan a: kunci lunak per kartu saat mengetik.**

- Cara kerja: membuka editor kartu mengirim `lock`. Server memberikan kunci bila kartu itu bebas, dan menyiarkan "Nala sedang mengedit". Selama memegang kunci, editor mengirim draf (dokumen JSON) lewat WebSocket paling sering setiap 150 ms; server memvalidasi draf dengan validator yang sama dengan REST lalu meneruskannya, jadi orang lain melihat teksnya berubah langsung. Isi disimpan lewat REST setiap 2 detik selama mengetik dan saat editor ditutup.
- Kunci lepas saat: editor ditutup (`unlock`), tab ditutup atau koneksi putus (hub melepas saat koneksi berakhir), atau tidak ada detak 30 detik (koneksi setengah mati). Sebelum melepas karena putus, hub tidak menyimpan apa pun sendiri: yang tersimpan hanya yang sudah lewat REST, maksimal 2 detik tertinggal. Saat tersambung lagi, klien mengirim sisa isinya dengan `base_content_version`, dan aturan 409 di atas menjaga hasilnya.
- Orang lain yang mencoba mengedit kartu terkunci melihat editor hanya-baca dengan bingkai warna pemegang, "Nala sedang mengedit", dan teks yang ikut berubah. Mereka tetap bisa memindah, mengubah ukuran, mewarnai, dan berkomentar, karena field itu digabung terpisah.
- Biaya: kecil. Hub menyimpan peta kartu ke pemegang kunci di memori, tanpa tabel baru. Validasi draf memakai fungsi yang sudah ada untuk REST. Dampak ke Go backend: satu paket hub dan beberapa jenis pesan, sekitar beberapa ratus baris dengan test.
- Risiko: dua orang tidak bisa mengetik di kartu yang sama bersamaan. Di tim sekitar 10 orang dengan kartu catatan pendek, ini jarang dibutuhkan, dan pemegang terlihat jelas sehingga orang bisa menunggu atau menulis kartu sendiri. Risiko kedua: koneksi pemegang setengah mati menahan kartu sampai 30 detik.

**Pilihan b: CRDT (Yjs) untuk teks kartu.**

- Cara kerja: setiap kartu catatan menjadi dokumen Yjs. TipTap memakai ekstensi kolaborasi (`y-prosemirror`), dan pembaruan biner Yjs dikirim antar klien lewat server. Beberapa orang bisa mengetik di kartu yang sama, dengan kursor teks masing-masing, seperti Google Docs.
- Dampak ke Go backend: tidak ada implementasi Yjs di Go yang matang. Pilihannya (1) server Node terpisah (mis. Hocuspocus atau `y-websocket`) sebagai container baru, bahasa kedua di backend, dengan autentikasi sesi, Origin, dan Admin hanya baca yang harus ditulis ulang di sana; atau (2) Go sebagai relay buta yang menyimpan biner Yjs tanpa memahaminya. Dengan (2), server tidak bisa memvalidasi isi (skema, `href` berbahaya, ukuran), tidak bisa membuat `plain` untuk pencarian, dan riwayat aktivitas tidak tahu isinya, padahal aturan proyek ini mewajibkan semua itu di backend. Validasi harus dipindah ke klien atau ke proses Node.
- Biaya: dependensi baru `yjs`, `y-prosemirror`, `@tiptap/extension-collaboration`, dan `y-protocols` (MIT), kompaksi dokumen Yjs yang terus tumbuh, migrasi format bila Yjs berubah, dan penyimpanan dua bentuk isi (biner Yjs untuk sinkron dan JSON untuk validasi, pencarian, ekspor) yang harus dijaga tetap sama. Test konflik menjadi jauh lebih sulit.
- Risiko: perawatan bergantung pada satu orang yang paham CRDT; galat sinkron biner sulit dilacak dari log; dan aturan "server sumber kebenaran" melemah karena isi yang tersimpan adalah hasil gabungan di klien.

**Rekomendasi: pilihan a, kunci lunak.** Pilihan a memenuhi semua yang diminta Manajer (terlihat siapa mengedit, teks berubah langsung di layar orang lain, tidak ada yang hilang atau tertimpa diam-diam) dengan satu backend Go, validasi tetap di server, dan pola yang sama dengan task. Pilihan b hanya menambah satu kemampuan, mengetik bersamaan di kartu yang sama, dengan biaya container dan bahasa kedua. Bila setelah W2 tim ternyata sering butuh mengetik bersamaan di satu kartu, CRDT bisa ditambahkan khusus untuk kartu catatan tanpa mengubah model kartu lain, karena isi kartu sudah terpisah di `content`.

### Undo dan redo

- Hanya untuk aksi sendiri, per tab, maksimal 100 langkah, hilang saat pindah papan. Setiap aksi menyimpan patch kebalikannya beserta versi field yang dihasilkan aksi itu.
- Undo mengirim patch kebalikan sebagai perubahan biasa, hanya untuk field yang nilainya masih hasil aksi sendiri. Field yang sudah diubah orang lain sesudahnya dilewati, dan klien menampilkan "Sebagian langkah tidak dibatalkan karena sudah diubah Nala." Jadi undo tidak pernah membatalkan perubahan orang lain.
- Undo hapus berarti pulihkan batch dari Tempat sampah; undo pindah antar papan berarti pindah kembali.
- Undo di dalam editor teks memakai riwayat ProseMirror selama editor terbuka; setelah ditutup, satu sesi edit menjadi satu langkah undo papan.

### Alternatif lain yang ditolak

- Satu dokumen JSON per papan: setiap seret kecil menulis ulang 500 kartu, konflik terjadi antar orang yang mengubah kartu berbeda, dan pencarian butuh ekstraksi ulang.
- Kunci keras per papan (hanya satu orang yang boleh mengubah papan): bertentangan dengan sasaran realtime.

## 3. Kolaborasi langsung

**Keputusan: WebSocket untuk kehadiran, kursor, pilihan, gerakan seret, kunci, draf teks, dan siaran perubahan. Perubahan data tetap lewat REST.**

Kenapa WebSocket, bukan SSE: kursor dan gerakan seret 20 sampai 30 pembaruan per detik adalah lalu lintas dua arah yang padat. Dengan SSE, setiap gerak menjadi POST terpisah yang melewati Cloudflare, nginx, middleware sesi, dan query sesi ke database. SSE tetap menjadi cadangan untuk siaran saja (tanpa kursor dan gerakan langsung) bila WebSocket bermasalah di Cloudflare.

Kenapa perubahan data lewat REST, bukan pesan WebSocket: middleware yang sudah ada (`sameOrigin`, `requireSession`, `requirePasswordChanged`, peran), `apperr.Invalid`, 409 versi, audit, dan test DB serta curl langsung bisa dipakai.

### Kehadiran

- Daftar avatar orang yang sedang membuka papan di pojok kanan atas toolbar, memakai komponen avatar yang sudah ada. Lebih dari 5 orang: 5 avatar plus "+3". Avatar bisa difokus dan menyebut nama, status ("sedang mengedit Brief Desember", "diam 4 menit"), dan tombol "Ikuti" untuk memindah viewport ke posisi kursor orang itu.
- Kursor tiap orang: panah kecil dengan label nama, di koordinat dunia (ikut geser dan zoom). Dikirim maksimal 20 kali per detik, dan klien penerima menginterpolasi supaya halus.
- Warna konsisten per orang: dipilih dari palet 8 warna kolaborasi berdasarkan `user_id mod 8`, jadi warna yang sama di semua papan dan sesi. Palet ini token baru `--color-collab-1` sampai `--color-collab-8` untuk terang dan gelap, karena token status yang ada hanya empat warna latar pucat yang tidak cukup berbeda untuk kursor; setiap warna dihitung minimal 3:1 terhadap `canvas` dan `surface`, dan label nama memakai teks dengan kontras 4.5:1 di atas warnanya. Warna tidak pernah satu-satunya penanda: label nama selalu ada.
- Diam: tidak ada gerak penunjuk, ketikan, atau tab tidak terlihat (`visibilitychange`) selama 3 menit, kursor disembunyikan dan avatar tampil redup dengan teks "diam". Kembali aktif begitu ada input.
- Pemberitahuan masuk dan keluar tidak dibacakan pembaca layar satu per satu; jumlah orang di papan ada di label daftar avatar.

### Siapa mengubah apa

- Kartu yang sedang dipilih orang lain diberi bingkai warna orang itu. Kartu yang sedang diedit diberi bingkai tebal plus label "Nala sedang mengedit". Banyak orang memilih kartu yang sama: bingkai milik yang terakhir, label menyebut semuanya ("Nala, Rina").
- Seret dan ubah ukuran: saat mulai, klien mengirim `grab` untuk kartu yang dipegang. Selama bergerak, posisi dan ukuran sementara dikirim 25 kali per detik (dibatasi di klien dan di server), dan orang lain melihat kartu bergerak langsung dengan bingkai warna penyeret. Saat dilepas, satu patch REST menyimpan posisi akhir, lalu siaran `changed` menggantikan posisi sementara.
- Gerakan sementara tidak disimpan. Bila penyeret putus di tengah seret, kartu di layar orang lain kembali ke posisi tersimpan terakhir setelah 3 detik tanpa pembaruan.

### Jalur koneksi

- Endpoint: `GET /api/whiteboard/ws`, satu koneksi per tab. Masuk dan pindah papan lewat pesan `join` tanpa menutup koneksi.
- Pustaka Go: `github.com/coder/websocket` (lisensi ISC), dipin di `go.mod` pada versi terbaru yang sudah berumur dua minggu saat W2 dimulai (per 2026-11-24: v1.8.15). Dipilih karena API berbasis `context` dan tanpa dependensi tambahan. `gorilla/websocket` (BSD-2) juga layak, ditolak hanya karena API-nya lebih rendah tingkatnya.
- nginx frontend: `location /api/whiteboard/ws` sendiri dengan `proxy_http_version 1.1`, `proxy_set_header Upgrade $http_upgrade`, `proxy_set_header Connection "upgrade"`, `Host $host`, header IP yang sama dengan `/api/`, dan `proxy_read_timeout 3600s`.
- Cloudflare Tunnel meneruskan WebSocket tanpa pengaturan tambahan. Cloudflare menutup koneksi yang diam sekitar 100 detik, jadi server mengirim ping setiap 30 detik.
- CSP: `connect-src 'self'` sudah mengizinkan `wss://` ke origin yang sama di browser modern. Diuji di W2 di Chrome, Firefox, dan Safari; bila ada yang menolak, `wss://novoidminds.xyz` ditambahkan eksplisit.

### Autentikasi koneksi

- Cookie `nvm_session` ikut terkirim saat handshake karena origin-nya sama. Handler memakai logika `requireSession` yang sama (token, `sid`, `token_version`, akun aktif, password sudah diganti).
- **Origin wajib dicek di handler WebSocket**, karena `sameOrigin()` melewatkan GET. Tanpa cek ini, situs lain bisa membuka WebSocket dengan cookie korban (cross-site WebSocket hijacking). Aturannya sama: host dari `Origin` harus sama dengan `Host`.
- Sesi diperiksa ulang setiap 60 detik dan saat batas mutlak 9 jam tercapai. Logout, ganti password, reset, atau nonaktif menutup koneksi paling lambat 60 detik kemudian dengan kode 4401, dan klien diarahkan ke halaman masuk.
- Admin boleh terhubung, muncul di daftar avatar, dan menerima siaran. Pesan `lock`, `grab`, `draft`, dan `select` dari Admin ditolak server; kursornya tetap terlihat supaya orang tahu Admin sedang melihat.

### Hub

- Satu hub di memori proses backend, dipetakan per papan, karena backend satu container. Bila suatu hari backend lebih dari satu instance, siaran pindah ke `LISTEN/NOTIFY` Postgres; event-nya sudah mandiri (baris lengkap plus `seq`), jadi klien tidak berubah.
- Setelah transaksi REST commit, handler menyerahkan baris hasilnya ke hub, dan hub menyiarkannya ke semua koneksi di papan itu, termasuk tab lain milik pengirim.
- Batas: 32 koneksi per pengguna, pesan masuk maksimal 64 KB untuk draf dan 1 KB untuk yang lain, kursor dan gerakan maksimal 30 pesan per detik per koneksi (lebihnya dibuang), antrean keluar per koneksi 512 pesan. Koneksi yang antreannya penuh ditutup dan menyambung ulang dengan sinkron `seq`, supaya satu klien lambat tidak menahan hub.

### Pesan

- Dari klien: `join` (papan, `last_seq`, `tab_id`), `cursor`, `select` (daftar id kartu), `grab` dan `move` (kartu, posisi dan ukuran sementara), `release`, `lock`, `draft`, `unlock`, `idle`, `active`, `ping`.
- Dari server: `presence` (orang di papan: id, nama, avatar, warna, status diam, jumlah tab), `cursor`, `select`, `grab_granted`, `grab_denied` (pemegang), `move`, `release`, `lock_granted`, `lock_denied` (pemegang), `locked` dan `unlocked` (untuk yang lain), `draft`, `changed` (baris lengkap dan `seq`), `deleted` (id, `deleted_batch`, oleh siapa, `seq`), `restored`, `board_moved`, `board_deleted`, `resync` (selisih terlalu besar, muat ulang).

### Penanda simpan dan koneksi

Satu penanda di toolbar, dengan teks (bukan ikon saja) dan `aria-live="polite"`:

- "Tersimpan": antrean kosong dan semua patch sudah diterima server.
- "Menyimpan...": ada patch yang sedang dikirim.
- "Terputus, menyambung ulang": WebSocket putus atau REST gagal karena jaringan. Penanda ini tetap tampil, bukan toast yang hilang.
- "Gagal disimpan": server menolak patch (400, 403, 409 yang tidak bisa diselesaikan). Klik membuka penjelasan dan pilihan (salin ke kartu baru, buang).

### Kasus tepi

**Dua orang menyeret kartu yang sama bersamaan.** `grab` pertama yang sampai ke server menang; server membalas `grab_denied` ke yang kedua, dan seret lokal yang kedua dibatalkan dalam satu frame dengan label "Nala sedang memindahkan". Karena jeda round trip sekitar 50 sampai 100 ms, yang kedua mungkin sempat melihat kartunya bergeser sedikit lalu kembali; itu disengaja dan jelas. Bila WebSocket putus sehingga `grab` tidak bisa diminta, seret tetap jalan secara lokal, dan patch posisi digabung last-write-wins dengan label "Dipindah <nama>" bagi yang kalah. Seret banyak kartu meminta `grab` untuk semuanya sekaligus; bila sebagian dipegang orang lain, hanya yang bebas yang ikut bergerak.

**Kartu yang sedang diedit dihapus orang lain.** Hapus tetap boleh (semua orang boleh menghapus), tetapi:

1. Sebelum memindah kartu ke Tempat sampah, server meminta draf terakhir pemegang kunci lewat hub dan menunggu paling lama 1 detik. Draf yang datang disimpan sebagai isi kartu dalam transaksi hapus yang sama, jadi ketikan terakhir ikut masuk Tempat sampah.
2. Pemegang menerima `deleted` berisi nama penghapus, editornya ditutup, dan muncul dialog: "Kartu ini dipindah ke Tempat sampah oleh Rina. Ketikan terakhirmu ikut tersimpan." dengan tombol "Pulihkan kartu" (memulihkan batch itu bila masih ada di Tempat sampah) dan "Tutup".
3. Bila draf tidak datang dalam 1 detik, klien pemegang yang menerima `deleted` masih menyimpan teksnya di memori dan dialog menawarkan "Pulihkan kartu" lalu menyimpan teks itu, atau "Salin teks ke kartu baru".
4. Penghapus diberi tahu sebelum mengonfirmasi: "Nala sedang mengedit kartu ini. Tetap hapus?"

**Koneksi putus sebentar.**

1. Penanda berubah menjadi "Terputus, menyambung ulang". Kursor dan avatar orang lain disembunyikan, kartu yang sedang diedit tetap bisa diketik, dan kunci tetap dianggap milik sendiri secara lokal.
2. Perubahan baru masuk antrean memori (bukan localStorage). Patch REST yang gagal karena jaringan diulang dari antrean dengan urutan yang sama.
3. Reconnect dengan jeda 1, 2, 4, 8, 16, lalu 30 detik ditambah acak sampai 20%, supaya 10 orang tidak menyambung serentak setelah backend di-restart oleh `rilis.sh`.
4. Setelah tersambung: kirim `join` dengan `last_seq`, ajukan ulang kunci kartu yang sedang diedit, kirim antrean, lalu terapkan semua baris dengan `seq` lebih besar dari server, termasuk yang dihapus. Selisih lebih dari 2000 baris berarti `resync` dan papan dimuat ulang utuh setelah antrean terkirim.
5. Bila kunci sudah diambil orang lain selama putus, isi yang belum tersimpan dikirim dengan `base_content_version` lama, ditolak 409, lalu disimpan sebagai kartu salinan dengan pesan yang menyebut pemegang baru.
6. Penanda kembali "Tersimpan" hanya setelah antrean kosong.
7. Menutup tab saat antrean belum kosong memunculkan peringatan bawaan browser (`beforeunload`).

**Undo dan redo.** Lihat bagian 2: hanya aksi sendiri, field yang sudah diubah orang lain dilewati dengan pesan.

**Dua tab dari akun yang sama.** Setiap tab punya `tab_id` acak dan koneksi sendiri.

- Kehadiran: satu avatar per orang, dengan jumlah tab di labelnya. Kursor hanya dari tab yang terakhir aktif.
- Kunci dimiliki pasangan pengguna dan tab, bukan pengguna saja. Tab kedua melihat "Kamu sedang mengedit kartu ini di tab lain", editor hanya-baca dengan teks yang ikut berubah, dan tombol "Edit di sini" yang memindahkan kunci (tab pertama menyimpan dulu, lalu editornya ditutup).
- Undo per tab: tab kedua tidak bisa membatalkan aksi tab pertama.
- Siaran `changed` ikut dikirim ke tab lain milik pengirim, jadi kedua tab selalu sama.
- Batas 32 koneksi per pengguna mencegah tab yang tidak sengaja terbuka puluhan kali.

**Kasus lain yang ditangani.**

- Papan yang sedang dibuka dihapus atau dipindah: `board_deleted` menampilkan "Papan ini dipindah ke Tempat sampah oleh Rina" dengan tombol "Pulihkan papan" dan "Ke Beranda Whiteboard"; `board_moved` hanya memperbarui breadcrumb.
- Kartu dipindah ke papan lain saat orang lain sedang mengeditnya: diperlakukan seperti hapus untuk papan asal (dengan nama papan tujuan dan tombol "Buka papan tujuan"), draf disimpan dengan cara yang sama.
- Sesi berakhir saat mengedit: koneksi ditutup 4401, antrean yang belum terkirim ditampilkan sebagai teks yang bisa disalin sebelum diarahkan ke halaman masuk.

## 4. Frontend

### Render kanvas

**Keputusan: DOM dengan CSS transform.** Satu elemen dunia dengan `transform: translate(...) scale(...)`, kartu sebagai elemen absolut di dalamnya, penghubung di satu lapisan SVG di ruang dunia yang sama.

- Teks berformat, pilih teks, tempel, input IME, pembaca layar, dan focus semuanya gratis dari DOM. Di canvas 2D semuanya harus dibuat ulang.
- Binding `:style` Vue dan `element.style` memakai CSSOM, yang tidak diatur `style-src`, jadi CSP tidak perlu `unsafe-inline` (sudah tercatat di komentar `nginx-security-headers.conf`). Yang dilarang adalah atribut `style="..."` dari HTML hasil parse dan tag `<style>` sisipan.
- Grid digambar dengan `background-image` di CSS berkas (gradien), bukan elemen per garis.
- Minimap: SVG kecil berisi persegi per kartu, `aria-hidden`, karena tampilan Daftar yang menjadi alternatif aksesibelnya.

Ditolak: canvas 2D (Konva, Fabric). Lebih cepat di atas ribuan objek, tetapi 500 kartu masih nyaman di DOM, dan canvas 2D membuat aksesibilitas, editor teks, dan kursor teks harus dibangun ulang di atas lapisan DOM tersembunyi. Juga ditolak: pustaka papan lengkap (tldraw, Excalidraw). Keduanya React, lisensi tldraw tidak bebas untuk produksi, dan keduanya membawa gaya dan model data sendiri yang bertabrakan dengan tema NVM dan aturan backend.

### Editor teks berformat

**Keputusan: TipTap 3 di atas ProseMirror (MIT), hanya untuk kartu yang sedang diedit.** Kartu lain dirender dari JSON lewat komponen Vue sendiri (render function per jenis node), bukan dengan menjalankan editor.

Ekstensi yang dipakai: paragraf, heading 1 sampai 3, tebal, miring, garis bawah, coret, kode inline, daftar berbutir dan bernomor, kutipan, tautan, hard break. Tanpa tabel, gambar di dalam teks, HTML mentah, atau gaya warna bebas.

Kecocokan dengan CSP:

- `injectCSS: false` wajib. TipTap bawaannya menyisipkan tag `<style>`, yang diblokir `style-src 'self'`. Gaya editor yang dibutuhkan disalin ke CSS proyek.
- ProseMirror menulis gaya lewat `element.style` (CSSOM), jadi aman.
- HTML yang ditempel diparse ProseMirror ke skema terbatas; atribut `style` dan elemen di luar skema dibuang sebelum menyentuh dokumen.

Ditolak: editor `contenteditable` buatan sendiri. Kelihatannya kecil, tetapi seleksi, undo, tempel dari Word dan Google Docs, IME, dan perilaku ponsel adalah sumber bug yang lama dibereskan. Juga ditolak: Quill (format Delta sendiri, perawatan lambat) dan Lexical (React lebih dulu).

### Dependensi baru

Hanya dimuat di rute Whiteboard (dynamic import), jadi halaman lain tidak bertambah berat.

| Paket | Lisensi | Versi yang dipin | Untuk |
|---|---|---|---|
| `@tiptap/core`, `@tiptap/pm`, `@tiptap/vue-3`, `@tiptap/starter-kit`, `@tiptap/extension-underline`, `@tiptap/extension-link` | MIT | `3.33.0` persis, tanpa `^` (terbit 2026-11-05; 3.34 dan 3.35 belum berumur dua minggu per 2026-11-24). Semua paket `@tiptap/*` harus versi yang sama. | Editor kartu catatan |
| `html-to-image` | MIT | `1.11.13` persis | Ekspor PNG |
| `github.com/coder/websocket` (Go) | ISC | v1.8.x terbaru yang berumur dua minggu saat W2 | WebSocket |

Dicek ulang saat pemasangan: versi, lisensi, dan ukuran bundle (target tambahan maksimal 150 KB gzip di rute Whiteboard).

Tidak ditambah: pustaka pan dan zoom, pustaka drag (Pointer Events cukup), pustaka PDF, pustaka state. PDF ditulis sendiri: satu halaman berisi satu gambar JPEG (`DCTDecode`) dari kanvas ekspor, sekitar 60 baris dengan unit test, tanpa jsPDF.

### Struktur kode

- `frontend/src/pages/WhiteboardHomePage.vue` (folder dan papan), `WhiteboardBoardPage.vue` (kanvas), `WhiteboardTrashPage.vue`.
- `frontend/src/whiteboard/`: `store.ts` (Pinia, kartu dalam `Map` dengan `shallowRef`), `viewport.ts` (pan, zoom, konversi koordinat), `socket.ts` (koneksi, antrean, reconnect), `history.ts` (undo dan redo), `cards/*.vue` per jenis, `richtext/` (render JSON dan editor).
- Backend: paket `internal/whiteboard` (service, validasi isi, tempat sampah, hub), `internal/linkpreview` (pengambil aman), handler di `internal/httpapi/whiteboard_handlers.go`.

## 5. Keamanan

### Hak akses

- Grup rute baru di `router.go`: baca di `app` (semua peran yang sudah ganti password, termasuk Admin), tulis di grup `requireRole(Staf, Manajer)`. Admin hanya baca, sama dengan Timeline dan Important Notes.
- Semua Staf dan Manajer bisa membuat, mengubah, dan menghapus semua folder, papan, dan kartu. Tidak ada papan pribadi.
- Komentar: menunggu Manajer (bagian 10). Bawaan sampai ada jawaban: penulis mengubah dan menghapus komentarnya sendiri, Manajer bisa menghapus komentar siapa pun.

### Pratinjau tautan (SSRF)

Diambil oleh server, karena CSP `img-src` tidak mengizinkan gambar dari situs luar dan browser tidak bisa membaca HTML situs lain. Paket `internal/linkpreview`:

1. URL hanya `http` atau `https`, port 80 atau 443, tanpa userinfo, panjang maksimal 2048. Host berupa IP literal diperiksa dengan aturan yang sama.
2. Dialer sendiri dengan `net.Dialer.Control` yang memeriksa **alamat IP yang benar-benar disambungi**, bukan hasil resolve terpisah, supaya DNS rebinding tidak bisa lolos. Ditolak: loopback, privat (10/8, 172.16/12, 192.168/16, termasuk LAN kantor 192.168.68.0/24), link-local (169.254/16, termasuk metadata 169.254.169.254, dan fe80::/10), CGNAT 100.64/10, 0.0.0.0/8, multicast, broadcast, ULA fc00::/7, IPv4-mapped dan NAT64 yang menunjuk alamat terlarang, serta alamat dokumentasi.
3. Redirect maksimal 3. Setiap lompatan melewati dialer yang sama, jadi redirect ke IP privat ditolak dengan sendirinya. Redirect ke skema selain http dan https ditolak.
4. Waktu total 5 detik per halaman. Body dibaca maksimal 512 KB dan hanya bila `Content-Type` HTML. Hanya `<title>`, `og:title`, `og:description`, `og:site_name`, dan `og:image` yang diambil, dengan parser HTML `golang.org/x/net/html` (sudah ada di `go.sum` sebagai dependensi tidak langsung).
5. `og:image` diambil dengan klien yang sama, maksimal 5 MB, harus PNG, JPEG, WebP, atau GIF menurut isi, lalu di-encode ulang menjadi JPEG dengan `imaging` (sudah dipakai untuk avatar) sebelum disimpan sebagai `wb_files`. Gambar asli tidak pernah disajikan.
6. Tanpa cookie, tanpa header dari permintaan pengguna, tanpa variabel proxy dari env. User-Agent umum tanpa nama atau kontak tim.
7. Batas 30 pratinjau per pengguna per jam. Hasil disimpan di `wb_link_previews` 7 hari, termasuk hasil `blocked` dan `failed`, supaya URL yang sama tidak diambil berulang.
8. Kegagalan tidak menggagalkan kartu: kartu tautan tetap tersimpan dengan URL saja.
9. Unit test dengan server lokal yang mencoba setiap kelas alamat terlarang, redirect ke 127.0.0.1, dan body besar, tanpa memanggil situs luar.

### Teks berformat dan XSS

- Disimpan sebagai JSON ProseMirror, bukan HTML. Go memvalidasi terhadap daftar putih: jenis node dan mark yang diizinkan, kedalaman maksimal 8, ukuran maksimal 64 KB, teks tanpa karakter kontrol. `href` tautan hanya `http`, `https`, atau `mailto`; `javascript:`, `data:`, dan lainnya ditolak 400.
- Dirender oleh komponen Vue dari JSON, tanpa `v-html` di seluruh fitur Whiteboard. Tautan dirender `target="_blank" rel="noopener noreferrer nofollow"`.
- Judul, teks checklist, komentar, dan nama berkas adalah teks polos yang di-escape Vue.
- `plain` dibuat di Go dari JSON yang sudah lolos validasi, jadi pencarian tidak pernah mengembalikan markup.

### Berkas

- Batas 50 MB, sama dengan lampiran task. Tidak ada alasan berbeda: gambar papan jarang besar, dan video tidak termasuk kebutuhan. Rute unggah Whiteboard memakai `client_max_body_size 51m` yang sudah berlaku di `/api/`.
- Daftar izin: daftar `attachment.Detect` untuk berkas, ditambah WebP dan GIF untuk gambar seperti `DetectActivity`. **SVG ditolak** sebagai gambar karena bisa berisi skrip.
- Berkas asli selalu dikirim dengan `Content-Disposition: attachment` dan `nosniff`, sama dengan lampiran task. Untuk tampil di kanvas, server membuat thumbnail JPEG (lebar maksimal 1200 px) saat unggah dengan `imaging`. Thumbnail itu hasil encode ulang, jadi boleh disajikan `inline` dengan `Content-Type: image/jpeg`. GIF animasi tampil sebagai gambar diam di kanvas.
- Tempel dan seret memakai `useUploadQueue`, `FileDropTarget`, dan `pasteFiles.ts` dari revisi 36.

## 6. Penyimpanan berkas dan gambar

- Memakai `storage.Store` yang sama, folder `attachments/`, nama berkas = `wb_files.id`. Thumbnail di `attachments/<2 huruf>/<id>.thumb.jpg`; pembersih berkas yatim di `maintenance` disesuaikan supaya mengenali akhiran itu (tanpa penyesuaian, berkas thumbnail dilaporkan sebagai "Unknown", tidak dihapus, tetapi juga tidak pernah dibersihkan).
- Ikut `deploy/backup.sh` dengan sendirinya: dump database dan snapshot storage sudah mencakup tabel dan folder ini.
- Salinan ke Google Drive malam (revisi 25): **bawaan tidak ikut** sampai Manajer memutuskan. Alasannya: struktur folder Drive saat ini mengikuti task dan Aktivitas, sedangkan papan bisa dipindah dan diganti nama kapan saja, sehingga path di Drive ikut berubah dan salinan lama menumpuk. Bila Manajer ingin ikut, rancangannya: `Whiteboard/<id papan pendek> <judul>/<nama berkas>`, ditulis ulang hanya bila berkas baru.
- Duplikat kartu atau papan dari template membuat baris `wb_files` baru dengan hardlink ke berkas yang sama (satu filesystem), bukan salinan byte.

## 7. Performa

Target: papan 500 kartu (campuran catatan, checklist, 50 gambar, 100 penghubung) dimuat dalam 2 detik di laptop kantor lewat domain publik, dan geser, zoom, serta seret kartu tetap sekitar 60 fps di laptop dan tetap bisa dipakai di ponsel kelas menengah.

Strategi muat:

- Satu `GET /api/whiteboard/boards/:id` mengembalikan papan, semua kartu hidup, penghubung, papan anak, dan orang yang terlibat. 500 kartu sekitar 400 KB JSON, sekitar 80 KB setelah gzip (nginx sudah mengompres `application/json`).
- Komentar dan riwayat aktivitas dimuat saat panelnya dibuka.
- Thumbnail gambar dimuat hanya untuk kartu yang terlihat, dengan `loading="lazy"` dan URL same-origin, sehingga browser yang mengelola cache.

Strategi render:

- Culling: hanya kartu yang beririsan dengan viewport plus satu layar margin yang di-mount. Sisanya tidak punya elemen DOM. Indeks spasial sederhana (grid bucket 512 px) dihitung ulang saat kartu pindah.
- Saat geser dan zoom, hanya `transform` elemen dunia yang berubah, lewat `requestAnimationFrame`. Culling dihitung ulang setelah gerak berhenti 100 ms dan setiap 250 ms selama bergerak.
- Saat seret kartu, posisi ditulis langsung ke `element.style.transform` tanpa melewati reaktivitas Vue; store diperbarui sekali saat dilepas, lalu satu patch dikirim. Kursor (20 per detik) dan posisi seret (25 per detik) disiarkan ke orang lain sebagai pesan WebSocket sementara, bukan disimpan. Gerakan orang lain juga ditulis langsung ke elemen kartunya, tanpa reaktivitas Vue.
- Teks berformat kartu yang tidak diedit dirender dari JSON sekali dan di-memo per `content_version`.
- Di bawah zoom 40%, isi kartu diganti blok judul dan warna saja (level of detail), karena teks tidak terbaca di skala itu.
- Penghubung: satu `<path>` per garis di satu SVG, dihitung ulang hanya untuk garis yang ujungnya bergerak.

Pembuktian: E2E `wb-performa` di W4 mengisi papan 500 kartu lewat API, lalu mengukur waktu muat dan durasi frame selama pan otomatis 3 detik dengan trace Playwright. Batas lulus: muat di bawah 2 detik, p95 frame di bawah 20 ms di 1280 px. Juga `EXPLAIN ANALYZE` muat papan dan pencarian di test DB dengan 5.000 kartu.

## 8. Aksesibilitas

- Kanvas punya `role="application"` dengan `aria-label` "Kanvas papan <judul>" dan petunjuk pintasan yang bisa dibuka (`?`). Kartu adalah elemen fokus dengan roving tabindex: Tab masuk ke kanvas dan ke kartu terpilih, panah berpindah ke kartu terdekat di arah itu, Home dan End ke kartu pertama dan terakhir menurut urutan baca.
- Urutan DOM kartu yang di-mount mengikuti urutan baca (atas ke bawah, kiri ke kanan), bukan urutan pembuatan, supaya pembaca layar tidak melompat-lompat.
- Pindah dengan keyboard: Shift+panah menggeser 8 px (satu sel grid), Shift+Ctrl+panah 64 px. Enter membuka editor, Escape keluar dari editor lalu membatalkan pilihan. Delete memindahkan ke Tempat sampah dengan pengumuman dan tombol Urungkan.
- Pintasan yang direncanakan: Ctrl+A, Ctrl+C, Ctrl+V, Ctrl+D (duplikat), Ctrl+Z, Ctrl+Shift+Z dan Ctrl+Y, Delete, N (catatan baru), Ctrl+0 (pas ke layar), Ctrl+= dan Ctrl+- (zoom), G (grid dan snap). Pintasan satu huruf tidak aktif saat fokus di field teks atau editor.
- Setiap kartu punya label yang bisa dibaca: "Catatan: <awal teks>", "Checklist <judul>, 3 dari 7 selesai", "Gambar <keterangan atau nama berkas>". Kartu yang sedang dikunci orang lain menyebut "sedang diedit <nama>".
- `aria-live="polite"` untuk hasil aksi sendiri ("4 kartu dipindah", "Kartu dipindah ke Tempat sampah. Urungkan tersedia."). Perubahan orang lain tidak dibacakan satu per satu; cukup ringkasan saat ada perubahan di kartu yang sedang difokus.
- **Tampilan Daftar**: tombol di toolbar mengganti kanvas menjadi daftar kartu berurutan, dikelompokkan per kolom dan per wilayah, dengan aksi yang sama (ubah, centang, pindah ke papan lain, hapus). Ini jalur utama untuk pembaca layar dan cadangan bila kanvas sulit dipakai di ponsel kecil.
- Focus ring memakai `focus-visible` dengan outline di koordinat layar, digambar di lapisan overlay di luar elemen yang di-scale, supaya tebalnya tetap 2 px pada zoom berapa pun.
- Warna kartu memakai pasangan token yang sudah ada (`requested`, `progress`, `ready`, `completed`, `tag`, `danger-bg`, dan `surface`), terang dan gelap, sehingga tidak perlu warna baru. Kontras teks di atas setiap latar dihitung di W1 dengan pemeriksa kontras; target 4.5:1. Penghubung dan border kartu minimal 3:1 terhadap `canvas`. Kursor dan bingkai orang lain memakai palet `--color-collab-1` sampai `--color-collab-8` (bagian 3), selalu dengan label nama, jadi tidak bergantung pada warna saja.
- Tap target 44x44 px untuk toolbar dan pegangan ubah ukuran. Pegangan kecil di kanvas diperbesar area sentuhnya tanpa memperbesar gambarnya.
- `prefers-reduced-motion`: animasi pas ke layar dan zoom menjadi lompatan langsung. Tidak ada animasi berulang; kursor orang lain bergerak tanpa transisi.

## 9. Rencana tahap W1 sampai W4

Setiap tahap selesai bila `./scripts/pra-deploy.sh` LOLOS (CI hijau), E2E tahap itu lolos di 360 dan 1280 px, dan keputusan baru tercatat di `docs/sprint.md`.

### W1: fondasi, catatan, dan Tempat sampah

Isi:

- Migrasi `000023_whiteboard`, perubahan `10-grants.sql`, `testdb` diperbarui.
- Beranda Whiteboard: folder dan papan, buat, ganti nama, pindah, breadcrumb, pencarian judul.
- Kanvas: geser, zoom, grid, snap, pas ke layar.
- Kartu `note` (TipTap), `heading`, `checklist`. Warna kartu.
- Seret, ubah ukuran, pilih satu dan banyak (kotak seleksi, Shift+klik, Ctrl+A), hapus.
- Tempat sampah dengan pulihkan per batch, hapus permanen 30 hari di job `maintenance`.
- Riwayat aktivitas per papan, audit log hapus, pulihkan, dan hapus permanen.
- Navbar: Timeline, Task, Aktivitas, Whiteboard, Arsip, lalu sisanya. Ponsel lewat menu Lainnya.
- Ponsel: lihat, geser, zoom (pinch), tambah catatan dan checklist, ubah isi kartu.
- Admin hanya baca.

Test DB: hak per peran (Admin 403 untuk setiap tulis), siklus pohon ditolak, validasi isi per jenis (termasuk `href` berbahaya), 409 isi basi, geometri last-write-wins, batch hapus dan pulihkan (termasuk aturan induk), job hapus permanen (29 hari tetap, 31 hari hilang), audit tercatat, penggabungan riwayat geometri.

E2E: `wb-dasar` (buat folder, papan, catatan, checklist, seret, hapus, pulihkan dari Tempat sampah), `wb-keyboard` (Tab, panah, Shift+panah, Enter, Escape, Delete), `wb-ponsel` (360 px: pinch dan tambah catatan), `wb-admin` (tombol tulis tidak ada, API 403).

Bisa dipakai tim: papan catatan dan checklist bersama tanpa sinkron langsung. Bila dua orang mengubah isi kartu yang sama, yang kedua mendapat salinan, tidak kehilangan ketikan. Perubahan orang lain terlihat setelah muat ulang. Penanda "Tersimpan", "Menyimpan...", dan "Gagal disimpan" sudah ada di W1.

### W2: kolaborasi langsung dan edit lanjutan

Isi:

- WebSocket (`/api/whiteboard/ws`), lokasi nginx, cek Origin, cek ulang sesi.
- Kehadiran: daftar avatar, kursor bernama dengan warna tetap per orang, status diam, tombol Ikuti.
- Siapa mengubah apa: bingkai pilihan dan edit, label nama, gerakan seret dan ubah ukuran langsung, `grab`.
- Kunci lunak dengan draf teks langsung, siaran perubahan, penanda "Terputus, menyambung ulang", antrean offline, sinkron ulang dengan `seq`.
- Semua kasus tepi di bagian 3, termasuk dua tab dari akun yang sama dan hapus kartu yang sedang diedit.
- Undo dan redo milik sendiri, salin dan tempel (termasuk antar papan dan antar tab), duplikat, perataan dan distribusi, semua pintasan, minimap.

Test backend dengan dua sampai tiga klien WebSocket sungguhan (`httptest.Server` plus klien `coder/websocket`, test DB untuk REST):

- Handshake: Origin asing ditolak, tanpa cookie ditolak, sesi dicabut atau akun dinonaktifkan menutup koneksi dengan 4401, Admin tidak bisa `lock`, `grab`, `draft`, atau `select`.
- Kehadiran: tiga klien `join`, semua menerima `presence` yang sama; satu putus, dua sisanya menerima `presence` tanpa dia; dua tab akun yang sama tampil satu orang dengan dua tab.
- Kunci: dua klien `lock` kartu yang sama bersamaan (goroutine, diulang 100 kali dengan `-race`), tepat satu `lock_granted`; draf hanya diteruskan dari pemegang; kunci lepas saat koneksi pemegang ditutup dan saat 30 detik tanpa detak (jam palsu).
- Seret: dua klien `grab` kartu yang sama bersamaan, tepat satu menang; `move` di atas 30 per detik dibuang.
- Hapus saat diedit: klien A memegang kunci dan punya draf, klien B menghapus lewat REST; draf A ikut tersimpan di baris yang masuk Tempat sampah, A menerima `deleted` dengan nama B, dan pulihkan batch mengembalikan isi draf itu.
- Urutan: siaran `changed` hanya terkirim setelah commit; patch yang ditolak (403, 409) tidak pernah disiarkan; event dengan versi lebih lama diabaikan (unit test store di frontend).
- Putus dan sambung: klien putus, klien lain membuat, mengubah, dan menghapus kartu, klien pertama `join` dengan `last_seq` dan menerima tepat perubahan itu termasuk yang dihapus; selisih di atas 2000 menghasilkan `resync`.
- Hub: antrean penuh menutup koneksi tanpa menahan klien lain.
- Riwayat aktivitas: setiap patch tersimpan tercatat dengan pelaku yang benar, termasuk patch dari dua klien yang saling menimpa.

E2E dengan beberapa konteks browser (dua atau tiga akun, ditambah konteks kedua untuk akun yang sama), di 360 dan 1280 px:

- `wb-hadir`: tiga orang membuka papan; avatar dan kursor bernama muncul di semua layar; satu orang diam 3 menit (jam dipercepat lewat `page.clock`) dan tampil redup.
- `wb-langsung`: A mengetik, B melihat teks berubah dan label "A sedang mengedit", B tidak bisa mengetik di kartu itu tetapi bisa mengganti warnanya, dan kedua perubahan tersimpan.
- `wb-seret`: A menyeret, B melihat kartu bergerak sebelum dilepas; A dan B menyeret kartu yang sama, satu dibatalkan dengan label.
- `wb-hapus-saat-edit`: A mengedit, B menghapus, A melihat dialog, menekan "Pulihkan kartu", dan isi terakhirnya ada.
- `wb-putus`: A diputus dengan `context.setOffline(true)`, penanda "Terputus, menyambung ulang" tampil, A membuat dua kartu, B mengubah kartu lain, A tersambung lagi, penanda menjadi "Tersimpan", dan kedua layar sama.
- `wb-undo`: A memindah kartu, B mengganti warnanya, A undo: posisi kembali, warna B tetap, dan pesan "sudah diubah" tampil untuk field yang dilewati.
- `wb-dua-tab`: akun yang sama di dua tab, kunci berpindah lewat "Edit di sini", undo tab kedua tidak membatalkan aksi tab pertama.

Bisa dipakai tim: kerja bersama di papan yang sama secara langsung, dengan semua orang terlihat dan tanpa perubahan yang hilang.

### W3: berkas, tautan, kolom, penghubung, komentar

Isi:

- Migrasi `000024_whiteboard_media`, `ListAttachmentFileIDs` diperbarui, grants ulang.
- Kartu `image` dan `file` (tempel, seret, pilih berkas, thumbnail), kartu `link` dengan pratinjau aman.
- Kartu `column` (menumpuk kartu, urutkan dengan seret dan keyboard), kartu `board` (papan di dalam papan).
- Pindah kartu antar papan (menu dan seret ke kartu papan; penghubung ke kartu yang tertinggal dihapus ke Tempat sampah dalam batch yang sama).
- Garis dan panah penghubung dengan label.
- Komentar di kartu.

Test: unit `linkpreview` (setiap kelas IP terlarang, redirect ke privat, DNS yang menunjuk 127.0.0.1, ukuran, waktu, tipe), test DB berkas yatim tidak menghapus berkas Whiteboard, test unggah (SVG ditolak, 50 MB), test pindah antar papan.

E2E: `wb-berkas`, `wb-tautan` (server uji lokal, bukan situs luar), `wb-kolom`, `wb-penghubung`, `wb-komentar`.

Bisa dipakai tim: moodboard dan rencana konten lengkap dengan gambar, referensi tautan, dan diskusi.

### W4: template, ekspor, pencarian isi, poles

Isi:

- Template papan: tandai papan sebagai template, buat papan dari template (menyalin kartu, penghubung, dan berkas lewat hardlink).
- Ekspor papan atau pilihan ke PNG dan PDF. Ukuran ekspor dibatasi supaya tidak melewati batas kanvas browser (sekitar 16 juta piksel di Safari ponsel); papan yang lebih besar diekspor dengan skala diperkecil dan keterangan itu ditampilkan.
- Pencarian isi kartu di semua papan (judul papan, teks kartu, judul checklist, keterangan berkas, judul pratinjau tautan).
- Keputusan ekspor Drive diterapkan.
- Tampilan Daftar, level of detail, uji performa 500 kartu, audit aksesibilitas dan kontras.

Test: `EXPLAIN ANALYZE` pencarian dan muat papan dengan 5.000 kartu, unit test penulis PDF (struktur berkas valid, dibuka `pdfinfo` di CI bila tersedia).

E2E: `wb-template`, `wb-ekspor` (berkas PNG dan PDF terunduh dan berukuran wajar), `wb-cari`, `wb-performa`, `wb-daftar`.

Bisa dipakai tim: fitur lengkap sesuai kebutuhan Manajer.

## 10. Risiko dan pertanyaan untuk Manajer

### Risiko

- **Ukuran pekerjaan.** Setara Milanote berarti empat tahap yang masing-masing sebesar beberapa revisi biasa. Risiko terbesarnya adalah W2 (koneksi langsung) dan W3 (penghubung dan kolom), karena interaksinya banyak dan sulit diuji. Rencana ini sengaja membuat W1 sudah berguna tanpa W2.
- **Semua orang bisa menghapus semuanya.** Tempat sampah 30 hari, riwayat per papan, audit log, dan backup harian mengurangi dampaknya, tetapi penghapusan folder besar oleh orang yang salah tetap mengganggu sampai seseorang memulihkannya.
- **WebSocket lewat Cloudflare.** Didukung, tetapi koneksi diputus saat Cloudflare memperbarui edge dan saat `rilis.sh` me-restart backend. Reconnect dan sinkron `seq` harus kuat; cadangannya SSE tanpa kursor.
- **Kunci lunak bukan Google Docs.** Dua orang tidak bisa mengetik di kartu yang sama bersamaan. Bila Manajer menganggap itu wajib, CRDT untuk kartu catatan menjadi tahap tambahan sesudah W2 dengan biaya di bagian 2.
- **Lalu lintas realtime.** Sepuluh orang dengan kursor 20 per detik dan seret 25 per detik masih kecil untuk satu proses Go, tetapi melewati Cloudflare dan nginx. Batas laju di klien dan server wajib sejak W2.
- **Ponsel.** Pinch, seret, dan editor teks di Safari iOS sering berbeda dari desktop. Uji di ponsel sungguhan masuk S13-08.
- **Disk VM.** Gambar papan bisa tumbuh cepat. Thumbnail menambah sekitar 10 sampai 20% per gambar. Pemantauan disk sudah ada di halaman Status.
- **Bundle.** TipTap menambah sekitar 100 KB gzip, hanya di rute Whiteboard.
- **Privasi pratinjau tautan.** Situs yang ditautkan melihat permintaan dari IP VM kantor.
- **Riwayat aktivitas tumbuh terus** sampai ada keputusan masa simpan.

### Pertanyaan untuk Manajer

Masuk ke "Menunggu jawaban Manajer" di `docs/sprint.md` saat W1 dimulai. Pilihan bawaan di kurung, dipakai sampai ada jawaban.

1. Komentar: siapa yang boleh mengubah dan menghapus komentar orang lain? (Penulis untuk komentarnya sendiri, Manajer untuk semua.)
2. Apakah berkas Whiteboard ikut salinan Google Drive malam? (Tidak.)
3. Berapa lama riwayat aktivitas papan disimpan? (Disimpan tanpa batas sampai diputuskan.)
4. Template: boleh dibuat dan diubah semua orang, atau hanya Manajer? (Semua Staf dan Manajer, sesuai aturan umum Whiteboard.)
5. Admin benar-benar hanya baca, termasuk tidak bisa memulihkan dari Tempat sampah? (Ya, hanya baca.)
6. Apakah perlu papan atau folder pribadi atau per divisi? (Tidak; semua orang melihat semua, sesuai kebutuhan.)
7. Apakah pratinjau tautan boleh mengambil halaman situs luar dari server kantor? (Ya, dengan batas di bagian 5. Bila tidak, kartu tautan hanya menampilkan URL dan nama domain.)
8. Apakah 30 hari di Tempat sampah sudah pas, dan perlu tombol "hapus permanen sekarang" untuk Manajer? (30 hari, tanpa tombol.)
9. Apakah perlu sebutan (@nama) dan notifikasi di komentar? (Tidak di W1 sampai W4.)
10. Apakah kunci lunak (satu orang mengetik per kartu, yang lain melihat teksnya berubah langsung) cukup, atau mengetik bersamaan di satu kartu seperti Google Docs wajib? (Kunci lunak.)

## Riwayat keputusan

- 2026-11-24: rancangan W0 ditulis.
- 2026-11-24: tambahan wajib Manajer untuk realtime penuh: kehadiran, siapa mengubah apa, gerakan seret langsung, perbandingan kunci lunak dan CRDT (rekomendasi kunci lunak), kasus tepi, server sumber kebenaran, dan rencana uji dengan beberapa klien. Bagian 2, 3, 7, 8, 9, dan 10 disesuaikan.
