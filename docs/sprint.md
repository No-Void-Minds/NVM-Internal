# Rencana Sprint Sistem Internal NVM

Acuan: SRS Sistem Internal NVM v1.0, ERD `nvm_internal.dbml`, layar Stitch, dan DESIGN.md.
Stack: Go, PostgreSQL, Vue 3 dengan Tailwind CSS, Docker di VM Ubuntu (Hyper-V) pada PC kantor.
Revisi terakhir: 2026-11-13 (sisa build pindah ke Claude Code, lihat Pembagian kerja). Sprint 7 selesai 2026-11-09, Sprint 8 selesai 2026-11-10, Sprint 9 selesai 2026-11-13.

## Cara kerja

- Durasi sprint 1 minggu. Kalau minggu itu banyak kerjaan kantor lain, sprint boleh diperpanjang jadi 2 minggu, asal ditulis di catatan sprint.
- Awal sprint: pilih item dari backlog sprint itu. Item yang belum selesai pindah ke sprint berikutnya, jangan dipaksa selesai.
- Akhir sprint: demo ke Manajer di LAN kantor (mulai Sprint 2), lalu retro singkat: apa yang lancar, apa yang macet, apa yang diubah.
- Catatan harian cukup satu baris di bagian "Log" tiap sprint: hari ini mengerjakan apa, ada hambatan apa.
- Git: `main` selalu bisa jalan. Fitur dikerjakan di branch `feat/<nama>`, merge setelah lolos DoD.

## Pembagian kerja

**Sprint 6 sampai S9-05 (selesai):** dikerjakan di percakapan bersama asisten. Kode per langkah ditempel di chat, Rifky menulis dan mengujinya manual, lalu commit.

**Mulai S9-06 (2026-11-13):** Manajer mengejar waktu, jadi sisa build (S9-06 sampai Sprint 12) dikerjakan Claude Code.
- Aturan untuk Claude Code ada di `CLAUDE.md` di root repo, dan larangan teknisnya di `.claude/settings.json` (tolak `git commit`, `git push`, `git reset`, `git rebase`, `git clean`, `migrate ... down`, membaca `.env`). Dua butir ini semula S13-01 dan S13-02, dimajukan.
- Claude Code tidak boleh commit dan push. Di akhir tiap langkah ia memberi perintah commit, Rifky yang menjalankan.
- Test otomatis tetap tidak ditulis per sprint. Semuanya di Sprint 13.
- Setiap langkah ditutup dengan `./scripts/check.sh` (mode bawaan) dan tabel status semua langkah sprint itu.
- Revisi dari Manajer dicatat di Daftar revisi, lalu disisipkan ke sprint yang sesuai. Keputusan yang belum pasti masuk "Menunggu jawaban Manajer".
- Plugin Claude Code yang aktif di level user: antislop, ponytail, ui-ux-pro-max. Kalau aturannya bertabrakan dengan `CLAUDE.md`, `CLAUDE.md` yang dipakai.

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
| 8 | Sub task, frontend | Form sub task, pemilih PIC ganda, centang sub task, task penuh di tab baru |
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

- [x] S7-00 Ubah `scripts/check.sh` jadi dua mode: bawaan (gofmt, `go build`, `npm run check`) dan `--full` (ditambah `go vet` dan `go test`). Alasan: selama fase build, test tidak ditulis ulang, dan `go test` lewat SSH tunnel yang membuat pemeriksaan 4 sampai 6 menit.
- [x] S7-01 Migrasi `000003`: tabel sub task (task, judul, urutan, waktu) dan tabel PIC sub task (sub task, akun, waktu pemasangan dengan `clock_timestamp()`). Tabel PIC level task yang lama dihapus; `down` mengembalikan strukturnya.
- [x] S7-02 Data task di database dev dikosongkan saat migrasi (belum ada data produksi).
- [x] S7-03 Query sqlc untuk sub task: tambah, ubah nama, hapus, atur urutan, ganti daftar PIC.
- [x] S7-04 Service: buat task beserta sub task dalam satu transaksi; ubah task menggantikan daftar sub task secara utuh, dengan optimistic locking `version` task.
- [x] S7-05 Validasi: judul sub task wajib dan tidak lebih panjang dari judul task; PIC hanya akun aktif berperan Staf atau Manajer; tanpa PIC ganda pada satu sub task.
- [x] S7-06 PIC dan divisi task menjadi turunan: PIC task adalah gabungan unik PIC semua sub task; divisi task mengikuti PIC pertama dari sub task pertama yang punya PIC. Dihitung ulang setiap daftar sub task berubah.
- [x] S7-07 Hak akses: Staf yang menjadi PIC di salah satu sub task dianggap PIC task itu. Hak PIC (pindah status tanpa Completed, label divisinya, tautan deliverable, lampiran deliverable) mengikuti definisi baru. Mengubah sub task dan PIC-nya hanya Manajer.
- [x] S7-08 Papan dan daftar: `scope=mine` dan filter `pic` membaca PIC lewat sub task. Kartu membawa jumlah sub task dan daftar PIC unik.
- [x] S7-09 `GET /api/tasks/:id` memuat sub task beserta PIC masing-masing.
- [x] S7-10 Audit: `subtask_added`, `subtask_renamed`, `subtask_removed`, `subtask_assignees_changed`, dengan `old_value` dan `new_value`.

