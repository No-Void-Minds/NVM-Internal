# Rencana Sprint Sistem Internal NVM

Acuan: SRS Sistem Internal NVM v1.0, ERD `nvm_internal.dbml`, layar Stitch, dan DESIGN.md.
Stack: Go, PostgreSQL, Vue 3 dengan Tailwind CSS, Docker di VM Ubuntu (Hyper-V) pada PC kantor.
Revisi terakhir: 2026-11-09 (sub task dan kartu profil, lihat Daftar revisi di bagian bawah).

## Cara kerja

- Durasi sprint 1 minggu. Kalau minggu itu banyak kerjaan kantor lain, sprint boleh diperpanjang jadi 2 minggu, asal ditulis di catatan sprint.
- Awal sprint: pilih item dari backlog sprint itu. Item yang belum selesai pindah ke sprint berikutnya, jangan dipaksa selesai.
- Akhir sprint: demo ke Manajer di LAN kantor (mulai Sprint 2), lalu retro singkat: apa yang lancar, apa yang macet, apa yang diubah.
- Catatan harian cukup satu baris di bagian "Log" tiap sprint: hari ini mengerjakan apa, ada hambatan apa.
- Git: `main` selalu bisa jalan. Fitur dikerjakan di branch `feat/<nama>`, merge setelah lolos DoD.

## Pembagian kerja (berlaku mulai Sprint 6)

- Sprint 6 sampai 12 dikerjakan di percakapan bersama asisten: kode per langkah ditempel di chat, Rifky menulis dan mengujinya secara manual, lalu commit.
- Tiap langkah ditutup dengan `./scripts/check.sh` (mode bawaan, cepat), lalu `git add`, `git commit`, `git push` oleh Rifky.
- Test baru tidak ditulis per sprint. Semua pengujian otomatis dikerjakan di Sprint 13, dan itu satu-satunya sprint yang memakai Claude Code.
- Claude Code tidak boleh commit dan push.
- Revisi dari Manajer dicatat di Daftar revisi di bagian bawah, lalu disisipkan ke sprint yang sesuai.
- Setelah tiap langkah, asisten menampilkan tabel semua langkah sprint beserta statusnya.

## Definition of Done (berlaku untuk semua item)

- Aturan hak akses dicek di backend, bukan cuma disembunyikan di frontend (FR-AUTH-03).
- Aturan bisnis baru ditulis di bagian Keputusan sprint-nya supaya Sprint 13 tahu apa yang harus diuji. Test ditulis di Sprint 13, bukan per sprint.
- UI berbasis data punya state kosong, loading, dan error.
- Jalan di lebar 360px, 768px, dan 1280px tanpa scroll horizontal. Tap target minimal 44x44px.
- Bisa dipakai dengan keyboard saja dan focus terlihat.
- Aksi yang mengubah data tercatat di `audit_logs` bila SRS memintanya (BR-12).
- Tidak ada teks karangan di UI. Data contoh hanya di seed dev.
- Console browser bersih, `go vet` dan lint frontend bersih.

## Ringkasan sprint

| Sprint | Fokus | Hasil yang bisa didemokan |
|---|---|---|
| 0 | Fondasi proyek | Repo, database dengan skema lengkap, backend dan frontend jalan, `/health` hijau |
| 1 | Login dan sesi | Login, logout, lockout, ganti password, idle timeout |
| 2 | Akun, peran, divisi | Halaman Pengguna, Pengaturan divisi, Profil, Audit Log |
| 3 | Task inti dan label | Manajer membuat task, alur status lengkap, riwayat, label |
| 4 | Papan Kanban | Papan dengan drag-and-drop, saklar Task saya / Semua task, filter |
| 5 | Lampiran | Lampiran brief dan deliverable, tempel screenshot, validasi tipe |
| 6 | Fondasi pencarian | Index database untuk pencarian |
| 7 | Sub task, backend | Task dengan sub task, PIC banyak per sub task |
| 8 | Sub task, frontend | Form sub task, pemilih PIC ganda, kartu papan baru |
| 9 | Arsip dan pencarian | Arsip dengan pencarian, filter, urut, pagination |
| 10 | Profil pengguna | Foto, jabatan, kartu profil di papan, ubah profil |
| 11 | Poles UI/UX | Papan dan semua halaman dipoles, konsisten |
| 12 | Persiapan deploy dan migrasi data | Impor Excel, file deploy, skrip backup (belum dijalankan) |
| 13 | Pengujian | Test otomatis oleh Claude Code, p95, peta AC |
| 14 | Deploy, UAT, go-live | Aplikasi di VM, semua AC lolos, sistem dipakai tim |

---

## Sprint 0: Fondasi proyek

**Tujuan:** kerangka kosong yang sudah jalan dari ujung ke ujung (frontend memanggil backend, backend membaca database), supaya sprint berikutnya tinggal menambah fitur.

### 0.1 Pasang alat di laptop dev

- Git
- Go (versi stabil terbaru)
- Node.js LTS dan npm
- Docker Desktop (atau Docker Engine di WSL/Linux)
- Klien database: DBeaver, TablePlus, atau `psql`
- Alat migrasi: `golang-migrate` CLI (usulan, alasannya: file SQL polos `up`/`down`, mudah dibaca dan tidak mengikat ke ORM)

Catat versi yang terpasang di `README.md` supaya VM produksi nanti memakai versi yang sama.

### 0.2 Keputusan teknis (dikunci)

| Hal | Pilihan | Alasan |
|---|---|---|
| Router Go | Gin | Sudah biasa dipakai, middleware auth dan rate limit gampang dipasang |
| Driver database | pgx v5 (pool `pgxpool`) | Driver PostgreSQL native, parameterized query (NFR-SEC-05) |
| Akses query | SQL tulis tangan, `pgx.CollectRows` + `RowToStructByName` | Tanpa generator, query tetap terlihat, satu file repo per domain |
| Migrasi | golang-migrate, file `up`/`down` | Versi skema tercatat di `schema_migrations` |
| Hash password | argon2id (`golang.org/x/crypto/argon2`) | Algoritme khusus password (NFR-SEC-01) |
| JWT | `github.com/golang-jwt/jwt/v5` | Library JWT Go yang umum dipakai dan masih dirawat |
| Frontend | Vite, Vue 3, TypeScript, Vue Router, Pinia | Tipe data API terjaga, state login dan filter di Pinia |
| CSS | Tailwind v4 lewat `@tailwindcss/vite`, token di `@theme` | Palet DESIGN.md sudah dihitung kontrasnya |
| Font | IBM Plex Sans dan Plex Mono, di-host sendiri | Jalan di LAN, tidak bergantung ke Google Fonts |
| Klien database | TablePlus lewat SSH, psql di container untuk darurat | Visual untuk sehari-hari, psql selalu tersedia di server |
| Package manager | npm | Sudah terpasang, satu lockfile |

