# Rencana Sprint Sistem Internal NVM

Acuan: SRS Sistem Internal NVM v1.0, ERD `nvm_internal.dbml`, layar Stitch, dan DESIGN.md.
Stack: Go, PostgreSQL, Vue 3 dengan Tailwind CSS, Docker di VM Ubuntu (Hyper-V) pada PC kantor.

## Cara kerja

- Durasi sprint 1 minggu. Kalau minggu itu banyak kerjaan kantor lain, sprint boleh diperpanjang jadi 2 minggu, asal ditulis di catatan sprint.
- Awal sprint: pilih item dari backlog sprint itu. Item yang belum selesai pindah ke sprint berikutnya, jangan dipaksa selesai.
- Akhir sprint: demo ke Manajer di LAN kantor (mulai Sprint 2), lalu retro singkat: apa yang lancar, apa yang macet, apa yang diubah.
- Catatan harian cukup satu baris di bagian "Log" tiap sprint: hari ini mengerjakan apa, ada hambatan apa.
- Git: `main` selalu bisa jalan. Fitur dikerjakan di branch `feat/<nama>`, merge setelah lolos DoD.

## Definition of Done (berlaku untuk semua item)

- Aturan hak akses dicek di backend, bukan cuma disembunyikan di frontend (FR-AUTH-03).
- Ada test untuk aturan bisnisnya (minimal test handler/service di Go untuk izin dan transisi status).
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
| 6 | Arsip dan pencarian | Arsip dengan pencarian, filter, urut, pagination |
| 7 | Migrasi dan operasional | Data Excel masuk, backup, autostart, uji restore |
| 8 | UAT dan go-live | Semua AC-01 sampai AC-15 lolos, sistem dipakai tim |

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

- [ ] S1-01 `POST /api/auth/login` dengan username dan password. Hash password memakai bcrypt atau argon2id.
- [ ] S1-02 JWT 30 menit di cookie HttpOnly dan SameSite=Lax, berisi `user_id`, `role`, `token_version`. Atribut Secure dimatikan lewat env selama masih HTTP.
- [ ] S1-03 Middleware auth: tolak token yang `token_version`-nya tidak sama dengan database atau akun yang nonaktif (FR-AUTH-08).
- [ ] S1-04 Idle timeout: token diperbarui saat ada request, kedaluwarsa setelah 30 menit tanpa aktivitas.
- [ ] S1-05 Lockout progresif: 5 kali gagal dikunci 5 menit, lalu 15, lalu 30 dan seterusnya 30. Reset saat login berhasil.
- [ ] S1-06 Rate limit per IP di endpoint login (NFR-SEC-04).
- [ ] S1-07 Pesan login gagal tidak membedakan "username tidak ada" dan "password salah".
- [ ] S1-08 Audit: `login_success`, `login_failed` (user_id kosong bila username tidak terdaftar), `account_locked`.
- [ ] S1-09 Perlindungan CSRF untuk request yang mengubah data (token double-submit atau cek header Origin).
- [ ] S1-10 `POST /api/auth/logout` dan `GET /api/auth/me`.
- [ ] S1-11 Ganti password sendiri dengan password lama. Login dengan `must_change_password` langsung diarahkan ke halaman ganti password.
- [ ] S1-12 Halaman Masuk sesuai Stitch, termasuk state akun dikunci (dengan sisa waktu) dan sesi berakhir.
- [ ] S1-13 Route guard di Vue: halaman selain login butuh sesi, `http://<IP>/` membuka login.

**Selesai kalau:** skenario AC-01 dan AC-02 lolos manual dan lewat test Go.

---
**Keputusan:**
- Logout mengakhiri sesi di semua perangkat (token_version naik), karena skema tidak punya tabel sesi.
- Akun terkunci menampilkan sisa waktu walaupun ini menandakan username ada; diterima karena sistem hanya di LAN.

## Sprint 2: Akun, peran, divisi

**Tujuan:** Admin dan Manajer bisa mengelola orang tanpa menyentuh database.
**SRS:** FR-AUTH-02, 03, 04, 10, NFR-SEC-03, 11. **AC:** AC-03, AC-04 (bagian akun).