**Keputusan sementara (konfirmasi ke Manajer):**
- Task tanpa sub task berarti tanpa PIC. Task seperti itu hanya bisa dipindahkan oleh Manajer. Perpindahan Requested ke On Progress tidak mewajibkan adanya PIC.
- Lampiran tetap per task, bukan per sub task.
- Test lama yang bergantung pada PIC level task akan gagal. Tidak diperbaiki di sprint ini; diperbaiki di Sprint 13.

**Selesai kalau:** `./scripts/check.sh` lolos dan alurnya terbukti lewat curl: buat task dengan dua sub task, sub task kedua punya dua PIC, task muncul di papan akun PIC mana pun.

**Hasil:** selesai 2026-11-09. Lolos lewat curl dan pemeriksaan database: buat task dengan sub task dan PIC ganda, task tanpa sub task, penolakan PIC tidak valid dan nama sub task kosong, ubah task (tambah, ganti nama, hapus, urut ulang, ganti PIC), 409 untuk versi lama, divisi mengikuti PIC pertama, "Task saya" dan hak PIC lewat sub task, enam jenis kejadian audit.

**Keputusan:**
- Daftar sub task yang dikirim saat ubah task menggantikan daftar lama secara utuh. Sub task yang tidak ikut terkirim dihapus, `id` 0 berarti baru. Frontend wajib selalu mengirim daftar lengkap dengan `id` untuk sub task yang sudah ada.
- Sub task dihapus permanen (tanpa kolom `is_deleted`); jejaknya ada di `audit_logs`, dan PIC-nya ikut terhapus lewat `ON DELETE CASCADE`.
- Posisi sub task unik per task lewat constraint `DEFERRABLE INITIALLY DEFERRED`, supaya dua sub task bisa bertukar urutan dalam satu transaksi.
- Maksimal 50 sub task per task dan 200 karakter per nama. Angka 50 hanya pengaman, bukan dari SRS.
- Task tanpa PIC memakai divisi pembuatnya saat dibuat. Saat diubah menjadi tanpa PIC, divisinya dibiarkan.
- Divisi dihitung dari database setelah sub task ditulis (`FirstPICDivision`), bukan dari urutan form, supaya sama dengan urutan pemasangan yang tersimpan.
- Perubahan yang hanya menyentuh sub task tetap menaikkan versi task.
- Aksi audit baru: `subtask_added`, `subtask_removed`, `subtask_renamed`, `subtask_assignees_changed`, `subtask_reordered`. Urutan hanya dibandingkan di antara sub task yang bertahan. Frontend memetakannya ke teks Indonesia di Sprint 8 (panel detail dan Audit Log).
- `task_created` menyimpan daftar sub task awal di `new_value.subtasks`.
- Database test `nvm_internal_test` belum dimigrasi ke `000003`, dan test lama yang memakai `PICIDs` belum diperbaiki. Keduanya dikerjakan di Sprint 13.

**Retro:**
- Macet: akhir baris CRLF dari kode tempelan (`gofmt -l` menandai file baru); `pic_ids` ditulis angka tunggal, bukan array (ditolak dengan benar); `dev-login.sh` dijalankan dari folder yang salah; `grep` terlalu rakus menangkap riwayat audit; backend belum di-restart setelah ubah kode.
- Lancar: (isi sendiri)
- Diubah untuk Sprint 8: (isi sendiri)

---

## Sprint 8: Sub task, frontend

**Tujuan:** Manajer mengisi sub task dan PIC dengan nyaman, PIC mencentang sub task yang sudah beres, dan semua orang melihat kemajuannya di papan. Revisi 1, 3, dan 4.
**Aturan baru (revisi 3 dan 4):**
- Sub task punya centang. Centang hanya bisa diubah saat task berstatus On Progress. Yang bisa mengubah: PIC sub task itu, dan Manajer untuk sub task yang tidak dipegang Manajer lain.
- Task yang punya sub task baru bisa pindah dari On Progress ke Ready kalau semua sub task sudah tercentang. Yang boleh memindahkan tetap Staf yang menjadi PIC di sub task mana pun, dan Manajer.
- Panel detail punya tombol untuk membuka task penuh di tab baru, di sebelah tombol tutup.
- Di form task, Deskripsi tampil di bagian utama, dan Lampiran brief pindah ke Detail tambahan.