### 0.3 Struktur repo

```
nvm-internal/
  backend/
    cmd/server/main.go
    internal/
      config/        baca env
      db/            koneksi pgx, query
      http/          router, middleware, handler
      auth/
      task/
      audit/
    migrations/      0001_init.up.sql, 0001_init.down.sql
    go.mod
  frontend/
    src/
      api/           wrapper fetch
      components/
      pages/
      stores/
      router/
    index.html
    tailwind.config.js
  deploy/
    docker-compose.dev.yml
    docker-compose.prod.yml
    nginx.conf
  docs/
    SRS_Sistem_Internal_NVM.docx
    nvm_internal.dbml
    DESIGN.md
    sprint.md
  .env.example
  .gitignore
  README.md
```

### 0.4 Backlog

- [x] S0-01 Buat repo Git privat dan struktur folder di atas. `.env` masuk `.gitignore`, sediakan `.env.example` (NFR-SEC-10).
- [x] S0-02 `docker-compose.dev.yml` berisi PostgreSQL saja, data di named volume, port hanya ke `127.0.0.1`.
- [x] S0-03 Migrasi `0001_init` dari ERD: 4 enum, 8 tabel, semua index, ditambah:
  - unique index `lower(username)`
  - `CHECK (due_date IS NULL OR start_date IS NULL OR due_date >= start_date)`
  - `CHECK ((status = 'completed') = (solved_date IS NOT NULL))`
  - sequence `task_number_seq` untuk ID T-nnnn
- [x] S0-04 Seed dev: divisi Marketing dan Creative, satu akun Admin dengan `must_change_password = true`. Password awal dibaca dari env, bukan ditulis di kode.
- [x] S0-05 Backend: baca config dari env, koneksi pool pgx, endpoint `GET /api/health` yang mengecek database, log terstruktur, graceful shutdown.
- [x] S0-06 Format galat API yang seragam, mis. `{ "error": { "code": "forbidden", "message": "..." } }`. Pesan untuk pengguna, detail teknis hanya di log (NFR-REL-06).
- [x] S0-07 Frontend: proyek Vite, Tailwind dengan token warna DESIGN.md, font Plex lokal, layout dasar (top bar desktop, bottom nav mobile), halaman 404.
- [x] S0-08 Proxy dev Vite `/api` ke backend, sehingga cookie nanti satu origin.
- [x] S0-09 Halaman sementara yang memanggil `/api/health` dan menampilkan status, lengkap dengan state loading dan error.
- [x] S0-10 Lint dan format: `gofmt`, `go vet`, ESLint, Prettier. Script `make dev` atau `npm run` untuk menjalankan semuanya.

### 0.5 Selesai kalau

- `docker compose up` lalu `migrate up` membuat 8 tabel tanpa galat, dan `migrate down` bisa membersihkannya lagi.
- Insert task dengan `due_date` lebih awal dari `start_date` ditolak database.
- Dua username `Staf.A` dan `staf.a` tidak bisa dibuat berdua.
- Halaman frontend menampilkan status database dari `/api/health`, dan menampilkan pesan galat saat database dimatikan.

### Log

- (isi satu baris per hari kerja)
### Retro

- Lancar: skema dan uji skema lolos di percobaan pertama; health check dan tiga state halaman Status sesuai rencana.
- Macet: tunnel SSH sering terputus atau lupa dibuka; file Go tersimpan CRLF sehingga gofmt gagal.
- Diubah untuk Sprint 1: pasang SSH key supaya tunnel tanpa password; jalankan ./scripts/check.sh sebelum setiap commit.
---

## Sprint 1: Login dan sesi

**Tujuan:** pengguna bisa masuk dan keluar dengan aman.
**SRS:** FR-AUTH-01, 05, 06, 07, 08, 09, 11, NFR-SEC-01, 02, 04, 06. **AC:** AC-01, AC-02.

- [x] S1-01 `POST /api/auth/login` dengan username dan password. Hash password memakai bcrypt.
- [x] S1-02 JWT 30 menit di cookie HttpOnly dan SameSite=Lax, berisi `user_id`, `role`, `token_version`. Atribut Secure dari env `COOKIE_SECURE`.
- [x] S1-03 Middleware auth: tolak token yang `token_version`-nya tidak sama dengan database atau akun yang nonaktif (FR-AUTH-08).
- [x] S1-04 Idle timeout: token diperbarui saat ada request, kedaluwarsa setelah 30 menit tanpa aktivitas.
- [x] S1-05 Lockout progresif: 5 kali gagal dikunci 5 menit, lalu 15, lalu 30 dan seterusnya 30. Reset saat login berhasil.
- [x] S1-06 Rate limit per IP di endpoint login (NFR-SEC-04). 10 percobaan, lalu 1 per 30 detik.
- [x] S1-07 Pesan login gagal tidak membedakan "username tidak ada" dan "password salah".
- [x] S1-08 Audit: `login_success`, `login_failed` (user_id kosong bila username tidak terdaftar), `account_locked`, `logout`, `password_changed`.
- [x] S1-09 Perlindungan CSRF lewat cek header Origin.
- [x] S1-10 `POST /api/auth/logout` dan `GET /api/auth/me`.
- [x] S1-11 Ganti password sendiri dengan password lama (halaman Profil). Login dengan `must_change_password` langsung diarahkan ke halaman Buat password baru.
- [x] S1-12 Halaman Masuk sesuai Stitch, termasuk state akun dikunci (jam buka kunci) dan sesi berakhir.
- [x] S1-13 Route guard di Vue: halaman selain login butuh sesi, `http://<IP>/` membuka login.