- [ ] S2-01 Middleware RBAC: `RequireRole(admin, manager)` dan seterusnya. Manajer memegang semua hak Admin.
- [ ] S2-02 CRUD akun: buat, ubah nama, peran, divisi, nonaktifkan, aktifkan, reset password. Nonaktif dan reset menaikkan `token_version`.
- [ ] S2-03 Cegah akun mengunci dirinya sendiri: tidak bisa menonaktifkan diri sendiri atau menghapus peran pengelola terakhir.
- [ ] S2-04 CRUD divisi (nama, aktif/nonaktif). Divisi yang masih dipakai tidak bisa dihapus, hanya dinonaktifkan.
- [ ] S2-05 Audit untuk perubahan akun, peran, dan divisi, dengan `old_value` dan `new_value`.
- [ ] S2-06 Halaman Pengguna, Pengaturan (bagian divisi), Profil (ganti password), dan Audit Log (hanya baca, dengan filter aksi dan tanggal).
- [ ] S2-07 Menu ikut peran: Staf tidak melihat Pengguna, Pengaturan, Audit Log. Backend tetap menolak dengan 403 kalau URL-nya dibuka langsung.

**Demo pertama ke Manajer:** login, buat akun Staf, login sebagai Staf, cek menu.
**Butuh dari Manajer:** daftar pengguna awal (nama, username, peran, divisi), termasuk setiap anggota Creative.

---

## Sprint 3: Task inti dan label

**Tujuan:** siklus hidup task benar di backend sebelum ada papan.
**SRS:** FR-TASK-01 sampai 07, 10, 11, 16, 17, 19, 21, 22, BR-01 sampai 05, 07. **AC:** AC-04 (bagian task), AC-08, AC-14.

- [ ] S3-01 `POST /api/tasks` khusus Manajer. ID T-nnnn dari `task_number_seq`, status awal Requested, wajib judul, minimal satu PIC, urgency.
- [ ] S3-02 Divisi task diisi dari divisi PIC pertama.
- [ ] S3-03 Ubah task (Manajer), soft delete (Manajer, tercatat audit).
- [ ] S3-04 Endpoint transisi status dengan tabel aturan:
  - PIC: Requested ke On Progress, On Progress ke Ready, Ready ke On Progress (wajib catatan).
  - Manajer: semua di atas, Ready ke Completed, Completed ke On Progress (buka kembali, `solved_date` dikosongkan).
  - Admin: tidak ada.
- [ ] S3-05 `solved_date` otomatis saat Completed. Setiap transisi tercatat sebagai `task_status_changed`.
- [ ] S3-06 Optimistic locking lewat kolom `version`: request membawa versi, beda versi dibalas 409 (persiapan AC-10).
- [ ] S3-07 Staf boleh membaca semua task, tetapi semua aksi pada task yang bukan miliknya dibalas 403 (NFR-SEC-03).
- [ ] S3-08 Label: CRUD oleh Manajer/Admin (nama, warna dari palet tetap, divisi opsional). Staf memasang label pada task miliknya, hanya label divisinya atau label untuk semua divisi.
- [ ] S3-09 `GET /api/tasks/:id` berisi detail, PIC, label, dan riwayat (terbaru di atas).
- [ ] S3-10 Dialog Buat task dan panel detail task di frontend (sementara di halaman daftar biasa).
- [ ] S3-11 Pengaturan bagian label.

**Selesai kalau:** test Go untuk setiap baris tabel transisi, termasuk yang harus ditolak, semuanya lolos.
**Butuh dari Manajer:** daftar label awal (nama dan warna).

---

## Sprint 4: Papan Kanban

**Tujuan:** layar utama yang dipakai tiap hari.
**SRS:** FR-TASK-08, 09, 18. **AC:** AC-05, AC-06, AC-10, AC-12.