- [x] S8-01 Form Buat dan Ubah task: bagian "Sub task" berupa daftar baris. Tiap baris punya nama dan pemilih PIC. Tambah, hapus, dan ubah urutan (tombol naik dan turun, tanpa wajib drag).
- [x] S8-02 Komponen pemilih PIC ganda: cari nama, pilih beberapa, tampil sebagai chip yang bisa dilepas. Tap target 44px, operasi penuh dengan keyboard.
- [x] S8-03 Panel detail task: daftar sub task beserta PIC tiap sub task. Teks riwayat dan Audit Log untuk lima aksi sub task.
- [x] S8-04 Form: Deskripsi pindah ke bagian utama, Lampiran brief pindah ke Detail tambahan (terbuka otomatis saat ada berkas ditambahkan).
- [x] S8-05 Backend centang sub task: migrasi `000004` (kolom `done_at` dan `done_by`), endpoint centang dan lepas centang, syarat semua tercentang untuk On Progress ke Ready, audit `subtask_checked` dan `subtask_unchecked`, jumlah sub task dan yang selesai ikut di respons papan dan daftar.
- [x] S8-06 Frontend centang: kotak centang di panel detail (aktif sesuai `can_check` dari server), kartu papan menampilkan kemajuan dan deretan PIC, tombol dan drag ke Ready ditahan dengan alasan yang terbaca selama masih ada yang belum tercentang, teks riwayat dan Audit Log untuk centang. Kategori "Task" di filter Audit Log ikut memuat aksi sub task dan lampiran.
- [x] S8-06b Revisi 4: Manajer boleh mencentang dan melepas centang sub task siapa pun, kecuali sub task yang dipegang Manajer lain. Satu fungsi aturan dipakai untuk penolakan 403 dan kolom `can_check`.
- [x] S8-07 Halaman task penuh di `/task/:id` dan tombol "Buka di tab baru" di panel detail, di sebelah tombol tutup. Dua kolom di layar lebar: isi kerja di kiri, label dan riwayat di kanan.
- [x] S8-08 State kosong, galat per baris, dan sisa UI PIC level task dibersihkan (hasil pemeriksaan: tidak ada UI PIC level task yang tersisa).

**Keputusan sementara revisi 3 dan 4 (konfirmasi ke Manajer):**
- Sub task tanpa PIC boleh dicentang Manajer. Sub task dengan PIC campuran (Staf dan Manajer lain) ditolak untuk Manajer yang bukan PIC-nya.
- Syarat semua tercentang berlaku untuk semua peran, termasuk Manajer. Task tanpa sub task tidak dikenai syarat ini.
- Di status Requested, Ready, dan Completed centang hanya terbaca. Saat Ready dikembalikan ke On Progress untuk revisi, centang dibiarkan seperti semula dan yang berhak bisa melepasnya.
- Mencentang tidak menaikkan `version` task, supaya dua orang yang mencentang sub task berbeda tidak saling menolak. Status task dikunci di dalam transaksi, jadi tidak bisa bentrok dengan perpindahan ke Ready.
- Menambah sub task pada task yang sudah Ready tidak mengembalikan statusnya.

**Selesai kalau:** `./scripts/check.sh` lolos dan uji manual di browser: Manajer membuat task dengan sub task, Staf PIC mencentang sub task miliknya, Ready ditahan sampai semuanya tercentang, task penuh terbuka di tab baru, Staf lain hanya bisa melihat.

**Hasil:** selesai 2026-11-10. Lolos `./scripts/check.sh`. Lewat curl: Ready ditolak 400 dengan jumlah sub task tersisa, non-PIC 403, body kosong 400, centang dan lepas centang 204, centang tidak menaikkan versi, lepas centang saat Ready ditolak 400, ringkasan di papan dan detail, lima kejadian audit, Manajer bukan PIC mendapat `can_check` true dan bisa mencentang dan melepas. Lewat browser: form sub task, pemilih PIC, panel detail, kotak centang dengan keyboard, tombol kartu "Lihat sub task", drag ke Ready ditolak dengan pesan, halaman `/task/:id` di tab baru, state kosong dan galat per baris, uji Manajer kedua.