**Keputusan:**
- Logout mengakhiri sesi di semua perangkat (token_version naik), karena skema tidak punya tabel sesi.
- Akun terkunci menampilkan jam buka kunci walaupun ini menandakan username ada; diterima karena sistem hanya di LAN.
- Layar Buat password baru memakai password login yang disimpan sementara di memori, sehingga hanya ada dua kolom seperti di Stitch. Bila halaman dimuat ulang, muncul kolom "Password sementara".
- Password baru minimal 8 karakter, maksimal 72 (batas bcrypt).
- Rate limit login: 10 percobaan per IP, lalu 1 per 30 detik.
- Sementara semua peran diarahkan ke halaman Status setelah login. "Halaman sesuai peran" (AC-01) diselesaikan saat halaman tujuannya ada: Pengguna untuk Admin (Sprint 2), Papan Task untuk Manajer dan Staf (Sprint 4).

### Retro

- Lancar: aturan lockout diuji sebagai fungsi murni dulu, jadi AC-02 terbukti sebelum ada HTTP; backend selesai dalam satu hari.
- Macet: server lupa di-restart setelah perubahan; proxy Vite mengganti Host sehingga cek Origin gagal; file CRLF dari VS Code; perintah curl dengan JSON rawan salah ketik.
- Diubah untuk Sprint 2: restart server setiap selesai mengubah kode Go; uji lewat browser sesegera mungkin, curl hanya untuk kasus yang tidak bisa dari halaman.
---

## Sprint 2: Akun, peran, divisi

**Tujuan:** Admin dan Manajer bisa mengelola orang tanpa menyentuh database.
**SRS:** FR-AUTH-02, 03, 04, 10, NFR-SEC-03, 11. **AC:** AC-03, AC-04 (bagian akun).

- [x] S2-01 Middleware RBAC: `requireRole(admin, manager)`. Manajer memegang semua hak Admin.
- [x] S2-02 Kelola akun: buat, ubah nama, peran, divisi, nonaktifkan, aktifkan, reset password, buka kunci. Nonaktif dan reset menaikkan `token_version`.
- [x] S2-03 Tidak bisa menonaktifkan diri sendiri, dan Admin atau Manajer aktif tidak boleh habis.
- [x] S2-04 Kelola divisi: tambah, ganti nama, nonaktif/aktif. Divisi tidak bisa dihapus.
- [x] S2-05 Audit perubahan akun, peran, dan divisi dengan `old_value` dan `new_value`, pelaku di `user_id`, sasaran di `entity` dan `entity_id`.
- [x] S2-06 Halaman Pengguna, Pengaturan (Divisi), dan Audit Log dengan filter bulan, pengguna, dan jenis aksi. Profil sudah selesai di Sprint 1.
- [x] S2-07 Menu ikut peran. Staf tidak melihat Pengguna, Pengaturan, Audit Log, dan backend menolak dengan 403.

**Demo pertama ke Manajer:** login, buat akun Staf, login sebagai Staf, cek menu.
**Butuh dari Manajer:** daftar pengguna awal (nama, username, peran, divisi), termasuk setiap anggota Creative.

**Keputusan:**
- Password awal dan reset password diketik oleh pengelola (sesuai Stitch), minimal 8 karakter, dan wajib diganti saat login berikutnya.
- Reset password juga membuka kunci akun dan mengakhiri semua sesi akun itu.
- Username 3 sampai 50 karakter: huruf, angka, titik, garis bawah, atau tanda hubung, tanpa spasi.
- Perlindungan pengelola terakhir: tidak bisa menonaktifkan diri sendiri, dan tidak ada aksi yang boleh membuat jumlah Admin dan Manajer aktif menjadi nol.
- Divisi tidak bisa dihapus, hanya dinonaktifkan. Divisi nonaktif tidak muncul di pilihan akun baru, tapi akun lama tetap di divisinya.
- Kolom user_id di audit log selalu berisi pelaku aksi; sasarannya di entity dan entity_id.
- Halaman awal sementara: Admin dan Manajer ke Pengguna, Staf ke Status, sampai Papan Task ada di Sprint 4.

- 2026-10-28: Sprint 2 langkah 1-4 selesai (backend). Helper audit pelaku dan sasaran, middleware peran, kelola akun (buat, ubah, nonaktif/aktif, reset password, buka kunci, perlindungan pengelola terakhir), kelola divisi, daftar audit log dengan filter dan halaman. Test database diantrekan dengan advisory lock setelah deadlock antar package.
- 2026-10-29: Sprint 2 langkah 5-6 selesai (frontend). Halaman Pengguna (tabel di layar lebar, kartu di ponsel, panel tambah/ubah, menu aksi, dialog reset dan nonaktif), Pengaturan divisi, Audit Log dengan teks aksi berbahasa Indonesia, menu sesuai peran. Semua uji browser lolos.

**Selesai kalau:** AC-03 dan AC-04 (bagian akun) lolos.
- AC-03: lolos, kecuali "nama akun nonaktif tetap tampil pada task lama", yang diuji di Sprint 4 setelah task dan papan ada.
- AC-04 bagian akun: lolos. Bagian task diuji di Sprint 3.

## Sprint 3: Task inti dan label

**Tujuan:** siklus hidup task benar di backend sebelum ada papan.
**SRS:** FR-TASK-01 sampai 07, 10, 11, 16, 17, 19, 21, 22, BR-01 sampai 05, 07. **AC:** AC-04 (bagian task), AC-08, AC-14.

- [x] S3-01 `POST /api/tasks` khusus Manajer. ID T-nnnn dari `task_number_seq`, status awal Requested, wajib judul, minimal satu PIC, urgency.
- [x] S3-02 Divisi task diisi dari divisi PIC pertama.
- [x] S3-03 Ubah task (Manajer), soft delete (Manajer, tercatat audit).
- [x] S3-04 Endpoint transisi status dengan tabel aturan:
  - PIC: Requested ke On Progress, On Progress ke Ready, Ready ke On Progress (wajib catatan).
  - Manajer: semua di atas, Ready ke Completed, Completed ke On Progress (buka kembali, `solved_date` dikosongkan).
  - Admin: tidak ada.