- [ ] S4-01 `GET /api/board` dengan parameter `scope` (mine/all), `pic`, `division`, `label`, `urgency`, `due`. Jumlah per status ikut lingkup.
- [ ] S4-02 Papan desktop: 4 kolom, drag-and-drop hanya untuk kartu yang boleh dipindah oleh pengguna itu.
- [ ] S4-03 Tombol "Pindahkan" di kartu sebagai pengganti drag (layar sentuh dan keyboard).
- [ ] S4-04 Mobile: tab per status, bukan 4 kolom yang dikecilkan.
- [ ] S4-05 Saklar Staf "Task saya | Semua task", default Task saya, reset setiap login, state di URL (`?scope=all&pic=staf.b`).
- [ ] S4-06 Filter PIC: tersembunyi di mode Task saya, muncul di Semua task. Memilih orang lain pindah otomatis ke Semua task, memilih diri sendiri saja kembali ke Task saya. Manajer tanpa saklar, default "Semua orang".
- [ ] S4-07 Kartu dan detail task orang lain: tanpa tombol aksi, baris "Kamu hanya bisa melihat task ini."
- [ ] S4-08 Tampilan kosong dengan tombol "Lihat semua task".
- [ ] S4-09 Dialog revisi saat Ready dikembalikan ke On Progress.
- [ ] S4-10 Konflik 409: tampilkan pesan bahwa task sudah diubah orang lain, lalu muat ulang kartu.
- [ ] S4-11 Penanda terlambat bila tenggat lewat dan status belum Completed.

**Demo:** satu sesi bareng Manajer dan satu Staf memindahkan task yang sama dari dua perangkat.

---

## Sprint 5: Lampiran

**Tujuan:** brief dan hasil kerja bisa diunggah dengan aman.
**SRS:** FR-TASK-15, 23, NFR-SEC-09, NFR-PERF-04. **AC:** AC-07, AC-15.

- [ ] S5-01 `POST /api/tasks/:id/attachments` dengan `kind` brief (Manajer) atau deliverable (PIC atau Manajer).
- [ ] S5-02 Validasi: allowlist PDF, CSV, TXT, DOC/DOCX, XLS/XLSX, PPT/PPTX, JPG/JPEG, PNG. Tipe dicek dari isi berkas, bukan ekstensi saja. Tolak berkas makro (DOCM, XLSM, PPTM) dan executable. Batas 50 MB di backend dan di Nginx.
- [ ] S5-03 Simpan di disk dengan nama UUID, di luar folder yang disajikan web. Simpan checksum SHA-256.
- [ ] S5-04 Unduh lewat endpoint yang cek izin, dengan `Content-Disposition: attachment` dan nama asli.
- [ ] S5-05 Hapus lampiran (soft delete) dan audit `attachment_added` / `attachment_deleted`.
- [ ] S5-06 Frontend: area tarik berkas, tombol pilih berkas (di ponsel menawarkan kamera atau galeri), tempel screenshot dengan Ctrl+V di dialog Buat task dan Ubah task.
- [ ] S5-07 Progres unggah dan pesan galat tipe atau ukuran.
- [ ] S5-08 Tautan deliverable hanya menerima `http` dan `https` (NFR-SEC-06).

---

## Sprint 6: Arsip dan pencarian

**Tujuan:** task lama gampang dicari.
**SRS:** FR-TASK-12, 13, 14, NFR-PERF-02, 03. **AC:** AC-09.

- [ ] S6-01 `GET /api/archive` untuk task Completed, ikut saklar lingkup Staf.
- [ ] S6-02 Pencarian kata kunci ke kolom teks task (judul, deskripsi, nama label, nama PIC). Usulan: kolom `tsvector` atau `pg_trgm` dengan index, supaya tidak memindai seluruh tabel.
- [ ] S6-03 Filter dan urut: PIC, divisi, urgency (bobot Urgent, High, Medium, Low), label, rentang tanggal.
- [ ] S6-04 Pagination berbasis halaman atau cursor.
- [ ] S6-05 Halaman Arsip desktop (tabel) dan mobile (daftar kartu), dengan tombol buka kembali untuk Manajer.
- [ ] S6-06 Ukur respons dengan data dummy sekitar 2.000 task, target p95 di bawah 1 detik.