**Keputusan:**
- Kolom `done_at` dan `done_by` selalu terisi berpasangan (constraint `CHECK`), jadi tidak ada centang tanpa tahu siapa yang mencentang. Untuk Manajer yang mencentang atas nama PIC, `done_by` berisi Manajer itu.
- `can_check` dihitung backend dengan fungsi yang sama dengan penolakan 403 (`canToggle`), supaya UI dan server tidak berbeda pendapat.
- Syarat Ready dicek di dalam transaksi `Move` yang sudah mengunci task. Pesannya 400, bukan 409, supaya papan tidak menampilkan pesan "diubah orang lain".
- Kartu On Progress yang sub task-nya belum lengkap memakai tombol "Lihat sub task" yang membuka detail, bukan dialog Pindahkan. Drag ke Ready ditolak dengan notice. Tombol "Tandai Ready" di panel memakai `aria-disabled` dengan alasan terhubung lewat `aria-describedby`, supaya tetap bisa difokus.
- Isi panel detail dipindah ke `TaskDetailBody` dan dipakai bersama oleh panel dan halaman `/task/:id`, supaya logika centang, label, pindah status, dan hapus hanya ada satu. Dialog revisi dan hapus dipindah ke body lewat `Teleport`.
- Bar aksi panel menempel di bawah area gulir. Bagian Label berada di bawah Deliverable supaya urutan Tab sama dengan urutan visual.
- Kategori "Task" di filter Audit Log memuat aksi sub task dan lampiran (`attachment_added` dan `attachment_deleted` sebelumnya tidak masuk kategori mana pun).
- `assignee_ids` di teks riwayat dipertahankan hanya untuk baris riwayat sebelum Sprint 7.
- Untuk Sprint 13: uji aturan centang (PIC, Manajer bukan PIC, Manajer dengan PIC Manajer lain, Staf lain, status selain On Progress, centang berulang yang idempoten, gate Ready dengan dan tanpa sub task), perbaiki test lama yang memakai `PICIDs`, dan migrasikan database test ke `000003` dan `000004`.

**Retro:**
- Macet: id sub task di skrip uji ditebak salah (5 dan 6, seharusnya 6 dan 7); tipe `Actor.Role` berbeda dari `sqlc.UserRole`; kolom nullable membuat sqlc memakai `pgtype.Int8`; terminal baru tanpa variabel `API` (curl `[000]`) dan skrip dijalankan dari folder yang salah; beberapa file frontend yang akan diubah tidak ada versi terbarunya sehingga diminta ulang.
- Lancar: (isi sendiri)
- Diubah untuk Sprint 9: (isi sendiri)

---

## Sprint 9: Arsip dan pencarian

**Tujuan:** task lama gampang dicari, termasuk lewat nama sub task. Sisa Sprint 6 versi lama.
**SRS:** FR-TASK-12, 13, 14, NFR-PERF-02, 03. **AC:** AC-09.

- [x] S9-01 `GET /api/archive`: task Completed tanpa batas 7 hari, ikut saklar lingkup Staf (Task saya atau Semua task).
- [x] S9-02 Kata kunci (pg_trgm, minimal 3 huruf) di ID, judul, deskripsi, tautan deliverable, nama pemohon, nama PIC, nama sub task, nama label, dan nama berkas lampiran.
- [x] S9-03 Filter PIC, divisi, label, urgency, dan rentang tanggal selesai. Urutan bawaan terakhir selesai di atas; pilihan terlama dan urgency berbobot.
- [x] S9-04 Halaman 25 task per halaman dengan total.
- [x] S9-05 Kata kunci juga di papan (`GET /api/board?q=`).
- [x] S9-06 Halaman Arsip `/arsip`: tabel di desktop, kartu di ponsel, filter dan kata kunci dan halaman di URL, tombol "Buka kembali" (lewat `MoveDialog`, hanya bila `move_to` memuat On Progress). Komponen `KeywordSearch` dipakai papan dan Arsip (cari saat Enter atau tombol Cari, galat per kolom bila kurang dari 3 huruf). Di papan, tautan "Cari ... juga di Arsip" saat ada kata kunci. Item Arsip di navigasi; item Task tetap tersorot di `/task/:id`; navigasi bawah ponsel tidak boleh membuat gulir horizontal. Teks Stitch "Semua proyek dan tugas studio..." dan "Tips Workflow" tidak dipakai. Kode lengkapnya sudah diberikan di chat 2026-11-10; cek dulu apakah sudah terpasang (`frontend/src/pages/ArchivePage.vue`).