- [x] S3-05 `solved_date` otomatis saat Completed. Setiap transisi tercatat sebagai `task_status_changed`.
- [x] S3-06 Optimistic locking lewat kolom `version`: request membawa versi, beda versi dibalas 409 (persiapan AC-10).
- [x] S3-07 Staf boleh membaca semua task, tetapi semua aksi pada task yang bukan miliknya dibalas 403 (NFR-SEC-03).
- [x] S3-08 Label: CRUD oleh Manajer/Admin (nama, warna dari palet tetap, divisi opsional). Staf memasang label pada task miliknya, hanya label divisinya atau label untuk semua divisi.
- [x] S3-09 `GET /api/tasks/:id` berisi detail, PIC, label, dan riwayat (terbaru di atas).
- [x] S3-10 Dialog Buat task dan panel detail task di frontend (sementara di halaman daftar biasa).
- [x] S3-11 Pengaturan bagian label.

**Selesai kalau:** test Go untuk setiap baris tabel transisi, termasuk yang harus ditolak, semuanya lolos.
**Butuh dari Manajer:** daftar label awal (nama dan warna).

**Keputusan:**
- PIC hanya boleh akun aktif dengan peran Staf atau Manajer. Admin tidak bisa jadi PIC karena Admin tidak boleh memindahkan task.
- Divisi task mengikuti PIC pertama yang dipilih, saat dibuat maupun saat daftar PIC diubah.
- Catatan revisi wajib saat Ready dikembalikan ke On Progress, maksimal 500 karakter. Buka kembali dari Completed tidak wajib catatan.
- Perpindahan yang tidak ada di alur (mis. Requested langsung ke Completed) ditolak untuk semua peran.
- Staf yang menjadi PIC boleh: memindahkan status (tanpa Completed), memasang/melepas label divisinya, dan mengisi tautan deliverable. Atribut lain hanya Manajer.
- Task yang dihapus (soft delete) hilang dari semua daftar dan tidak bisa dipulihkan dari aplikasi; jejaknya tetap di audit log.
- Riwayat task di panel detail dibaca dari audit log (pembuatan, perubahan, perpindahan status), terbaru di atas.
- Palet label: 8 warna terang dengan teks warna ink, semuanya di atas 12:1.

**Hasil:** selesai 2026-10-29. Lolos lewat test Go: AC-04 bagian task, AC-08 bagian backend, dan AC-14 bagian backend.
Sisa: AC-08 papan dan arsip (Sprint 4 dan 6), AC-14 filter papan per label (Sprint 4), AC-03 nama akun nonaktif di task lama (Sprint 4).

**Keputusan:**
- Perpindahan di luar alur dibalas 409, sedangkan perpindahan yang bukan hak pengguna dibalas 403. Frontend jadi bisa membedakan "alurnya tidak ada" dan "bukan hakmu".
- Divisi task mengikuti PIC pertama. Urutan PIC disimpan dengan `clock_timestamp()` supaya urutannya tetap walaupun beberapa PIC dimasukkan dalam satu transaksi.
- Staf hanya mengatur label divisinya dan label untuk semua divisi. Label divisi lain di task miliknya dipertahankan, tidak ikut terhapus.
- Label nonaktif tetap menempel di task lama, tetapi ditolak untuk pemasangan baru.
- Untuk sementara, detail task dibuka dari halaman daftar. Papan menyusul di Sprint 4.
- Di Pengaturan, Divisi dan Label disusun berurutan ke bawah, tanpa tab, karena kedua daftarnya pendek.
- Aksi task di Audit Log menampilkan ID (T-nnnn), bukan judul, karena judul bisa berubah.

**Retro:**
- Macet: `task_number_seq` tidak ikut di-reset oleh TRUNCATE di test; `entity_id` yang nullable jadi `pgtype.Text` di sqlc; ReferenceError karena `watch` dengan `immediate` jalan sebelum `const` dideklarasikan; garis merah palsu di VS Code karena indeks basi.
- Lancar: (isi sendiri)
- Diubah untuk Sprint 4: (isi sendiri)
---

## Sprint 4: Papan Kanban

**Tujuan:** layar utama yang dipakai tiap hari.
**SRS:** FR-TASK-08, 09, 18. **AC:** AC-05, AC-06, AC-10, AC-12.

- [x] S4-01 `GET /api/board` dengan parameter `scope` (mine/all), `pic`, `division`, `label`, `urgency`, `due`. Jumlah per status ikut lingkup.
- [x] S4-02 Papan desktop: 4 kolom, drag-and-drop hanya untuk kartu yang boleh dipindah oleh pengguna itu.
- [x] S4-03 Tombol "Pindahkan" di kartu sebagai pengganti drag (layar sentuh dan keyboard).
- [x] S4-04 Mobile: tab per status, bukan 4 kolom yang dikecilkan.
- [x] S4-05 Saklar Staf "Task saya | Semua task", default Task saya, reset setiap login, state di URL (`?scope=all&pic=staf.b`).
- [x] S4-06 Filter PIC: tersembunyi di mode Task saya, muncul di Semua task. Memilih orang lain pindah otomatis ke Semua task, memilih diri sendiri saja kembali ke Task saya. Manajer tanpa saklar, default "Semua orang".
- [x] S4-07 Kartu dan detail task orang lain: tanpa tombol aksi, baris "Kamu hanya bisa melihat task ini."
- [x] S4-08 Tampilan kosong dengan tombol "Lihat semua task".
- [x] S4-09 Dialog revisi saat Ready dikembalikan ke On Progress.
- [x] S4-10 Konflik 409: tampilkan pesan bahwa task sudah diubah orang lain, lalu muat ulang kartu.
- [x] S4-11 Penanda terlambat bila tenggat lewat dan status belum Completed.

**Demo:** satu sesi bareng Manajer dan satu Staf memindahkan task yang sama dari dua perangkat.

**Hasil:** selesai 2026-10-29. Lolos lewat test Go dan uji manual: AC-05 (bagian papan), AC-06, AC-10, AC-12 (halaman papan), AC-03 (nama akun nonaktif di task lama), AC-08 bagian papan, AC-14 filter per label.
Sisa: AC-05 dan AC-08 bagian arsip (Sprint 9).