---

## Sprint 7: Migrasi dan operasional

**Tujuan:** data lama masuk dan server siap jalan terus di PC kantor.
**SRS:** FR-TASK-20, BR-09, NFR-SEC-07, 08, NFR-REL-01 sampai 05, NFR-RET-03. **AC:** AC-11, AC-13.

Migrasi:
- [ ] S7-01 Skrip impor (CLI Go) dari sheet Task Tracker dan Completed Task di `NVM - Marketing & Sales.xlsx`.
- [ ] S7-02 ID asli dipertahankan, `is_migrated = true`. Nama di Request By dan Solved By dipetakan ke username lewat file CSV pemetaan. Nilai "Creative" dipetakan ke akun anggota divisi Creative sesuai pemetaan.
- [ ] S7-02b Task Completed di Excel yang tidak punya Solved Date akan ditolak CHECK database. Putuskan saat dry-run: isi dari tanggal lain yang ada di baris itu, atau tandai untuk diisi manual.
- [ ] S7-03 Mode `--dry-run` yang menghasilkan laporan: baris terbaca, baris gagal beserta alasannya.
- [ ] S7-04 Setelah impor, `setval` sequence ke ID terbesar supaya task baru tidak bentrok.
- [ ] S7-05 Halaman atau laporan verifikasi untuk Admin dan Manajer.

Operasional:
- [ ] S7-06 VM Ubuntu di Hyper-V, Docker, `docker-compose.prod.yml` (PostgreSQL, backend, Nginx yang menyajikan build frontend di port 80).
- [ ] S7-07 Autostart: VM menyala otomatis saat Windows menyala, container `restart: unless-stopped`.
- [ ] S7-08 Port database tidak dibuka keluar VM. Firewall hanya izinkan port 80 dari subnet kantor.
- [ ] S7-09 Backup penuh harian (`pg_dump` atau `pg_basebackup`) plus arsip WAL untuk RPO 15 menit, termasuk folder lampiran.
- [ ] S7-10 Salin backup ke hard disk eksternal, PC IT, dan laptop IT (salinan laptop dienkripsi).
- [ ] S7-11 Job harian yang menghapus audit log login yang lebih tua dari 1 tahun.
- [ ] S7-12 Uji restore pertama ke database kosong, catat waktunya (target di bawah 4 jam).

**Butuh dari Manajer:** pemetaan nama di Excel ke username akun.

---

## Sprint 8: UAT dan go-live

**Tujuan:** tim memakai sistem ini, bukan Excel.

- [ ] S8-01 Jalankan checklist AC-01 sampai AC-15 bersama Manajer, catat hasilnya PASS atau FAIL.
- [ ] S8-02 Perbaiki temuan UAT. Temuan yang bukan bug masuk backlog setelah go-live.
- [ ] S8-03 Cek keamanan dasar: semua endpoint diuji dengan akun Staf dan Admin untuk aksi yang harus ditolak.
- [ ] S8-04 Cek aksesibilitas: keyboard saja, kontras, tap target, di ponsel tim yang sebenarnya.
- [ ] S8-05 Migrasi final dari Excel versi terakhir, lalu Excel dibekukan (hanya baca).
- [ ] S8-06 Buat akun semua anggota tim, bagikan username dan password awal secara langsung.
- [ ] S8-07 Panduan singkat satu halaman: cara login, cara memindahkan task, cara unggah.
- [ ] S8-08 Catatan operasional di README: cara backup, cara restore, cara update versi.

**Selesai kalau:** semua AC PASS dan tim sudah memakai sistem minimal satu minggu tanpa kembali ke Excel.

---

## Setelah go-live (belum dijadwalkan)

- HTTPS, lalu atribut cookie Secure, sebelum ada akses dari luar kantor.
- Fitur ekspor untuk Manajer (BR-11, belum ada di versi ini).
- Uji restore rutin tiap 3 bulan (NFR-REL-04).