**Hasil:** selesai 2026-11-13. S9-01 sampai S9-05 ter-commit 2026-11-10, S9-06 ter-commit 2026-11-13. Lolos `./scripts/check.sh`. Lewat browser (S9-06): kata kunci kurang dari 3 huruf ditolak per kolom, cari di papan dan tautan ke Arsip membawa kata kunci, Hapus filter ikut mengosongkan kata kunci, filter dan urutan dan halaman Arsip tersimpan di URL (refresh dan Back), Buka kembali lewat `MoveDialog`, panel detail dari Arsip, state kosong dan galat, saklar Task saya dan Semua task untuk Staf, kartu di 360 px dan tabel di 1280 px, Task tetap tersorot di `/task/:id`, tanpa gulir horizontal di 360, 768, dan 1280 px untuk akun Manajer (7 item navigasi). Lewat curl: 12 skenario Arsip (urutan bawaan dan terlama, kata kunci judul dan nama PIC, kata kunci kurang dari 3 huruf ditolak, `%` dicari apa adanya, rentang tanggal, rentang terbalik ditolak, halaman melewati akhir kembali ke 1, halaman 0 ditolak, Task saya) dan 6 skenario papan dengan kata kunci. Belum terbukti dengan data: pagination lebih dari 25 baris, filter divisi dan label, pencarian lewat label, lampiran, dan nama sub task (Sprint 13).