**Keputusan:**
- Kolom Completed hanya berisi task yang selesai 7 hari terakhir; selebihnya hanya di Arsip. Perlu konfirmasi Manajer karena AC-08 menyebut task Completed hilang dari papan aktif.
- `move_to` di setiap kartu dihitung backend dengan aturan yang sama dengan detail task, supaya tombol dan drag tidak menawarkan perpindahan yang akan ditolak.
- Filter PIC memilih satu orang, memakai username di URL. Staf yang memilih dirinya sendiri kembali ke "Task saya"; URL dengan PIC orang lain otomatis menjadi "Semua task".
- Login biasa selalu mulai dari "Task saya". Login ulang setelah sesi habis mengembalikan tampilan terakhir lewat `?kembali=`.
- "Hari ini" untuk filter dan penanda terlambat dihitung dalam WIB, bukan jam server.
- Drag-and-drop memakai HTML5 bawaan untuk mouse; layar sentuh dan keyboard memakai tombol Pindahkan. Revisi lewat drag tetap membuka dialog catatan.
- Di layar sempit, papan menampilkan satu kolom dengan deretan tombol kolom (pola `aria-pressed`), bukan widget tab.
- Kotak "Cari task" dan tautan "Lihat semua di Arsip" dari Stitch ditunda ke Sprint 9 karena halamannya belum ada.

**Retro:**
- Macet: helper test HTTP (`liveServer`) ternyata belum ada di repo; sqlc salah menulis ulang `@today::date + 6` karena operator `@` lebih lemah dari `+`, jadi harus `(@today::date) + 6`; rute `/board` dan `/people` sempat terdaftar di grup tanpa pemeriksaan login, tertangkap oleh `TestTaskReadRoutesNeedLogin`.
- Lancar: (isi sendiri)
- Diubah untuk Sprint 5: (isi sendiri)
- Catatan uji: tampilan mobile diuji lewat simulator di browser; uji di ponsel sungguhan masuk S13-08.

---

## Sprint 5: Lampiran

**Tujuan:** brief dan hasil kerja bisa diunggah dengan aman.
**SRS:** FR-TASK-15, 23, NFR-SEC-09, NFR-PERF-04. **AC:** AC-07, AC-15.

- [x] S5-01 `POST /api/tasks/:id/attachments` dengan `kind` brief (Manajer) atau deliverable (PIC atau Manajer).
- [x] S5-02 Validasi: allowlist PDF, CSV, TXT, DOC/DOCX, XLS/XLSX, PPT/PPTX, JPG/JPEG, PNG. Tipe dicek dari isi berkas, bukan ekstensi saja. Tolak berkas makro (DOCM, XLSM, PPTM) dan executable. Batas 50 MB di backend dan di Nginx.
- [x] S5-03 Simpan di disk dengan nama UUID, di luar folder yang disajikan web. Simpan checksum SHA-256.
- [x] S5-04 Unduh lewat endpoint yang cek izin, dengan `Content-Disposition: attachment` dan nama asli.
- [x] S5-05 Hapus lampiran (soft delete) dan audit `attachment_added` / `attachment_deleted`.
- [x] S5-06 Frontend: area tarik berkas, tombol pilih berkas (di ponsel menawarkan kamera atau galeri), tempel screenshot dengan Ctrl+V di dialog Buat task dan Ubah task.
- [x] S5-07 Progres unggah dan pesan galat tipe atau ukuran.
- [x] S5-08 Tautan deliverable hanya menerima `http` dan `https` (NFR-SEC-06).

**Hasil:** selesai 2026-10-30. Lolos lewat test Go, curl, dan uji browser: AC-07 dan AC-15.
Sisa: S5-02 bagian Nginx (`client_max_body_size 51m`) dipasang di Sprint 12; S5-08 sudah selesai sejak Sprint 3 (`cleanURL`).

**Keputusan:**
- Berkas disimpan di `STORAGE_DIR` (dev: `backend/storage`, produksi: `/srv/nvm-internal/storage` di VM, milik UID 10001, mode 0750). Di dalamnya ada `attachments/` dan `tmp/` pada disk yang sama, supaya pemindahan dari `tmp/` bersifat atomik.
- Nama berkas di disk adalah UUID dari database, disebar per dua huruf pertama. Nama asli hanya disimpan di database, dan tidak pernah dipakai sebagai path.
- Tipe dicek dari isi dan ekstensi: tanda tangan format untuk PDF, PNG, dan JPG; isi ZIP untuk DOCX, XLSX, dan PPTX; stream Compound File untuk DOC, XLS, dan PPT (library `richardlehane/mscfb`); tanpa byte kontrol untuk TXT dan CSV.
- Batasan yang diketahui: makro di PPT lama (97-2003) tidak terdeteksi. Diterima karena SRS mengizinkan PPT, dengan peredam berupa unduhan yang selalu `attachment` dan `nosniff`. **Perlu ditanyakan ke Manajer apakah tim masih memakai .ppt; kalau tidak, hapus dari daftar izin.**
- Izin dicek sebelum isi berkas dibaca (`kind` dikirim lewat URL), lalu dicek ulang di dalam transaksi.
- Melihat dan mengunduh boleh oleh semua akun yang bisa membuka detail task (FR-TASK-18). Unggah dan hapus: brief hanya Manajer, deliverable Manajer atau PIC.
- Hapus lampiran berupa soft delete; berkasnya tetap di disk.
- Unggah tidak menaikkan `version` task, supaya tidak memicu konflik 409 dengan orang yang sedang mengedit.
- Di form Buat/Ubah task, brief ditahan di browser lalu diunggah setelah task tersimpan. Kalau ada yang gagal, task tidak dibatalkan; pesannya menyebut berkas yang gagal, dan detail task terbuka untuk unggah ulang.

**Retro:**
- Macet: Go menganggap paket `storage` tidak ada karena filenya tersimpan di `backend/storage` (folder data yang di-gitignore), bukan `backend/internal/storage`; otomasi Word lewat PowerShell menggantung karena dialog yang tersembunyi; sqlc membuat `pgtype.UUID` dari `@id::uuid`, sehingga butuh konversi di Go.
- Catatan uji: menu kamera atau galeri di ponsel sungguhan, dan Ctrl+V di Safari, masuk S13-08.
- Untuk Sprint 12: pasang `client_max_body_size` di Nginx, masukkan `/srv/nvm-internal/storage` ke backup harian, dan buat pembersihan berkas yatim (berkas di `attachments/` tanpa baris database).
- Lancar: (isi sendiri)
- Diubah untuk Sprint 6: (isi sendiri)

---

## Sprint 6: Fondasi pencarian

**Tujuan:** index database untuk pencarian sudah ada sebelum fitur yang memakainya.

- [x] S6-01 Migrasi `000002_search`: ekstensi `pg_trgm`, index GIN trigram di `tasks.id`, `title`, `description`, `deliverable_url`, dan `task_attachments.original_name`; index `tasks_requested_by_idx`; index parsial `tasks_archive_idx` untuk task Completed yang belum dihapus.

**Keputusan:**
- Pencarian memakai `pg_trgm`, bukan `tsvector`, karena PostgreSQL tidak punya kamus bahasa Indonesia dan kita butuh pencocokan potongan kata serta ID (mis. `T-00`).
- Kata kunci minimal 3 huruf, mengikuti cara kerja trigram.
- Sisa pekerjaan Arsip (S6-01 sampai S6-06 versi lama) dipindah ke Sprint 9, karena pencarian harus sadar sub task.

---

## Sprint 7: Sub task, backend

**Tujuan:** task punya sub task, dan PIC (bisa lebih dari satu) menempel di sub task. Revisi 1 dari Manajer.
**Aturan baru:** Manajer mengisi judul task, lalu sub task di bawahnya. Tiap sub task punya nama dan satu atau lebih PIC. Status Kanban tetap di task induk, sub task tidak punya status. Sub task opsional.

- [ ] S7-00 Ubah `scripts/check.sh` jadi dua mode: bawaan (gofmt, `go build`, `npm run check`) dan `--full` (ditambah `go vet` dan `go test`). Alasan: selama fase build, test tidak ditulis ulang, dan `go test` lewat SSH tunnel yang membuat pemeriksaan 4 sampai 6 menit.
- [ ] S7-01 Migrasi `000003`: tabel sub task (task, judul, urutan, waktu) dan tabel PIC sub task (sub task, akun, waktu pemasangan dengan `clock_timestamp()`). Tabel PIC level task yang lama dihapus; `down` mengembalikan strukturnya.
- [ ] S7-02 Data task di database dev dikosongkan saat migrasi (belum ada data produksi).
- [ ] S7-03 Query sqlc untuk sub task: tambah, ubah nama, hapus, atur urutan, ganti daftar PIC.
- [ ] S7-04 Service: buat task beserta sub task dalam satu transaksi; ubah task menggantikan daftar sub task secara utuh, dengan optimistic locking `version` task.
- [ ] S7-05 Validasi: judul sub task wajib dan tidak lebih panjang dari judul task; PIC hanya akun aktif berperan Staf atau Manajer; tanpa PIC ganda pada satu sub task.
- [ ] S7-06 PIC dan divisi task menjadi turunan: PIC task adalah gabungan unik PIC semua sub task; divisi task mengikuti PIC pertama dari sub task pertama yang punya PIC. Dihitung ulang setiap daftar sub task berubah.
- [ ] S7-07 Hak akses: Staf yang menjadi PIC di salah satu sub task dianggap PIC task itu. Hak PIC (pindah status tanpa Completed, label divisinya, tautan deliverable, lampiran deliverable) mengikuti definisi baru. Mengubah sub task dan PIC-nya hanya Manajer.
- [ ] S7-08 Papan dan daftar: `scope=mine` dan filter `pic` membaca PIC lewat sub task. Kartu membawa jumlah sub task dan daftar PIC unik.
- [ ] S7-09 `GET /api/tasks/:id` memuat sub task beserta PIC masing-masing.
- [ ] S7-10 Audit: `subtask_added`, `subtask_renamed`, `subtask_removed`, `subtask_assignees_changed`, dengan `old_value` dan `new_value`.

**Keputusan sementara (konfirmasi ke Manajer):**
- Task tanpa sub task berarti tanpa PIC. Task seperti itu hanya bisa dipindahkan oleh Manajer. Perpindahan Requested ke On Progress tidak mewajibkan adanya PIC.
- Lampiran tetap per task, bukan per sub task.
- Test lama yang bergantung pada PIC level task akan gagal. Tidak diperbaiki di sprint ini; diperbaiki di Sprint 13.

**Selesai kalau:** `./scripts/check.sh` lolos dan alurnya terbukti lewat curl: buat task dengan dua sub task, sub task kedua punya dua PIC, task muncul di papan akun PIC mana pun.

---

## Sprint 8: Sub task, frontend

**Tujuan:** Manajer bisa mengisi sub task dan PIC dengan nyaman, semua orang melihat hasilnya di papan.

- [ ] S8-01 Form Buat dan Ubah task: bagian "Sub task" berupa daftar baris. Tiap baris punya nama dan pemilih PIC. Tambah, hapus, dan ubah urutan (tombol naik dan turun, tanpa wajib drag, supaya bisa lewat keyboard dan layar sentuh).
- [ ] S8-02 Komponen pemilih PIC ganda: cari nama, pilih beberapa, tampil sebagai chip yang bisa dilepas. Tap target 44px, operasi penuh dengan keyboard. Sementara pakai inisial; foto menyusul di Sprint 10.
- [ ] S8-03 Panel detail task: daftar sub task beserta PIC tiap sub task.
- [ ] S8-04 Kartu papan: jumlah sub task dan deretan PIC unik. Filter PIC dan "Task saya" mengikuti aturan baru.
- [ ] S8-05 State kosong ("Belum ada sub task"), galat per baris, dan pesan konflik 409 saat daftar sub task diubah dua orang bersamaan.
- [ ] S8-06 Hapus sisa UI PIC level task.

**Selesai kalau:** `./scripts/check.sh` lolos dan uji manual di browser: Manajer membuat task dengan sub task, Staf PIC melihatnya di Task saya dan memindahkannya, Staf lain hanya bisa melihat.

---