**Keputusan:**
- Arsip memuat semua task Completed. Papan tetap hanya Completed 7 hari terakhir.
- Kata kunci dicari sebagai satu potongan teks utuh dengan `ILIKE`. Karakter `%`, `_`, dan `\` di-escape supaya dicari apa adanya. Minimal 3 huruf (batas index trigram), maksimal 100.
- Aturan kata kunci satu fungsi (`keywordPattern`) untuk papan dan Arsip. Blok SQL-nya disalin di `ListBoardTasks` dan `ListArchiveTasks` karena sqlc tidak mendukung potongan query; keduanya harus diubah bersama.
- Total diambil dengan `count(*) OVER ()` dalam query yang sama. Halaman yang melewati akhir daftar mengembalikan halaman 1, dan respons menyebut halaman yang benar-benar dikembalikan.
- Rentang tanggal selesai dibaca sebagai tanggal WIB, keduanya inklusif.
- Arsip dan papan memakai `withMoveTo` yang sama untuk menghitung `move_to`.
- Di panel detail, galat saat memuat ulang diam-diam tidak lagi mengganti seluruh panel dengan tampilan galat.
- Buka kembali (Completed ke On Progress) hanya hak Manajer. Staf yang menjadi PIC pun tidak melihat tombolnya, karena tombol hanya tampil bila `move_to` dari server memuat On Progress.
- Kata kunci di URL yang kurang dari 3 huruf dibuang saat dibaca (`cleanKeyword`), bukan dikirim lalu ditolak server. Komponen `KeywordSearch` dipakai papan dan Arsip.
- Filter apa pun yang berubah di Arsip kembali ke halaman 1. Hapus filter di papan dan Arsip ikut mengosongkan kata kunci.
- Sorotan navigasi memakai `isCurrent(to)`: path sama persis atau diawali `to/`, karena `/task/:id` rute terpisah dari `/task`.
- Label navigasi bawah ponsel dipotong (`truncate`) supaya 7 item muat di 360 px. Ini pengaman sementara sampai navigasi ponsel dirancang ulang di Sprint 11.

**Retro:**
- Macet: kode S9-06 dari chat 2026-11-10 baru terpasang separuh (halaman dan komponen ada, rute, navigasi, dan template papan belum); empat file baru belum dirapikan Prettier sehingga `check.sh` gagal di format; `py` tidak tersedia di Git Bash yang dipakai Claude Code.
- Lancar: (isi sendiri)
- Diubah untuk Sprint 10: (isi sendiri)

---

## Sprint 10: Profil pengguna dan kartu profil di papan

**Tujuan:** setiap orang punya identitas yang terlihat: foto, nama, jabatan, divisi. Revisi 2 dari Manajer (bagian data dan kartu).
**Aturan baru:** area kiri papan yang kosong tidak diisi sidebar navigasi, tetapi kartu profil.

- [x] S10-01 Migrasi baru (nomor berikutnya setelah migrasi centang sub task): kolom jabatan (`job_title`) dan foto profil di tabel akun.
- [x] S10-02 Unggah foto: JPG atau PNG, maksimal 2 MB, tipe dicek dari isi. Backend memotong jadi persegi, mengecilkan ke ukuran tetap, lalu menyimpan ulang tanpa metadata EXIF (alasan: koordinat lokasi di foto ponsel tidak ikut tersimpan). Disimpan di `STORAGE_DIR/avatars/` dengan nama UUID.
- [x] S10-03 Endpoint: ubah nama dan jabatan sendiri, ganti dan hapus foto sendiri, ambil foto akun mana pun (wajib login, dengan ETag supaya tidak diunduh ulang).
- [x] S10-04 Aturan hak: tiap pengguna mengubah nama, jabatan, dan foto miliknya sendiri. Divisi dan peran tetap hanya Admin atau Manajer, ditampilkan baca-saja di kartu dengan keterangan siapa yang mengaturnya. Audit `profile_updated`.
- [x] S10-05 Kartu profil di area kiri papan (desktop): foto atau inisial, nama, jabatan, divisi, tombol "Ubah profil" yang membuka panel. Isian jabatan kosong tampil sebagai ajakan mengisi, bukan teks karangan.
- [x] S10-06 Ringkasan di kartu: jumlah task saya per status, dihitung dari data papan (tidak ada angka karangan). Klik angka menyaring papan.
- [x] S10-07 Ponsel: kartu menjadi bar ringkas di atas papan yang bisa dibuka, bukan kolom kiri.
- [x] S10-08 Foto atau inisial dipakai konsisten di top bar, chip PIC, kartu task, daftar Pengguna, dan Audit Log. Halaman Profil yang lama (ganti password) digabung ke panel ubah profil.
- [x] S10-09 State: foto gagal dimuat (kembali ke inisial), unggah berjalan, ukuran atau tipe ditolak, simpan gagal.

**Butuh dari Manajer:** apakah jabatan diisi sendiri oleh tiap orang atau hanya pengelola (usulan di atas: diisi sendiri). Sementara memakai usulan itu: diisi sendiri.

**Hasil:** kode S10-01 sampai S10-09 selesai 2026-11-13, menunggu uji Rifky. S10-01 sudah dimigrasi ke database dev (versi 5). Lolos `./scripts/check.sh`. Pemroses foto diuji lewat test sementara (dihapus lagi, test baru ditulis di Sprint 13): orientasi EXIF diterapkan, EXIF tidak ikut tersimpan, PNG transparan jadi putih, berkas teks ditolak, PNG 100 megapiksel ditolak dari header saja, berkas di atas 2 MB dan JPEG terpotong ditolak. Uji API belum dijalankan karena `DATABASE_URL` tidak tersedia untuk Claude Code; semuanya ada di `scripts/uji-sprint10.sh`.

**Keputusan:**
- Migrasi `000005_profile`: `users.job_title varchar(100)` dan `users.avatar_id uuid`, keduanya boleh NULL. Akun lama tetap sah tanpa diisi.
- Jabatan kosong disimpan NULL, bukan teks kosong. Constraint menolak teks kosong dan spasi di awal atau akhir, jadi backend wajib merapikan isian sebelum menyimpan.
- `avatar_id` dipakai sebagai nama berkas di `STORAGE_DIR/avatars/` dan sebagai ETag foto. Setiap unggahan memberi UUID baru, jadi foto lama tidak tertimpa di tempat dan cache browser otomatis basi.
- Foto: batas 2 MB (`http.MaxBytesReader`, dan `Content-Length` yang sudah terlalu besar ditolak sebelum isi dibaca). Tipe ditentukan dari isi: hanya JPEG dan PNG, dan format dari `image.DecodeConfig` harus sama dengan tanda di awal berkas. Gambar di atas 40 juta piksel ditolak sebelum decode penuh.
- Pemrosesan memakai `github.com/disintegration/imaging` v1.6.2: decode dengan `AutoOrientation(true)` (arah putar EXIF diterapkan dulu), `Fill` 256x256 dari tengah, diratakan di atas latar putih, simpan ulang JPEG kualitas 85. Simpan ulang inilah yang membuang EXIF. `golang.org/x/image` dinaikkan ke v0.48.0 karena versi 2019 bawaan imaging punya celah di decoder TIFF dan WebP.
- Berkas foto ditulis ke berkas sementara di `avatars/` lalu di-rename, baru database diubah. Kalau transaksi gagal, berkas baru dihapus. Berkas lama dihapus setelah transaksi berhasil; kalau gagal, hanya dicatat di log (pembersih berkas yatim di S12-07). Berkas sementara yang tertinggal di `avatars/` juga urusan S12-07.
- Galat foto: tipe atau isi rusak 415 (`unsupported_file`), terlalu besar 413 (`file_too_large`), tidak punya foto 404. Hapus foto saat tidak ada foto dianggap beres (204, tanpa audit).
- Endpoint profil di grup `app` (password sudah diganti): `PUT /api/me/profile`, `PUT /api/me/avatar`, `DELETE /api/me/avatar`, `GET /api/me/task-counts`, `GET /api/users/:id/avatar`. Rute `/me` tidak menerima id dari URL atau body; sasarannya selalu akun yang login.
- `GET /api/auth/me` (endpoint sesi yang sudah ada, bukan `/api/me`) kini membawa `job_title`, `avatar_id`, dan nama divisi. `avatar_id` juga ikut di `/api/people`, daftar Pengguna, PIC task dan sub task, serta aktor Audit Log.
- `GET /api/users/:id/avatar` memakai `Cache-Control: private, max-age=31536000, immutable` hanya bila `?v=` sama dengan `avatar_id` saat ini. Tanpa `?v=` yang cocok, `private, no-cache`, supaya URL tanpa versi tidak menyimpan foto lama. 304 ditangani `http.ServeContent` lewat ETag. Foto yang berkasnya hilang dari disk dibalas 404 dan dicatat di log.
- Validasi nama memakai aturan nama akun yang sudah ada (wajib, maksimal 100). Galat validasi akun kini membawa `field` di JSON (`name`, `job_title`), supaya form menaruh pesan di bawah kolomnya.
- Audit `profile_updated` masuk kategori "account". Untuk nama dan jabatan, `old_value` dan `new_value` berisi field yang berubah (jabatan kosong ditulis null). Untuk foto, hanya `new_value` `{"avatar": "diganti"}` atau `{"avatar": "dihapus"}`, karena berkas foto bukan nilai yang bisa dibandingkan. Simpan tanpa perubahan tidak dicatat.
- Papan memakai `meta.wide`, jadi AppShell memakai kontainer `max-w-[120rem]` khusus rute itu. Kartu profil menjadi kolom kiri 17rem bila kontainer papan (`@container/board`) selebar 80rem atau lebih, artinya mulai lebar jendela sekitar 1344 px. Di 1280 px ke bawah kartu menjadi bar ringkas di atas papan yang dibuka dengan tombol (`aria-expanded`).
- Angka di kartu (`GET /api/me/task-counts`) memakai `Board` dengan PIC akun sendiri tanpa filter lain, jadi sama dengan `scope=mine` dan Completed hanya 7 hari. Angka dimuat saat papan dibuka dan setelah setiap aksi, tidak saat filter berubah. Admin tidak pernah menjadi PIC, jadi bagian angka tidak ditampilkan untuk Admin.
- Klik angka: Staf mendapat `scope=mine`, Manajer mendapat `pic=<username sendiri>`, filter lain dan kata kunci dikosongkan, dan di layar sempit kolom status itu dipilih.
- Komponen `UserAvatar` (foto dengan inisial sebagai cadangan, `alt` kosong bila nama sudah tertulis di sebelahnya) dipakai di top bar, kartu dan bar profil, kartu task, chip dan daftar `PicPicker`, sub task di detail task, daftar Pengguna, dan Audit Log. Foto yang gagal dimuat kembali ke inisial.
- `ProfileForm` dipakai di panel "Ubah profil" papan dan di halaman `/profil`. Isi S10-08 di atas diubah: ganti password tetap di halaman `/profil`, tidak masuk panel. Item Profil di navigasi tetap ada sampai Sprint 11.
- Foto diunggah langsung saat dipilih (dengan progres), terpisah dari tombol Simpan profil. Tipe dan ukuran dicek juga di browser supaya berkas yang pasti ditolak tidak diunggah dulu; backend tetap memeriksa ulang.
- Perbaikan di luar rencana: `.gitignore` baris `storage/` mengabaikan paket kode `backend/internal/storage` (belum pernah ter-commit sejak Sprint 5, jadi repo di remote tidak bisa di-build). Diganti `/storage/`; folder data `backend/storage/` tetap diabaikan. Baris `!.vscode/settings.json` yang menempel dengan `backend/.data/` dipisah. Tag JSON `subtasks` di `updateTaskRequest` memakai kutip miring (ditandai `go vet`) dan diperbaiki; selama ini tetap jalan karena `encoding/json` tidak peka huruf besar-kecil.
- Untuk Sprint 13: uji `ProcessAvatar` (orientasi 1 sampai 8, PNG transparan, batas piksel, format tidak cocok), hak rute `/me`, ETag dan cache, audit `profile_updated`, kecocokan task-counts dengan papan, dan pembersihan berkas saat transaksi gagal.

**Retro:**
- Macet: `DATABASE_URL` tidak tersedia untuk Claude Code, jadi uji API tidak bisa dijalankan sendiri; paket `internal/storage` ternyata tidak pernah masuk git karena pola `.gitignore`; `py` tidak tersedia di Git Bash.
- Lancar: (isi sendiri)
- Diubah untuk Sprint 11: (isi sendiri)

---

## Sprint 11: Poles UI/UX

**Tujuan:** papan Kanban dan semua halaman terasa rapi, hidup, dan konsisten. Revisi 2 dari Manajer (bagian tampilan).
**Syarat mulai:** `docs/DESIGN.md` ditempel ke percakapan. Dial ENERGY, RHYTHM, MOTION ditetapkan di awal sprint beserta satu kalimat alasan, lalu dipegang sampai akhir. Tanpa DESIGN.md hasilnya ditandai draf.

- [ ] S11-01 Token desain dan satu motif identitas, diturunkan dari DESIGN.md dan brand NVM.
- [ ] S11-02 Papan: kartu, kolom, rasa drag-and-drop, transisi saat kartu pindah kolom, penanda terlambat, tampilan kartu dengan sub task dan PIC.
- [ ] S11-03 Panel detail, halaman task penuh, dan form task, termasuk daftar sub task dan centangnya.
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

## Sprint 13: Pengujian

**Tujuan:** semua yang dibangun di Sprint 7 sampai 12 teruji, dikerjakan Claude Code secara otomatis, lalu direview.
**Aturan:** Claude Code tidak boleh commit dan push. Commit dan push dijalankan Rifky.

- [x] S13-01 Kunci Claude Code lewat `.claude/settings.json` (dimajukan ke 2026-11-13, lihat Pembagian kerja).
- [x] S13-02 Tulis `CLAUDE.md` (dimajukan ke 2026-11-13). Perbarui bila ada aturan baru saat Sprint 13.
- [ ] S13-03 Prompt Claude Code: perbaiki test lama yang rusak karena Sprint 7, lalu tulis test backend untuk izin, transisi status, sub task dan PIC turunan, profil dan foto, arsip dan pencarian, lampiran, centang sub task beserta syarat Ready.
- [ ] S13-04 Test frontend untuk komponen dengan logika (pemilih PIC ganda, filter di URL, kartu profil).
- [ ] S13-05 Peta AC-01 sampai AC-15 ke test atau langkah uji manual, dengan hasil PASS atau FAIL.
- [ ] S13-06 Data dummy sekitar 2.000 task, ukur p95 dari dalam VM (bukan dari PC), cek `EXPLAIN ANALYZE` bahwa index pencarian terpakai. Target p95 di bawah 1 detik. Blok kata kunci di `ListArchiveTasks` dan `ListBoardTasks` memakai `OR` dengan banyak `EXISTS`, yang kemungkinan membuat index trigram tidak terpakai. Kalau lambat atau tidak memakai index, tulis ulang jadi `t.id IN (SELECT ... UNION SELECT ...)` di kedua query sekaligus.
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
| 3 | 2026-11-09 | Sub task bisa dicentang oleh PIC-nya saat On Progress; Ready butuh semua sub task tercentang; task penuh bisa dibuka di tab baru; di form, Deskripsi menggantikan posisi Lampiran brief, dan Lampiran brief masuk ke Detail tambahan | Sprint 8 |
| 4 | 2026-11-10 | Manajer boleh mencentang dan melepas centang sub task siapa pun, kecuali sub task yang dipegang Manajer lain | Sprint 8 |

SRS v1.0 belum memuat keempat revisi. SRS diperbarui sebelum UAT supaya checklist AC sesuai aplikasi.

## Menunggu jawaban Manajer

- Daftar label awal (nama dan warna) dan daftar pengguna awal (nama, username, peran, divisi).
- Apakah tim masih memakai berkas .ppt lama. Kalau tidak, hapus dari daftar izin.
- Konfirmasi kolom Completed di papan hanya 7 hari terakhir (AC-08 menyebut task Completed hilang dari papan aktif).
- Task tanpa sub task berarti tanpa PIC, hanya Manajer yang memindahkan: setuju?
- Jabatan diisi sendiri oleh tiap orang atau hanya pengelola.
- Sub task yang sudah dicentang lalu PIC-nya diganti Manajer saat ini tetap tercentang atas nama orang lama. Dibiarkan, atau centangnya dilepas otomatis saat PIC diganti?
- Aturan centang: Manajer boleh mencentang sub task siapa pun kecuali yang dipegang Manajer lain. Bagaimana dengan sub task tanpa PIC (sekarang boleh) dan sub task dengan PIC campuran Staf dan Manajer lain (sekarang ditolak)?

## Setelah go-live (belum dijadwalkan)

- HTTPS, lalu atribut cookie Secure, sebelum ada akses dari luar kantor.
- Fitur ekspor untuk Manajer (BR-11, belum ada di versi ini).
- Uji restore rutin tiap 3 bulan (NFR-REL-04).
- Upgrade npm 12 (ditunda).