## Sprint 9: Arsip dan pencarian

**Tujuan:** task lama gampang dicari, termasuk lewat nama sub task. Sisa Sprint 6 versi lama.
**SRS:** FR-TASK-12, 13, 14, NFR-PERF-02, 03. **AC:** AC-09.

- [ ] S9-01 `GET /api/archive`: task Completed tanpa batas 7 hari, ikut saklar lingkup Staf (Task saya atau Semua task).
- [ ] S9-02 Kata kunci (pg_trgm, minimal 3 huruf) di ID, judul, deskripsi, tautan deliverable, nama pemohon, nama PIC, nama sub task, nama label, dan nama berkas lampiran.
- [ ] S9-03 Filter PIC, divisi, label, urgency, dan rentang tanggal selesai. Urutan bawaan terakhir selesai di atas; pilihan terlama dan urgency berbobot.
- [ ] S9-04 Halaman 25 task per halaman dengan total.
- [ ] S9-05 Kata kunci juga di papan (`GET /api/board`).
- [ ] S9-06 Halaman Arsip: tabel di desktop, kartu di ponsel, filter di URL, tombol buka kembali untuk Manajer. Kotak "Cari task" dan tautan "Lihat semua di Arsip" di papan. Teks Stitch "Semua proyek dan tugas studio..." dan "Tips Workflow" tidak dipakai.

**Catatan:** pengukuran p95 dengan data dummy 2.000 task dipindah ke Sprint 13.

---

## Sprint 10: Profil pengguna dan kartu profil di papan

**Tujuan:** setiap orang punya identitas yang terlihat: foto, nama, jabatan, divisi. Revisi 2 dari Manajer (bagian data dan kartu).
**Aturan baru:** area kiri papan yang kosong tidak diisi sidebar navigasi, tetapi kartu profil.

- [ ] S10-01 Migrasi `000004`: kolom jabatan (`job_title`) dan foto profil di tabel akun.
- [ ] S10-02 Unggah foto: JPG atau PNG, maksimal 2 MB, tipe dicek dari isi. Backend memotong jadi persegi, mengecilkan ke ukuran tetap, lalu menyimpan ulang tanpa metadata EXIF (alasan: koordinat lokasi di foto ponsel tidak ikut tersimpan). Disimpan di `STORAGE_DIR/avatars/` dengan nama UUID.
- [ ] S10-03 Endpoint: ubah nama dan jabatan sendiri, ganti dan hapus foto sendiri, ambil foto akun mana pun (wajib login, dengan ETag supaya tidak diunduh ulang).
- [ ] S10-04 Aturan hak: tiap pengguna mengubah nama, jabatan, dan foto miliknya sendiri. Divisi dan peran tetap hanya Admin atau Manajer, ditampilkan baca-saja di kartu dengan keterangan siapa yang mengaturnya. Audit `profile_updated`.
- [ ] S10-05 Kartu profil di area kiri papan (desktop): foto atau inisial, nama, jabatan, divisi, tombol "Ubah profil" yang membuka panel. Isian jabatan kosong tampil sebagai ajakan mengisi, bukan teks karangan.
- [ ] S10-06 Ringkasan di kartu: jumlah task saya per status, dihitung dari data papan (tidak ada angka karangan). Klik angka menyaring papan.
- [ ] S10-07 Ponsel: kartu menjadi bar ringkas di atas papan yang bisa dibuka, bukan kolom kiri.
- [ ] S10-08 Foto atau inisial dipakai konsisten di top bar, chip PIC, kartu task, daftar Pengguna, dan Audit Log. Halaman Profil yang lama (ganti password) digabung ke panel ubah profil.
- [ ] S10-09 State: foto gagal dimuat (kembali ke inisial), unggah berjalan, ukuran atau tipe ditolak, simpan gagal.

**Butuh dari Manajer:** apakah jabatan diisi sendiri oleh tiap orang atau hanya pengelola (usulan di atas: diisi sendiri).

---

## Sprint 11: Poles UI/UX

**Tujuan:** papan Kanban dan semua halaman terasa rapi, hidup, dan konsisten. Revisi 2 dari Manajer (bagian tampilan).
**Syarat mulai:** `docs/DESIGN.md` ditempel ke percakapan. Dial ENERGY, RHYTHM, MOTION ditetapkan di awal sprint beserta satu kalimat alasan, lalu dipegang sampai akhir. Tanpa DESIGN.md hasilnya ditandai draf.

- [ ] S11-01 Token desain dan satu motif identitas, diturunkan dari DESIGN.md dan brand NVM.
- [ ] S11-02 Papan: kartu, kolom, rasa drag-and-drop, transisi saat kartu pindah kolom, penanda terlambat, tampilan kartu dengan sub task dan PIC.
- [ ] S11-03 Panel detail dan form task, termasuk daftar sub task.
- [ ] S11-04 Arsip, Pengguna, Pengaturan, Audit Log, dan kartu profil memakai bahasa visual yang sama.
- [ ] S11-05 State kosong, loading (kerangka sesuai bentuk konten), dan galat yang konsisten. Ilustrasi hanya SVG buatan sendiri.
- [ ] S11-06 Animasi: tiap animasi punya tujuan UX (umpan balik, orientasi, atau hubungan sebab akibat), tanpa loop. Versi diam untuk `prefers-reduced-motion`.
- [ ] S11-07 Audit kontras dengan perhitungan, tap target, fokus keyboard, dan lebar 360, 768, 1280, serta 1920 px tanpa scroll horizontal.

**Selesai kalau:** `./scripts/check.sh` lolos dan semua halaman dicek di rentang lebar di atas.

---

## Sprint 12: Persiapan deploy dan migrasi data

**Tujuan:** semua yang dibutuhkan untuk go-live sudah tertulis, tetapi belum dijalankan di VM.
**SRS:** FR-TASK-20, BR-09, NFR-SEC-07, 08, NFR-REL-01 sampai 05, NFR-RET-03.

Migrasi data:
- [ ] S12-01 CLI impor Go dari sheet Task Tracker dan Completed Task di `NVM - Marketing & Sales.xlsx`. ID asli dipertahankan, `is_migrated = true`. Setiap baris Excel menjadi satu task tanpa sub task.
- [ ] S12-02 Pemetaan nama di Request By dan Solved By ke username lewat file CSV. Nilai "Creative" dipetakan ke akun anggota divisi Creative.
- [ ] S12-03 Task Completed di Excel tanpa Solved Date ditolak CHECK database. Putuskan saat dry-run: isi dari tanggal lain di baris itu, atau tandai untuk diisi manual.
- [ ] S12-04 Mode `--dry-run` dengan laporan baris terbaca dan baris gagal beserta alasannya. Setelah impor, `setval` sequence ke ID terbesar.

Operasional (hanya ditulis):
- [ ] S12-05 `Dockerfile` backend, `docker-compose.prod.yml` (PostgreSQL, backend, Nginx), dan `nginx.conf` dengan `client_max_body_size 51m` serta `proxy_set_header Host $host`.
- [ ] S12-06 Skrip backup harian (database dan `/srv/nvm-internal/storage`), penyalinan ke hard disk eksternal, PC IT, dan laptop IT (salinan laptop dienkripsi).
- [ ] S12-07 Pembersih berkas yatim (berkas di `attachments/` dan `avatars/` tanpa baris database) dan job harian penghapus audit log login yang lebih tua dari 1 tahun.
- [ ] S12-08 Panduan satu halaman untuk tim (login, memindahkan task, sub task, unggah, profil) dan catatan operasional di README (backup, restore, update versi).

**Butuh dari Manajer:** pemetaan nama di Excel ke username akun.

---

## Sprint 13: Pengujian (satu-satunya sprint yang memakai Claude Code)

**Tujuan:** semua yang dibangun di Sprint 7 sampai 12 teruji, dikerjakan Claude Code secara otomatis, lalu direview.
**Aturan:** Claude Code tidak boleh commit dan push. Commit dan push dijalankan Rifky.

- [ ] S13-01 Kunci Claude Code lewat `.claude/settings.local.json` (tolak `git commit`, `git push`, `git reset`, `git checkout`).
- [ ] S13-02 Tulis `CLAUDE.md` (belum ada): stack, aturan hemat kuota, perintah `check.sh`, aturan tanpa commit, dan gerbang AntiSlop.
- [ ] S13-03 Prompt Claude Code: perbaiki test lama yang rusak karena Sprint 7, lalu tulis test backend untuk izin, transisi status, sub task dan PIC turunan, profil dan foto, arsip dan pencarian, lampiran.
- [ ] S13-04 Test frontend untuk komponen dengan logika (pemilih PIC ganda, filter di URL, kartu profil).
- [ ] S13-05 Peta AC-01 sampai AC-15 ke test atau langkah uji manual, dengan hasil PASS atau FAIL.
- [ ] S13-06 Data dummy sekitar 2.000 task, ukur p95 dari dalam VM (bukan dari PC), cek `EXPLAIN` bahwa index pencarian terpakai. Target p95 di bawah 1 detik.
- [ ] S13-07 Review hasil bersama, perbaiki temuan, commit.
- [ ] S13-08 Uji yang tidak bisa otomatis: ponsel sungguhan (kamera atau galeri, Safari, tempel gambar), pembaca layar, cek keyboard saja.

---

## Sprint 14: Deploy, UAT, dan go-live

**Tujuan:** tim memakai sistem ini, bukan Excel.

Deploy (dijalankan Rifky di VM):
- [ ] S14-01 Rapikan `~/nvm-internal` ke `/srv/nvm-internal`; mount hard disk baru ke `/srv/nvm-internal/storage` bila sudah ada.
- [ ] S14-02 Jalankan `docker-compose.prod.yml`; VM menyala otomatis saat Windows menyala; container `restart: unless-stopped`.
- [ ] S14-03 Port database tidak dibuka keluar VM; firewall hanya izinkan port 80 dari subnet kantor.
- [ ] S14-04 Backup harian berjalan, dan uji restore pertama ke database kosong dicatat waktunya (target di bawah 4 jam).
- [ ] S14-05 Diagnosa jaringan PC kantor (`Get-NetAdapter`, `Get-VMSwitch`, ping ke gateway) dan catat hasilnya.

UAT:
- [ ] S14-06 Checklist AC-01 sampai AC-15 bersama Manajer, hasil PASS atau FAIL.
- [ ] S14-07 Perbaiki temuan UAT. Temuan yang bukan bug masuk backlog setelah go-live.
- [ ] S14-08 Migrasi final dari Excel versi terakhir, lalu Excel dibekukan (hanya baca).
- [ ] S14-09 Buat akun semua anggota tim, bagikan username dan password awal secara langsung.

**Selesai kalau:** semua AC PASS dan tim sudah memakai sistem minimal satu minggu tanpa kembali ke Excel.

---

## Daftar revisi

| No | Tanggal | Isi | Masuk ke |
|---|---|---|---|
| 1 | 2026-11-09 | Task punya sub task; PIC dipilih per sub task, bisa lebih dari satu | Sprint 7 dan 8 |
| 2 | 2026-11-09 | Area kiri papan diisi kartu profil (foto, nama, jabatan, divisi, bisa diubah); poles UI/UX papan dan semua halaman | Sprint 10 dan 11 |

SRS v1.0 belum memuat kedua revisi. SRS diperbarui sebelum UAT supaya checklist AC sesuai aplikasi.

## Menunggu jawaban Manajer

- Daftar label awal (nama dan warna) dan daftar pengguna awal (nama, username, peran, divisi).
- Apakah tim masih memakai berkas .ppt lama. Kalau tidak, hapus dari daftar izin.
- Konfirmasi kolom Completed di papan hanya 7 hari terakhir (AC-08 menyebut task Completed hilang dari papan aktif).
- Task tanpa sub task berarti tanpa PIC, hanya Manajer yang memindahkan: setuju?
- Jabatan diisi sendiri oleh tiap orang atau hanya pengelola.

## Setelah go-live (belum dijadwalkan)

- HTTPS, lalu atribut cookie Secure, sebelum ada akses dari luar kantor.
- Fitur ekspor untuk Manajer (BR-11, belum ada di versi ini).
- Uji restore rutin tiap 3 bulan (NFR-REL-04).
- Upgrade npm 12 (ditunda).