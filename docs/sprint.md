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
| 11b | Identitas brand NVM | Monokrom NVM dengan aksen krem, mode terang dan gelap, D-DIN, logo resmi |
| 11c | Satu kerangka tata letak | Top bar dan tepi kiri sama di semua halaman, kepala halaman dan toolbar filter bersama |
| 11d | PIC task dan sub task mandiri | Manajer memilih PIC di task, PIC menyusun sendiri sub task-nya |
| 11e | Sub task ditugaskan | PIC membagi sub task ke PIC task lain, satu sub task satu PIC |
| 11f | Isi sub task dan aturan masuk | Sub task punya deskripsi dan lampiran sendiri; password 6, sesi 9 jam |
| 11g | Tautan hasil sub task | Sub task punya tautan deliverable; pembuat sub task ikut boleh mencentang |
| 11h | Acuan dan pratinjau sub task | Tautan acuan diisi saat membuat; berkas sub task bisa dipratinjau |
| 11i | PIC sub task dari semua akun | Sub task bisa dititipkan ke siapa pun, walau bukan PIC task |
| 11j | Manajer membagi, bukan menerima | Sub task hanya bisa ditugaskan ke Staf (dibatalkan revisi 13) |
| 11k | Manajer kembali boleh menjadi PIC sub task | Daftar "Dikerjakan" memuat Staf dan Manajer aktif |
| 11l | Popover profil | Klik foto atau nama orang membuka kartu profil kecil |
| 11m | Filter PIC ikut PIC sub task | Filter PIC, Arsip, dan Task saya memuat task yang sub task-nya dipegang orang itu |
| 11n | Form Buat task dan tenggat sub task | Sub task punya tenggat; form Buat task mengisi sub task lengkap dalam satu simpan |
| 11o | PIC sub task memindahkan task, form tambah dilipat | PIC sub task bisa memulai task dan menandai Ready; panel detail tidak langsung panjang |
| 11p | Catatan hasil sub task | Detail sub task dibagi Arahan dan Hasil; PIC sub task menulis catatan hasil |
| 11q | Task berulang | Task harian, mingguan, atau bulanan muncul sendiri sebagai task baru |
| 11r | Filter Hari ini dan istilah Deadline | Filter deadline hari ini di papan; kata Tenggat diganti Deadline |
| 11s | Aktivitas Harian | Tabel aktivitas per jam di papan task, halaman Aktivitas, dan ekspor Excel |
| 11t | PIC sub task di kartu dan revisi oleh PIC sub task | Kartu selalu menampilkan PIC sub task; PIC sub task mengembalikan Ready ke On Progress dengan membuka lagi sub task miliknya |
| 11u | Halaman Status hanya untuk Admin | Menu dan rute Status hanya untuk Admin; peran lain melihat 404 |
| 11v | Lampiran Aktivitas Harian | Berkas per jam di papan dan di halaman Aktivitas; backup storage jadi snapshot inkremental |
| 11w | Salinan ke Google Drive | Job malam menyalin berkas ke Drive akun tim dengan struktur yang dibaca orang |
| 11x | Panel Buat task: dua bug pemilih berkas dan halaman penuh | Batal memilih berkas tidak menutup panel, panel tidak melompat, dan rute /task/baru |
| 11y | Semua anggota membuat task, To Do pribadi, dan perbaikan task berulang | Staf boleh membuat task; To Do pribadi di papan; detail task berulang menampilkan kemunculan dan sub task templat |
| 11z | Sub task otomatis masuk templat seri | Perubahan sub task di kemunculan terbaru seri aktif langsung dipakai kemunculan berikutnya |
| 12a | Halaman Timeline | Kalender bersama launching, produksi, dan kehadiran, plus Important Notes; hanya Manajer yang menulis |
| 12b | Hapus task berulang dan hak PIC atas sub task | Dialog hapus memilih nasib seri, pengaman ticker, PIC task mengatur semua sub task di task-nya |
| 12d | Hak centang sub task, Output beberapa tautan, tempel dan seret lampiran | Centang untuk PIC sub task, PIC task, pembuat task, dan Manajer; setiap URL di Output Aktivitas jadi tombol; Ctrl+V dan seret-lepas di lampiran task dan sub task |
| 12e | Aktivitas mundur 2 hari kerja dan akses tambahan per pengguna | Staf mengisi aktivitas sampai 2 hari kerja ke belakang; akses Kelola Timeline dan Kelola Important Notes per pengguna; alasan Kehadiran terbuka untuk semua; Timeline jadi halaman awal; kolom Completed maksimal 10 |
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

**Hasil:** kode S10-01 sampai S10-09 selesai 2026-11-13; uji otomatis lolos, menunggu uji manual Rifky dengan foto asli dari ponsel. S10-01 sudah dimigrasi ke database dev (versi 5). Lolos `./scripts/check.sh`.
- Pemroses foto diuji lewat test sementara (dihapus lagi, test baru ditulis di Sprint 13): orientasi EXIF diterapkan, EXIF tidak ikut tersimpan, PNG transparan jadi putih, berkas teks ditolak, PNG 100 megapiksel ditolak dari header saja, berkas di atas 2 MB dan JPEG terpotong ditolak.
- API: `scripts/uji-sprint10.sh` lolos 26 dari 26 (profil, jabatan kosong jadi null, batas 100 karakter, galat per kolom, unggah JPEG dan PNG, hasil 256x256 tegak tanpa EXIF, header foto, 304, 401 tanpa login, 415, 413, hapus lalu 404, empat jalur ubah profil orang lain ditolak, task-counts cocok dengan papan untuk Staf dan Manajer, audit).
- Browser: Playwright dengan Edge (di folder sementara di luar repo, tidak ada dependensi baru), sesi dari cookie `dev-login.sh`. Lolos 21 kasus: kartu kiri di 1920 px, ajakan jabatan, ubah profil, galat per kolom, foto EXIF orientasi 6 tegak di halaman, tipe dan ukuran ditolak tanpa mengunggah, foto baru dan foto terhapus langsung terlihat di kartu profil, top bar, kartu task, detail task, dan Pengguna tanpa refresh, klik angka untuk Manajer dan Staf, angka tidak dimuat saat filter berubah tetapi dimuat setelah aksi, bar di 768 dan 360 px, keyboard penuh di panel, chip PIC dan Audit Log, berkas foto hilang tampil sebagai inisial, Admin tanpa angka, dan tanpa gulir horizontal di 75 kombinasi (Admin, Manajer, Staf x 360, 768, 1024, 1280, 1920 px x 5 halaman). Tidak ada galat console selain 404 gambar yang memang disengaja di uji berkas hilang.
- Belum diuji otomatis: foto asli dari kamera ponsel, pembaca layar sungguhan, dan form login (sesi uji memakai cookie).
- Setelah uji 2026-11-13, database dev pindah ke VM `nvm-server` 192.168.68.69. Semua hasil uji di atas dijalankan di VM lama (192.168.68.144). Uji foto ponsel asli masih menunggu dan dijalankan di VM baru.

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
- Hasil review dan uji browser (diperbaiki 2026-11-13):
  - Top bar meluap di 768 px untuk Admin dan Manajer (7 item navigasi ditambah foto, nama, dan Keluar; `scrollWidth` 854). Di lebar `md` nama dan peran disembunyikan secara visual (`sr-only lg:not-sr-only`, tetap dibaca pembaca layar), teks navigasi `text-sm` dengan `px-2`, jarak header `gap-4`, dan nama aplikasi tidak boleh terlipat. Mulai `lg` tampilan kembali seperti semula.
  - Setelah foto diganti atau dihapus, kartu task masih memakai foto lama karena hanya data sesi yang dimuat ulang. `ProfileForm` kini memberi tahu induknya setelah unggah dan hapus foto, dan papan memuat ulang datanya.
  - Permintaan angka kartu profil yang lama kini dibatalkan (`AbortController`), supaya jawaban yang terlambat tidak menimpa angka terbaru.
  - Galat per kolom di form profil hilang begitu kolomnya diketik ulang, bukan menunggu Simpan berikutnya.
- Untuk Sprint 13: uji `ProcessAvatar` (orientasi 1 sampai 8, PNG transparan, batas piksel, format tidak cocok), hak rute `/me`, ETag dan cache, audit `profile_updated`, kecocokan task-counts dengan papan, dan pembersihan berkas saat transaksi gagal.

**Retro:**
- Macet: `DATABASE_URL` tidak tersedia untuk Claude Code di putaran pertama, jadi uji API baru jalan setelah Rifky menyalakan backend; paket `internal/storage` ternyata tidak pernah masuk git karena pola `.gitignore`; `py` tidak tersedia di Git Bash; curl di MSYS2 UCRT64 adalah program Windows dan tidak membaca path `/tmp/...` di argumen `-F` yang mengandung `;` (skrip uji memakai `cygpath -m`); memuat `backend/.env` untuk login Playwright ditolak, jadi sesi uji memakai cookie dari `dev-login.sh`; beberapa kegagalan Playwright pertama ternyata salah skrip (menunggu `networkidle` sebelum permintaan dimulai, cache foto `immutable`, Tab ke UI browser di `<dialog>`).
- Lancar: (isi sendiri)
- Diubah untuk Sprint 11: (isi sendiri)

---

## Sprint 11: Poles UI/UX

**Tujuan:** papan Kanban dan semua halaman terasa rapi, hidup, dan konsisten. Revisi 2 dari Manajer (bagian tampilan).
**Syarat mulai:** `docs/DESIGN.md` ditempel ke percakapan. Dial ENERGY, RHYTHM, MOTION ditetapkan di awal sprint beserta satu kalimat alasan, lalu dipegang sampai akhir. Tanpa DESIGN.md hasilnya ditandai draf.

- [x] S11-01 Token desain dan satu motif identitas, diturunkan dari DESIGN.md dan brand NVM.
- [x] S11-02 Papan: kartu, kolom, rasa drag-and-drop, transisi saat kartu pindah kolom, penanda terlambat, tampilan kartu dengan sub task dan PIC.
- [x] S11-03 Panel detail, halaman task penuh, dan form task, termasuk daftar sub task dan centangnya.
- [x] S11-04 Arsip, Pengguna, Pengaturan, Audit Log, dan kartu profil memakai bahasa visual yang sama.
- [x] S11-05 State kosong, loading (kerangka sesuai bentuk konten), dan galat yang konsisten. Ilustrasi hanya SVG buatan sendiri.
- [x] S11-06 Animasi: tiap animasi punya tujuan UX (umpan balik, orientasi, atau hubungan sebab akibat), tanpa loop. Versi diam untuk `prefers-reduced-motion`.
- [ ] S11-07 Dipindah ke S13-09 (2026-11-16): audit visual dan aksesibilitas ditunda ke Sprint 13 bersama hasil Sprint 10 dan tombol Dashboard.

**Selesai kalau:** `./scripts/check.sh` lolos dan semua halaman dicek di rentang lebar di atas.

**Keputusan (ditetapkan 2026-11-16, sebelum mulai):**
- Dial ENERGY 2 (seimbang): aplikasi dipakai seharian dan DESIGN.md meminta suasana tenang ala lab arsip, jadi kesan rapi datang dari tipografi, kerapian, dan detail interaksi, bukan warna atau efek ramai.
- Dial RHYTHM 2 (sedikit variasi): papan butuh struktur seragam supaya mudah dipindai; variasi hanya di hierarki (kartu profil, kepala kolom, panel detail).
- Dial MOTION 2 (transisi dan umpan balik): animasi hanya untuk umpan balik dan orientasi, tanpa parallax atau loop.
- Motif identitas "label spesimen": ID task dan kode lain tampil sebagai label kecil IBM Plex Mono 13px dengan latar `#E6E1D7` dan garis 1px, sama di kartu, panel, Arsip, dan Audit Log.
- Satu aksen: olive `#3F5A3A`. Radius 6px untuk kartu, input, tombol, dan dialog; chip 4px; bentuk bulat hanya untuk avatar. Bayangan hanya untuk menu, dialog, dan kartu yang sedang di-drag.
- Larangan dari DESIGN.md dan AntiSlop: tanpa glassmorphism, blur, glow, gradien biru-ungu, grid titik, ikon sparkle atau robot, dan emoji.
- Header YAML di atas `docs/DESIGN.md` adalah palet otomatis Stitch dan tidak dipakai. Yang berlaku nilai di bagian teks: kanvas `#F4F1EA`, surface `#FFFFFF`, teks `#1F1D1A`, muted `#635C52`, garis `#8F8678`, aksen `#3F5A3A`, chip status, urgency `#9C3A22` dan `#8A5A12`. Nilai itu sudah sama dengan token di `style.css`, jadi S11-01 menyamakan yang belum sama, bukan mengganti palet.
- Label urgency tetap Low, Medium, High, Urgent (mengikuti SRS dan Excel lama), bukan Mendesak, Tinggi, Sedang, Rendah seperti di DESIGN.md.
- Contoh isi di DESIGN.md ("Contoh: ...", "Staf A", "Manajer A") hanya untuk mockup dan tidak dipakai sebagai teks aplikasi.
- Font IBM Plex Sans dan Mono di-host sendiri lewat `@fontsource` (hanya bobot yang dipakai), bukan Google Fonts, karena aplikasi tidak boleh bergantung ke server pihak ketiga untuk tampil benar (semula alasannya LAN kantor; sejak revisi 5 aplikasi publik, tetapi alasan ini tetap berlaku).
- Selama pengerjaan:
  - Skala teks mengikuti DESIGN.md: body 15px/24px, `text-xs` dinaikkan ke 13px (tidak ada teks di bawah itu), 18px dan 24px untuk judul. Kontras dihitung ulang dan semua pasangan teks lolos AA tanpa mengubah palet. Garis 1px label spesimen di atas latarnya 2.76:1, tetapi yang memisahkan label dari sekelilingnya adalah garis terhadap surface (3.59) dan canvas (3.18), dan isinya dibawa teks (12.90).
  - Mono hanya untuk ID dan kode (DESIGN.md). Angka jumlah di papan, kartu profil, dan sub task memakai Plex Sans `tabular-nums`.
  - Urgency High dan Urgent diberi penanda bentuk (persegi dan belah ketupat) di depan kata (`UrgencyLabel`), dipakai di kartu, detail, dan Arsip. Tenggat lewat ditandai ikon jam dan kata "Terlambat".
  - Drag: kartu terangkat (bayangan dan miring 1 derajat) hanya saat browser mengambil gambar drag-nya. Kolom sah bergaris putus-putus olive dengan teks "Lepas di sini untuk memindahkan". Kolom yang ditolak meredup, dan berubah merah saat disentuh sebelum dilepas. Kartu yang masuk kolom muncul dengan transisi 200 ms.
  - Navigasi ponsel: bar bawah berisi Task, Arsip, Profil, dan Lainnya. Menu Lainnya memakai pola disclosure (tombol `aria-expanded` dan daftar tautan biasa): Tab, Enter, Escape (fokus kembali ke tombol), tertutup saat fokus keluar atau pindah halaman. Isinya sisa halaman sesuai peran, nama akun, dan Keluar. Navigasi desktop memindahkan Task dan Arsip ke depan.
  - Kerangka memuat (`LoadingSkeleton`) dalam bentuk kolom papan, baris daftar, dan isi detail, sengaja diam (tanpa pulse). State galat dan kosong sudah seragam sebelum sprint ini, jadi tidak diubah. Tidak ada ilustrasi, karena pesan kosong yang ada sudah jelas.
  - Animasi 150 sampai 220 ms: notifikasi, panel masuk dari kanan, dialog dengan latar memudar, menu dari arah pemicunya, dan kartu masuk kolom. Satu aturan `prefers-reduced-motion` mematikan semuanya.
  - Bug yang ditemukan dan diperbaiki: aturan "garis isian olive saat fokus" di `@layer base` kalah oleh kelas utilitas `border-line`, sehingga tidak pernah berlaku. Dipindah ke luar layer, dan isian yang sedang galat (`aria-invalid`) tetap bergaris merah.

**Hasil:** S11-01 sampai S11-06 selesai 2026-11-16. `./scripts/check.sh` lolos di setiap langkah; uji browser ditunda ke Sprint 13 (S13-09), karena backend dan Vite tidak menyala selama sprint ini.

**Retro:**
- Macet: backend dan Vite tidak bisa dijangkau sepanjang sprint, jadi screenshot sebelum dan sesudah serta audit S11-07 tidak bisa dijalankan.
- Lancar: (isi sendiri)
- Diubah untuk Sprint 12: (isi sendiri)

---

## Sprint 11b: Identitas brand NVM

**Tujuan:** menyelesaikan revisi 2 (bagian tampilan) yang belum benar di Sprint 11: task tracker memakai identitas NVM, monokrom dengan satu aksen krem, mode terang dan gelap. Tidak ada fitur baru dan tidak ada perubahan backend.

- [x] S11b-01 Token warna kedua mode, font D-DIN dan JetBrains Mono, favicon, logo resmi, tombol mode gelap.
- [x] S11b-02 Login dan papan (dilihat Rifky di kedua mode, 2026-11-17).
- [x] S11b-03 Sisa halaman dan komponen mengikuti token kedua mode: panel detail, halaman task penuh, form task, dialog, Arsip, Pengguna, Pengaturan, Audit Log, Profil, Status, 404, navigasi, state memuat, kosong, dan galat.
- [x] S11b-04 `docs/DESIGN.md` ditulis ulang (menggantikan versi Stitch).

**Keputusan:**
- Dial ENERGY 2, RHYTHM 2, MOTION 2, dengan alasan yang sama dengan Sprint 11: aplikasi kerja yang dibaca berjam-jam, papan butuh struktur seragam, animasi hanya untuk umpan balik.
- Mode terang bawaan, mode gelap lewat tombol (top bar desktop, menu Lainnya ponsel), mengikuti `prefers-color-scheme` saat belum dipilih, disimpan di `localStorage` (`nvm-tema`, dengan try/catch). `public/tema.js` dimuat sinkron di `<head>` supaya tidak ada kilatan terang; CSP `script-src 'self'` mengizinkannya. Perubahan setelan sistem setelah halaman terbuka belum diikuti langsung (berlaku saat dimuat ulang).
- Warna gelap dari tema Shopify NVM. Kanvas terang `#f3f2ee` diambil dari latar ikon NVM. Nilai lengkap di `docs/DESIGN.md`.
- Tombol utama: tinta dengan teks krem di mode terang, krem dengan teks tinta di mode gelap, lewat token `accent` dan `on-accent` yang berganti per mode, jadi satu kelas berlaku di kedua mode. Label ID dan chip label memakai `tag` dan `tag-ink` yang tetap (krem dan tinta) di kedua mode. Chip label pastel selalu bertulisan tinta (kontras terendah 12.51).
- Token `mark` (tinta di terang, krem di gelap) untuk cincin fokus, garis terpilih, teks aktif, dan isi bar kemajuan, karena krem tidak terlihat di latar terang. `scrim` (hitam 55%) untuk latar dialog di kedua mode. `on-urgent` untuk teks tombol hapus.
- Logo: tidak ada logo resmi terpisah untuk latar terang dan gelap, dan tidak ada SVG. Dipakai `brand/nvm-logo.png` dari compro (putih di atas transparan) sebagai mask CSS yang diisi warna `ink`, cara yang sama dengan `BrandLogo.vue` compro. Di top bar lebar `md`, kata "Internal" hanya untuk pembaca layar supaya top bar tujuh item tetap muat.
- Font: D-DIN 400 dan 700 dari `/fonts` (`font-display: swap`, `OFL.txt` ikut tersaji); JetBrains Mono 400 dan 500 lewat `@fontsource` untuk ID, kode, dan angka hitungan. Paket IBM Plex dihapus. Tanggal dengan nama bulan tetap D-DIN (bukan angka tabular murni).
- Radius 4px (kartu, isian, tombol, dialog) dan 2px (chip); bulat hanya avatar. Bayangan hanya menu, dialog, dan drag, lebih pekat di mode gelap.
- Kerangka memuat memakai `line` 25% (sebelumnya krem, menyilaukan di mode gelap).
- Nginx: CSP tidak lagi butuh `data:` (favicon kosong diganti favicon asli); `/fonts/` di-cache 30 hari karena nama berkas tanpa hash.
- Kontras (terang / gelap), semua lolos AA: teks di kanvas 15.23 / 16.66, di surface 17.06 / 16.05; teks redup 5.73 sampai 6.42; teks tombol utama 13.31 / 13.31, hover 10.16 / 14.99; label ID 13.31; penanda di surface 17.06 / 14.97; garis isian 3.45 sampai 4.05; urgent 6.18 sampai 9.14; high 5.67 sampai 10.67; danger 6.81 sampai 9.56; chip status 7.21 sampai 10.25.

**Hasil:** kode selesai 2026-11-17, `./scripts/check.sh` lolos, build memuat D-DIN, JetBrains Mono, logo, favicon, dan `tema.js`. Menunggu lihat Rifky di kedua mode dan di lebar ponsel (login dan papan sudah dilihat). Audit browser otomatis (lebar, Tab, Escape, reduced motion, console) tetap di S13-09.

**Retro:**
- Macet: (isi sendiri)
- Lancar: (isi sendiri)
- Diubah untuk sprint berikutnya: (isi sendiri)

---

## Sprint 11c: Satu kerangka tata letak

**Tujuan:** semua halaman yang login terasa satu aplikasi. Masalah dari Rifky di produksi (1920 px): top bar dan logo berpindah posisi antar halaman (x sekitar 32 px di `/task`, sekitar 416 px di `/arsip`), papan rata kiri selebar layar sementara halaman lain ditengahkan, dan papan terasa kosong (banner kosong ganda, cari dan filter memakan tinggi, judul tanpa baris aksi). Hanya frontend.

- [x] S11c-01 Kerangka AppShell: top bar dan wadah konten selebar layar dengan satu sumber padding, tanpa `mx-auto`.
- [x] S11c-02 Komponen kepala halaman bersama (`PageHeader`) di semua halaman yang login.
- [x] S11c-03 Papan: banner kosong dihapus, toolbar cari dan filter, "Buat task" di slot aksi, batas kartu profil.
- [x] S11c-04 Arsip, Pengguna, dan Audit Log memakai toolbar yang sama (`FilterToolbar`).

**Keputusan (aturan kerangka):**
- Tepi kiri bersama: AppShell memakai satu konstanta padding (`gutter`, `px-4 md:px-8`) untuk top bar, baris galat logout, dan wadah konten. Top bar selalu selebar layar, jadi logo, menu, dan aksi kanan tidak berpindah antar halaman. `meta.wide` dihapus karena lebar konten sekarang ditentukan di AppShell.
- Lebar konten (revisi Rifky 2026-11-17, mengubah sebagian S11c-01): hanya papan (`/task`) memakai lebar penuh, karena empat kolom membutuhkannya. Halaman lain ditengahkan dengan `mx-auto max-w-6xl`, karena tabel dan form yang direntang sampai 1920 px menyisakan ruang kosong besar di kanan dan barisnya jadi sulit diikuti. (Revisi 21 menambah satu pengecualian, `/aktivitas`, dengan alasan yang sama seperti papan: jumlah kolomnya ditentukan data. Lihat Keputusan Sprint 11s.) Konsekuensinya tepi kiri judul di halaman non-papan tidak lagi sejajar dengan logo, dan itu diterima: yang mengganggu sebelumnya adalah top bar yang ikut berpindah, bukan konten yang ditengahkan.
- Kapan memakai `max-w`: hanya halaman yang isinya form atau teks (Profil `max-w-xl`, Status `max-w-xl`, Pengaturan `max-w-3xl`, 404 `max-w-xl`), selalu rata kiri pada tepi yang sama. Halaman tabel (Arsip, Pengguna, Audit Log), papan, dan halaman task penuh memakai lebar penuh wadah.
- Login dan Buat password baru tetap di layout tanpa AppShell (kartu di tengah). Buat password baru tidak dipindah ke AppShell karena menu navigasi di sana hanya akan mengembalikan pengguna ke halaman yang sama sampai password diganti.
- `PageHeader`: judul `text-2xl` tebal, deskripsi opsional, dan slot aksi di kanan (turun ke bawah judul di layar sempit). Dipakai di Papan, Arsip, Pengguna, Audit Log, Pengaturan, Profil, Status, halaman task penuh (judul berupa label ID lewat slot `title`), dan 404.
- `FilterToolbar`: kolom cari di kiri (slot `search`), filter sebaris di sebelahnya, lalu "Hapus filter" bila ada yang aktif. Kontrol memakai kelas bersama `filterControl` (`utils/ui.ts`), tinggi minimal 44 px. Label filter hanya untuk pembaca layar bila opsi pertama sudah menjelaskan ("Semua divisi" dan seterusnya). Kontrol yang opsinya tidak menjelaskan diri sendiri (tanggal dan urutan di Arsip) memakai label sebaris di depan kontrol (`inlineLabel`), supaya tetap terlihat tanpa menambah tinggi. Di bawah 768 px filter dilipat ke tombol "Filter (n)" dengan `aria-expanded`; n menghitung filter aktif tanpa kata kunci dan urutan.
- `KeywordSearch` dibuat padat: label dan petunjuk hanya untuk pembaca layar, isinya diringkas di placeholder ("Cari ID, judul, PIC, label, atau berkas"); galat kurang dari 3 huruf tetap terlihat. Warna placeholder memakai `ink-muted` supaya kontrasnya terjaga di kedua mode (5.73 sampai 6.42).
- Papan: banner "Belum ada task aktif" dan variannya dihapus; kolom kosong sudah memberi tahu sendiri. Bila filter atau pencarian membuat papan kosong, satu baris di atas kolom menyebutnya dengan tombol "Hapus filter". "Buat task" ada di slot aksi `PageHeader` dan tetap hanya untuk Manajer (peran dari sesi server).
- Kartu profil di kiri papan baru muncul bila kontainer papan selebar 90rem atau lebih (jendela sekitar 1504 px; sebelumnya 80rem). Di bawah itu, termasuk 1280 px, kartu menjadi bar ringkas di atas papan. Alasannya: di sekitar 80rem empat kolom hanya kebagian sekitar 230 px; di 90rem sekitar 280 px.

**Hasil:** kode selesai 2026-11-17, `./scripts/check.sh` lolos. Pemeriksaan kode: tidak ada `mx-auto` di wadah halaman, semua halaman yang login memakai `PageHeader`, padding top bar dan wadah konten dari satu konstanta di AppShell. Menunggu lihat Rifky di 360, 768, 1280, dan 1920 px pada kedua mode.

**Retro:**
- Macet: (isi sendiri)
- Lancar: (isi sendiri)
- Diubah untuk sprint berikutnya: (isi sendiri)

---

## Sprint 11d: PIC task dan sub task mandiri

**Tujuan:** revisi 6 Manajer. Manajer membuat task dan menentukan siapa PIC-nya. PIC lalu menyusun sendiri sub task yang ia kerjakan, dan Manajer boleh ikut menambah sub task. Ini membalik sebagian Sprint 7, yang memindahkan PIC dari task ke sub task.

- [x] S11d-01 Migrasi `000008`: tabel `task_assignees` kembali, diisi dari PIC sub task yang sudah ada supaya task lama tidak kehilangan pemiliknya. `down` membuang tabelnya.
- [x] S11d-02 Semua pembacaan PIC task (papan, Arsip, "Task saya", filter PIC, pencarian nama, divisi task) membaca `task_assignees`, bukan lagi gabungan PIC sub task.
- [x] S11d-03 Buat dan Ubah task: Manajer wajib memilih minimal satu PIC task. Pemilih PIC per sub task dihapus dari form.
- [x] S11d-04 `POST`, `PUT`, dan `DELETE /api/tasks/:id/subtasks` untuk PIC task dan Manajer.
- [x] S11d-05 Frontend: pemilih PIC pindah ke bagian atas form task; panel detail punya tambah sub task, ubah nama, dan hapus sesuai `can.add_subtask` dan `can_edit` dari server.

**Keputusan:**
- PIC hanya ada di task induk. Sub task tidak punya pemilih PIC. Baris di `subtask_assignees` sekarang berarti pemilik sub task, yaitu orang yang membuatnya, dan isinya paling banyak satu orang. Tabelnya dipertahankan supaya aturan centang yang lama tidak perlu ditulis ulang.
- Sub task yang dibuat lewat form Manajer tidak punya pemilik. Sub task seperti itu boleh dicentang PIC task mana pun, kalau tidak, pekerjaan titipan Manajer tidak bisa diselesaikan siapa pun dan task tertahan sebelum Ready.
- Yang boleh mengubah nama dan menghapus sub task: pembuatnya, dan Manajer. PIC lain di task yang sama tidak bisa menghapus pekerjaan orang.
- Menambah, mengubah nama, dan menghapus sub task hanya saat task berstatus Requested atau On Progress. Ditutup mulai Ready supaya pekerjaan tidak bertambah saat Manajer mereview, dan supaya syarat "semua sub task tercentang" tidak bisa digoyang setelah task lolos.
- Ketiga aksi itu menaikkan `version` task. Wajib, karena daftar sub task dari form Manajer menggantikan daftar lama secara utuh: tanpa kenaikan versi, form yang sudah terbuka akan menghapus sub task yang baru ditambahkan PIC.
- Divisi task mengikuti PIC task pertama, sesuai urutan pemasangan (`clock_timestamp`). Aturan "divisi mengikuti PIC pertama dari sub task pertama" (S7-06) tidak berlaku lagi.
- Kejadian audit baru `task_pics_changed`. `subtask_assignees_changed` tidak dihasilkan lagi, tetapi teksnya dipertahankan di frontend untuk riwayat lama.
- Task sekarang selalu punya PIC, jadi catatan Sprint 7 "task tanpa sub task berarti tanpa PIC" tidak berlaku lagi.

- Pratinjau lampiran mengambil berkas sebagai blob, jadi CSP Nginx produksi perlu `blob:` di `img-src` dan `frame-src`. Ketahuan di produksi, bukan di dev: server Vite tidak memasang CSP, jadi pratinjau jalan di dev dan gambar gagal dimuat di `novoidminds.xyz`. Blob hanya bisa dibuat origin sendiri, dan iframe PDF tetap `sandbox="allow-scripts"` tanpa `allow-same-origin`.

**Selesai kalau:** `./scripts/check.sh` lolos dan alurnya terbukti lewat curl: Manajer membuat task dengan PIC Staf tanpa sub task, Staf itu menambah sub task sendiri lalu mencentangnya, Staf lain ditolak, dan sub task buatan Manajer bisa dicentang PIC.

**Hasil:** kode selesai 2026-11-17, `./scripts/check.sh` lolos. Migrasi `000008` dijalankan di dev dan di produksi 2026-11-17, grants diulang (enam tabel, `task_assignees` masuk), image dibangun ulang, `/api/health` balas `ok`. Uji curl dan uji browser menunggu Rifky. Pratinjau lampiran sempat gagal di produksi karena CSP, diperbaiki dan dibangun ulang di hari yang sama; pratinjau gambar terbukti jalan di `novoidminds.xyz`.

## Sprint 11e: Sub task ditugaskan ke PIC lain

**Tujuan:** revisi 7 Manajer. Sub task bentuknya mirip task kecil di dalam task induk: punya nama dan punya satu PIC. PIC task mana pun boleh menyusun sub task dan menugaskannya ke PIC task yang lain, misalnya saat salah satu PIC bertindak sebagai supervisor yang membagi pekerjaan ke timnya.

- [x] S11e-01 Migrasi `000009`: kolom `created_by` di `task_subtasks`. Sejak sekarang pembuat dan yang mengerjakan bisa orang berbeda, jadi keduanya disimpan terpisah. Data lama diisi dari `subtask_assignees.assigned_by`.
- [x] S11e-02 `POST` dan `PUT /api/tasks/:id/subtasks` menerima `pic_id`. Kandidatnya hanya PIC task ini, divalidasi di backend. `pic_id` 0 berarti belum ditugaskan.
- [x] S11e-03 Hak ubah nama dan hapus sub task: pembuatnya, PIC-nya, dan Manajer. Dipakai service dan dikirim ke frontend lewat `can_edit`.
- [x] S11e-04 Manajer melepas seseorang dari PIC task: sub task yang ditugaskan ke dia ikut dilepas, jadi tidak ada sub task yang menunggu orang yang sudah tidak ada di task ini.
- [x] S11e-05 Frontend: tiap sub task jadi satu kartu berisi centang, nama, dan PIC-nya. Form tambah dan form ubah punya pilihan PIC. Sub task yang dikerjakan pengguna ini ditandai garis di tepi kiri.

**Keputusan:**
- Satu sub task satu PIC. Kalau pekerjaannya butuh dua orang, dibuat dua sub task, supaya saat belum tercentang selalu jelas siapa yang ditunggu.
- PIC sub task hanya boleh dipilih dari PIC task. Mau orang lain, Manajer menambahkannya dulu sebagai PIC task. Dengan begitu daftar PIC task tetap menjadi daftar lengkap siapa yang mengerjakan task itu, dan papan serta filter PIC tidak perlu membaca sub task.
- Sub task boleh dibiarkan tanpa PIC, dan yang begitu boleh dicentang PIC task mana pun. Aturan ini dipertahankan dari revisi 6, karena sub task dari form Manajer memang tidak punya PIC.
- Yang boleh mengubah nama dan menghapus sub task: pembuatnya, PIC-nya, dan Manajer. PIC boleh menghapus karena pekerjaan yang dibatalkan paling sering diketahui oleh yang mengerjakannya.
- Form task milik Manajer tetap tanpa pilihan PIC per sub task. Sub task dari form berarti belum ditugaskan, dan penugasannya dilakukan di panel detail. Kalau pilihan PIC ikut masuk ke form, daftar sub task dari form akan menimpa penugasan yang sudah dibuat PIC.
- Perubahan nama dan perubahan PIC dicatat sebagai satu kejadian `subtask_renamed`, dengan `pic_id` di nilai lama dan barunya. Teks riwayat memilih kalimatnya dari apa yang benar-benar berubah.

**Selesai kalau:** `./scripts/check.sh` lolos dan alurnya terbukti di browser: Manajer memilih tiga PIC, PIC pertama membuat sub task untuk PIC kedua, PIC kedua mencentangnya, PIC ketiga tidak bisa mencentang sub task itu, dan sub task tanpa PIC bisa dicentang siapa pun di antara mereka.

**Hasil:** kode selesai 2026-11-17, `./scripts/check.sh` lolos. Migrasi `000009` belum dijalankan di dev maupun produksi. Uji browser menunggu Rifky.

## Sprint 11f: Isi sub task dan aturan masuk

**Tujuan:** revisi 8 Manajer. Sub task perlu menunjukkan siapa yang membuatnya dan siapa yang mengerjakannya, serta menampung deskripsi dan berkasnya sendiri. Sekalian dua perubahan aturan masuk yang diminta Manajer: panjang minimal password dan umur sesi.

- [x] S11f-01 Migrasi `000010`: kolom `description` di `task_subtasks` dan kolom `subtask_id` di `task_attachments` (dengan index parsial). `subtask_id` NULL berarti lampiran task induk.
- [x] S11f-02 `POST` dan `PUT` sub task menerima `description` (maksimal 2000 karakter). Detail task mengirim `description` dan `created_by` per sub task.
- [x] S11f-03 `POST /api/tasks/:id/subtasks/:subtaskId/attachments` untuk lampiran sub task. Berkas task dan berkas sub task dibaca dalam satu query, lalu dipilah dari `subtask_id`.
- [x] S11f-04 Frontend: kartu sub task menampilkan centang, nama, PIC, dan pembuatnya dalam satu baris ringkas. Deskripsi dan berkas disembunyikan di balik tombol Detail, satu kartu terbuka sekaligus.
- [x] S11f-05 Password minimal 6 karakter (sebelumnya 12), `auth.SessionMaxAge` 9 jam (sebelumnya 12), dan idle timeout dihapus sehingga umur sesi jadi satu-satunya batas. Panduan tim, runbook, dan job pembersih sesi ikut diperbarui.

**Keputusan:**
- Lampiran sub task menumpang tabel `task_attachments` dengan `kind` deliverable, bukan jenis baru. Batas ukuran, daftar tipe yang diizinkan, pembersih berkas yatim, dan hak akses sudah berlaku di tabel itu, jadi tidak ada aturan kedua yang bisa menyimpang.
- Hak unggah dan hapus lampiran sub task sama dengan berkas deliverable task (PIC task dan Manajer), bukan hanya PIC sub task itu. Alasannya, berkas sering diunggah orang yang mereview atau membantu, dan pembatasan yang lebih ketat hanya memindahkan pekerjaan ke Manajer.
- Menghapus sub task ikut menghapus baris lampirannya lewat `ON DELETE CASCADE`. Berkas di disk dibersihkan job berkas yatim (S12-07), jadi tidak ada berkas yang hilang di tengah transaksi.
- Hanya satu sub task yang terbuka sekaligus di panel detail. Deskripsi dan daftar berkas membuat kartu tinggi, dan dua kartu terbuka berdampingan membuat daftar sulit dibaca.
- Form task milik Manajer tetap hanya berisi nama sub task. Deskripsi, PIC, dan berkas diisi di panel detail, supaya daftar dari form tidak menimpa isi yang sudah ditambahkan PIC.
- Password minimal 6 karakter atas permintaan Manajer. Ini lebih lemah dari sebelumnya untuk sistem yang publik di internet, jadi pengamannya bersandar pada kuncian akun setelah gagal berulang dan batas percobaan login per IP yang sudah ada. Dicatat di sini supaya Sprint 13 menguji kedua pengaman itu, bukan hanya panjang password.
- Umur sesi 9 jam, kira-kira satu hari kerja, supaya tim tidak diminta login ulang di tengah hari.
- Idle timeout 30 menit dihapus atas permintaan Manajer, jadi umur sesi itulah satu-satunya batas waktu. Konsekuensinya: perangkat yang ditinggal dalam keadaan login tetap bisa dipakai siapa pun sampai 9 jam, termasuk untuk membuka dash dan studio yang dijaga `/api/auth/verify`. `last_seen_at` tetap dicatat sebagai keterangan, tetapi tidak lagi memutus sesi. Pencabutan cepat tetap ada lewat logout, ganti password, reset password, dan menonaktifkan akun.

**Selesai kalau:** `./scripts/check.sh` lolos dan terbukti di browser: PIC menambah sub task dengan deskripsi untuk PIC lain, PIC itu membuka Detail dan mengunggah berkas, berkasnya muncul di sub task itu dan tidak tercampur di lampiran task induk, dan password 6 karakter diterima saat ganti password.

**Hasil:** kode selesai 2026-11-17, `./scripts/check.sh` lolos. Migrasi `000010` belum dijalankan di dev maupun produksi. Uji browser menunggu Rifky.

## Sprint 11g: Tautan hasil sub task

**Tujuan:** revisi 9 Manajer. Hasil kerja sering berupa tautan (Figma, Drive, Docs), bukan berkas yang diunggah, jadi sub task perlu tautan deliverable sendiri. Sekalian aturan centang diperlebar: yang membagi pekerjaan juga boleh menutupnya.

- [x] S11g-01 Migrasi `000011`: kolom `deliverable_url` di `task_subtasks`. Aturan tautannya sama dengan deliverable task: hanya `http` dan `https` (`cleanURL`, NFR-SEC-06).
- [x] S11g-02 `POST` dan `PUT` sub task menerima `deliverable_url`. Detail task mengirimkannya kembali per sub task.
- [x] S11g-03 `canToggle` menerima pembuat sub task. Yang boleh mencentang sekarang: PIC sub task, pembuatnya, PIC task mana pun untuk sub task tanpa PIC, dan Manajer kecuali sub task itu dipegang Manajer lain.
- [x] S11g-04 Frontend: tautan diisi lewat form Ubah sub task, ditampilkan sebagai tombol di Detail, dan baris ringkasnya menandai "1 tautan".
- [x] S11g-05 Bar aksi panel detail menempel di dasar panel, baik saat isinya pendek maupun saat digulir sampai habis.

**Keputusan:**
- Tautan diisi di form Ubah, bukan lewat kotak isian terpisah seperti Deliverable task induk. Saat sub task baru dibuat hasilnya belum ada, dan satu form yang mengurus nama, deskripsi, tautan, dan PIC lebih sedikit tombolnya daripada dua jalur simpan.
- Satu tautan per sub task. Kalau hasilnya lebih dari satu, tautannya diarahkan ke folder, atau berkasnya diunggah sebagai lampiran sub task.
- Pembuat sub task boleh mencentang. Dia yang membagi pekerjaan itu, jadi dia yang paling tahu hasilnya sudah sesuai. Aturan Manajer lain (revisi 4) tidak berubah: Manajer tetap tidak bisa mencentang sub task yang dipegang Manajer lain.
- Tautan dibuka di tab baru dengan `rel="noopener noreferrer"`, karena isinya alamat luar yang diketik pengguna.
- Bar aksi panel detail: `sticky bottom-0` saja tidak pernah cukup, dan dua percobaan sebelumnya meleset karena ditebak dari tangkapan layar. Diukur langsung di browser dengan CSS build aplikasi, hasilnya dua sebab terpisah. Pertama, sticky hanya menahan elemen saat digulir dan tidak pernah mendorongnya ke dasar induk, jadi isi yang pendek membuat bar berhenti di tengah; obatnya induk `flex min-h-full flex-col` dengan `mt-auto` di bar. Kedua, sticky `bottom-0` menempel ke tepi dalam padding, bukan tepi area gulir, jadi bar berhenti 16 px di atas dasar karena `py-4` milik panel; obatnya `bottom-[-1rem]`. `gap-6` menggantikan `space-y-6` di pembungkusnya, karena `space-y` menulis `margin-top` ke tiap anak dan menimpa `mt-auto`.

**Selesai kalau:** `./scripts/check.sh` lolos dan terbukti di browser: PIC mengisi tautan Figma di sub task miliknya, tautan muncul sebagai tombol di Detail dan terbuka di tab baru, tautan `javascript:` ditolak, dan pembuat sub task bisa mencentang sub task yang ia tugaskan ke orang lain.

**Hasil:** kode selesai 2026-11-17, `./scripts/check.sh` lolos. Migrasi `000011` belum dijalankan di dev maupun produksi. Uji browser menunggu Rifky.

## Sprint 11h: Acuan dan pratinjau sub task

**Tujuan:** revisi 10 Manajer. Tautan perlu bisa diisi sejak sub task dibuat, bukan hanya lewat Ubah. Tautannya juga dua arah: yang menugaskan menaruh acuan, yang mengerjakan menaruh hasil. Berkas sub task dibuka di web seperti lampiran task, tidak harus diunduh dulu.

- [x] S11h-01 Migrasi `000012`: kolom `reference_url` di `task_subtasks`. `deliverable_url` tetap untuk hasil.
- [x] S11h-02 Isi sub task yang bisa ditulis pengguna dikumpulkan di satu struct `task.SubtaskFields`, dipakai Add dan Update, supaya menambah kolom tidak memanjangkan daftar parameter di tiap lapisan.
- [x] S11h-03 Form tambah sub task punya isian tautan acuan. Form Ubah punya keduanya, acuan dan hasil.
- [x] S11h-04 Detail sub task menampilkan tautan berlabel Acuan dan Hasil; baris ringkas menghitung jumlah tautan.
- [x] S11h-05 Pratinjau gambar dan PDF dipindah ke komponen bersama `AttachmentPreview.vue`, dipakai lampiran task dan lampiran sub task.
- [x] S11h-06 Tautan hasil punya kotak isian sendiri di Detail, tepat di atas daftar berkas, jadi tidak perlu masuk mode Ubah.

**Keputusan:**
- Dua kolom tautan, bukan satu yang dipakai bergantian. Kalau satu kolom, hasil yang dimasukkan PIC akan menimpa acuan dari yang menugaskan, dan acuannya hilang justru saat pekerjaan sedang diperiksa.
- Form tambah hanya menawarkan tautan acuan. Saat sub task baru dibuat hasilnya belum ada, jadi isian hasil di situ hanya menambah kolom kosong yang harus dilewati.
- Dua tautan, dua tempat mengisi. Acuan diisi di form tambah dan form Ubah, karena biasanya ditulis sekali saat pekerjaan dibagi. Tautan hasil diisi di kotak sendiri di Detail, mengikuti pola Deliverable task induk, karena itu yang paling sering diperbarui: satu sub task bisa berganti tautan hasil beberapa kali sampai disetujui, dan membuka form Ubah tiap kali hanya menambah langkah. Form Ubah mengirim ulang tautan hasil apa adanya supaya tidak terhapus.
- Siapa yang boleh mengisi tautan mana tidak dibatasi per kolom: siapa pun yang boleh mengubah sub task (pembuat, PIC, Manajer) boleh mengisi keduanya. Labelnya yang memberi tahu maksudnya. Aturan per kolom akan menambah tiga cabang hak baru tanpa mencegah kesalahan yang berarti.
- Pratinjau sub task memakai komponen yang sama dengan lampiran task, jadi aturan blob dan `sandbox` tanpa `allow-same-origin` hanya ditulis sekali. CSP produksi sudah mengizinkan `blob:` sejak revisi 8 dicatat di Sprint 11d.

**Selesai kalau:** `./scripts/check.sh` lolos dan terbukti di browser: tautan acuan terisi langsung dari form tambah, Detail menampilkan Acuan dan Hasil terpisah, dan berkas gambar atau PDF di sub task terbuka lewat tombol mata tanpa diunduh.

**Hasil:** kode selesai 2026-11-17, `./scripts/check.sh` lolos. Migrasi `000012` belum dijalankan di dev maupun produksi. Uji browser menunggu Rifky.

## Sprint 11i: PIC sub task dari semua akun

**Tujuan:** revisi 11 Manajer. Pekerjaan satu sub task sering perlu dititipkan ke orang di luar daftar PIC task, misalnya minta tolong satu desain ke divisi lain. Yang menyusun sub task tetap PIC task pilihan Manajer, tetapi yang ditugaskan boleh siapa saja.

- [x] S11i-01 `setSubtaskPIC` memvalidasi kandidat dengan `GetEligiblePICs` (akun aktif berperan Staf atau Manajer), bukan lagi `IsTaskAssignee`.
- [x] S11i-02 `lockForSubtask` menerima `subtaskID`. Menambah sub task tetap hanya untuk PIC task dan Manajer (`subtaskID` 0), sedangkan mengubah, menghapus, dan mengisi tautan hasil juga terbuka untuk orang yang ditugaskan di sub task itu.
- [x] S11i-03 Lampiran sub task: `canAttachSubtask` memberi jalan masuk tambahan untuk PIC sub task, supaya yang mengerjakan bisa menaruh dan menghapus berkasnya sendiri.
- [x] S11i-04 Frontend: pilihan PIC sub task diisi dari `/people` (semua akun aktif), dengan PIC task di urutan atas karena merekalah yang paling sering dipilih.

**Keputusan:**
- Yang boleh menyusun sub task tidak berubah: PIC task dan Manajer. Kalau orang luar juga boleh menambah sub task, daftar pekerjaan sebuah task bisa bertambah tanpa sepengetahuan siapa pun yang bertanggung jawab atasnya.
- Orang luar yang ditugaskan tidak ikut menjadi PIC task. Daftar PIC task tetap keputusan Manajer, dan papan, filter PIC, serta divisi task tetap membacanya. Konsekuensinya: task itu tidak muncul di "Task saya" milik orang luar tadi, dan dia menemukannya lewat papan atau tautan langsung. Dicatat di sini supaya Sprint 13 tahu ini memang disengaja, bukan bug. **Diganti revisi 15 (Sprint 11m):** filter PIC dan "Task saya" sekarang ikut menghitung PIC sub task. Yang tetap: orang luar tidak menjadi PIC task.
- Orang luar yang ditugaskan tetap boleh mencentang sub task-nya (aturan `canToggle` tidak berubah, dia PIC sub task), mengisi tautan hasil, dan mengurus berkas sub task itu. Tanpa itu penugasannya tidak ada gunanya.
- Kandidatnya akun aktif berperan Staf atau Manajer, aturan yang sama dengan PIC task (`GetEligiblePICs`). Akun Admin tetap di luar, karena Admin mengelola sistem, bukan mengerjakan task.

**Selesai kalau:** `./scripts/check.sh` lolos dan terbukti di browser: PIC task memilih orang yang bukan PIC task sebagai PIC sub task, orang itu bisa membuka task lewat papan, mencentang sub task-nya, mengisi tautan hasil, dan mengunggah berkas, tetapi tidak bisa menambah sub task baru.

**Hasil:** kode selesai 2026-11-17, `./scripts/check.sh` lolos. Tidak ada migrasi baru. Uji browser menunggu Rifky.

## Sprint 11j: Manajer membagi, bukan menerima

**Dibatalkan revisi 13 (Sprint 11k).** Aturan PIC di bawah tidak berlaku lagi; perbaikan bar aksi S11j-03 tetap.

**Tujuan:** revisi 12 Manajer. Sub task adalah pekerjaan yang dibagikan, dan Manajer ada di sisi yang membagi. Jadi Manajer tidak bisa dipilih sebagai PIC sub task, tetapi tetap bisa membuat sub task dan menugaskannya ke orang lain.

- [x] S11j-01 Query `IsSubtaskPICEligible`: kandidat PIC sub task adalah akun aktif berperan Staf saja. Dipakai `setSubtaskPIC`, jadi aturannya berlaku untuk form tambah maupun form Ubah.
- [x] S11j-02 Frontend: pilihan "Dikerjakan" menyaring peran Manajer, dan PIC task yang berperan Staf tetap di urutan atas.
- [x] S11j-03 Panel detail diberi `overflow-hidden`: tanpa itu panelnya sendiri ikut tergulir dan menyeret bar aksi naik keluar layar.

**Keputusan:**
- Yang berubah hanya sisi penerima. Manajer tetap boleh membuat sub task, menugaskannya, mengubah namanya, menghapusnya, dan mencentangnya (aturan revisi 4 tidak berubah). Yang hilang hanya kemampuan menjadi PIC sub task.
- Akun Admin memang sudah di luar daftar sejak awal, karena Admin mengelola sistem dan tidak mengerjakan task.
- Sub task lama yang terlanjur ditugaskan ke Manajer tidak diubah dan tidak ada migrasi untuk itu. Barisnya tetap sah, dan aturan baru hanya berlaku saat menyimpan. Di form Ubah, PIC lama yang sudah tidak memenuhi syarat tetap ditampilkan sebagai pilihan barisnya sendiri, supaya mengubah nama sub task tidak memaksa mengganti PIC-nya lebih dulu.
- PIC task boleh tetap berisi Manajer. Revisi ini hanya tentang sub task, dan Manajer yang menjadi PIC task memang mengerjakan task itu di tingkat task.
- Bar aksi panel detail, sebab ketiga dan yang sebenarnya. Dua perbaikan sebelumnya (Sprint 11g S11g-05) benar tetapi tidak cukup. Diukur di panel yang sungguhan: `<dialog>` bawaan browser punya `overflow: auto`, dan isi panel melebihi tingginya karena elemen `sr-only` (input berkas tersembunyi dan label pembaca layar) yang `position: absolute` tanpa induk ber-`relative`, sehingga posisinya jatuh jauh di bawah dan menambah `scrollHeight` panel sampai sekitar 400 px. Akibatnya panel itu sendiri bisa tergulir, menyeret area gulir beserta bar aksinya naik keluar layar: terukur `scrollTop` 431 px dengan bar berhenti di 462 dari layar 893. Obatnya `overflow-hidden` pada panel, karena isinya memang sudah diatur flex (header tetap, isian menggulir sendiri). Setelah itu bar menempel di dasar panel pada 1024 px maupun 393 px, di awal gulir, tengah, dan mentok.

**Selesai kalau:** `./scripts/check.sh` lolos dan terbukti di browser: daftar "Dikerjakan" tidak memuat satu pun akun Manajer, Manajer masih bisa menambah sub task dan menugaskannya ke Staf, dan permintaan yang memaksa `pic_id` akun Manajer ditolak dengan pesan "Sub task hanya bisa ditugaskan ke akun Staf yang aktif."

**Hasil:** kode selesai 2026-11-17, `./scripts/check.sh` lolos. Tidak ada migrasi baru. Uji browser menunggu Rifky.

## Sprint 11k: Manajer kembali boleh menjadi PIC sub task

**Tujuan:** revisi 13 Manajer, membatalkan revisi 12. Manajer kembali bisa dipilih sebagai PIC sub task, dan tetap bisa membuat sub task serta menugaskan orang, sama seperti PIC task.

- [x] S11k-01 `setSubtaskPIC` memvalidasi kandidat dengan `GetEligiblePICs` (akun aktif berperan Staf atau Manajer), aturan yang sama dengan PIC task. Query `IsSubtaskPICEligible` dari revisi 12 dihapus. Tidak ada migrasi.
- [x] S11k-02 Frontend: pilihan "Dikerjakan" kembali memuat Manajer. Saringan peran dihapus, karena `/people` sudah hanya berisi Staf dan Manajer aktif.

**Keputusan:**
- Kandidat PIC sub task: akun aktif berperan Staf atau Manajer. Admin tetap tidak bisa, sesuai Sprint 11i: Admin mengelola sistem, bukan mengerjakan task.
- PIC lama yang akunnya sudah nonaktif tetap bertahan saat hanya nama, deskripsi, atau tautan sub task yang diubah. Backend hanya memeriksa kandidat bila `pic_id` berubah, dan form Ubah tetap menampilkan PIC lama itu sebagai pilihan barisnya sendiri karena dia tidak ada di `/people`.
- Aturan centang tidak berubah (`canToggle`): PIC sub task, pembuatnya, PIC task mana pun untuk sub task tanpa PIC (Sprint 11e), dan Manajer kecuali sub task itu dipegang Manajer lain. Pembuat dicek lebih dulu, jadi Manajer yang membuat sub task tetap boleh mencentangnya walaupun PIC-nya Manajer lain. `can_check` dihitung dari fungsi yang sama.
- Hak ubah, hapus, tautan hasil, dan berkas sub task tidak berubah: pembuat, PIC sub task, dan Manajer mana pun (Sprint 11e, 11i).
- Sub task yang selama revisi 12 berlaku ditugaskan ke Staf tidak diubah.

**Selesai kalau:** `./scripts/check.sh` lolos, test sub task Sprint 13 lolos, dan terbukti di browser: daftar "Dikerjakan" memuat Manajer, Manajer A bisa ditugaskan di sub task, Manajer B tidak bisa mencentang sub task itu kecuali dia pembuatnya.

**Hasil:** kode selesai 2026-11-17, `./scripts/check.sh` lolos. Tidak ada migrasi baru. Perlu restart backend.

## Sprint 11l: Popover profil

**Tujuan:** revisi 14 Manajer. Klik foto atau nama orang di tempat yang menampilkan orang, lalu muncul kartu profil kecil: foto atau inisial, nama, peran, divisi, dan jabatan, dengan satu aksi "Lihat task-nya di papan".

- [x] S11l-01 `GET /api/people/:id` untuk semua pengguna yang login (grup `app`). Respons hanya id, nama, divisi, jabatan, peran, status aktif, dan URL foto. 404 untuk id yang tidak ada.
- [x] S11l-02 Detail task mengirim `created_by_id` per sub task dan `actor_id` per riwayat; lampiran mengirim `uploaded_by_id`. Tambahan saja, field lama tetap.
- [x] S11l-03 Komponen bersama `PersonChip` (pemicu) dan `PersonPopover` (satu instance di `App.vue`), dengan state buka di `usePersonPopover`.
- [x] S11l-04 Pemicu dipasang di bulatan PIC kartu papan, panel dan halaman task (pemohon, PIC task, PIC dan pembuat sub task, pengunggah lampiran task, pelaku riwayat), kolom PIC Arsip, dan pelaku Audit Log.
- [x] S11l-05 `scripts/e2e/uji13.mjs` punya bagian popover (`E2E_ONLY=popover`), dan pemeriksaan tombol 44 px membaca area klik `::after` pemicu profil.

**Keputusan:**
- Endpoint baru, bukan memperluas `/people`. `/people` adalah daftar calon PIC (Staf dan Manajer aktif, dengan username untuk filter papan), sedangkan kartu profil juga dibutuhkan untuk Admin dan akun nonaktif yang muncul di riwayat dan audit log. Kolom query dipilih satu per satu, jadi username, password, dan jejak login tidak pernah terbaca.
- Filter PIC papan memakai username (S4-06), padahal kartu profil sengaja tanpa username. Tautan "Lihat task-nya di papan" mencari username lewat `/people` yang memang sudah dipakai filter itu. Akibatnya aksi hanya muncul untuk calon PIC: Admin tidak punya aksi ini karena memang tidak pernah menjadi PIC.
- Popover memakai atribut `popover="manual"` dan di-teleport ke `<dialog>` terdekat dari pemicunya. Panel detail adalah modal dialog yang membuat semua di luarnya inert, jadi popover dari panel harus berada di dalam panel; top layer menaruhnya di atas panel. "manual" dipilih supaya klik luar diatur sendiri.
- Klik di luar popover hanya menutup popover, dan klik itu ditelan: tidak menutup panel, tidak membuka kartu, tidak mengikuti tautan. Escape pertama menutup popover (keydown dibatalkan sehingga panel tidak menerima Escape yang sama), Escape kedua menutup panel.
- Bulatan PIC di kartu papan tidak lagi bertumpuk. Area klik 44 px per bulatan 32 px butuh jarak 12 px supaya tidak ada bulatan yang tertutup area klik tetangganya. Ukuran bulatan tetap; area klik diperluas lewat `::after`, dan bulatan memakai `z-10` di atas lapisan klik judul. Menekan bulatan tidak memulai drag kartu (`draggable` dengan `dragstart` dibatalkan).
- Di bawah 640 px kartu menjadi lembar dari bawah dengan scrim dan `env(safe-area-inset-bottom)`. Di atasnya kartu menempel di pemicu, rata kanan bila mepet kanan, dibuka ke atas bila mepet bawah, lalu dijepit 8 px dari tepi layar. Elemen top layer tidak menambah lebar halaman, jadi tidak ada gulir horizontal.
- Fokus pindah ke tombol Tutup di dalam kartu saat dibuka, dan kembali ke pemicu saat ditutup lewat Tutup, Escape, atau klik luar. Fokus yang pindah ke luar (Tab) menutup popover tanpa menarik fokus kembali.
- Data dimuat saat dibuka dan disimpan per id selama halaman terbuka. Yang gagal tidak disimpan, supaya "Coba lagi" benar-benar meminta ulang.
- Animasi hanya fade dan geser 4 px selama 150 ms, mati di `prefers-reduced-motion`. Tidak ada warna atau radius baru.
- Pembuat sub task tanpa akun tercatat dan pelaku riwayat "Sistem" tetap teks biasa, karena tidak ada orang untuk ditampilkan. Pencentang sub task ("Dicentang ...") tidak termasuk daftar pemicu revisi ini.

**Selesai kalau:** `./scripts/check.sh` lolos, curl `GET /api/people/:id` sebagai Staf, Manajer, dan Admin membalas tanpa field terlarang dan 404 untuk id yang tidak ada, dan `E2E_ONLY=popover node uji13.mjs` lolos di 360 dan 1280 px.

**Hasil:** kode selesai 2026-11-17, `./scripts/check.sh` lolos. Tidak ada migrasi. Perlu restart backend; curl dan e2e popover menunggu backend jalan.

## Sprint 11m: Filter PIC ikut PIC sub task

**Tujuan:** revisi 15 Manajer. Filter PIC = X menampilkan task yang PIC task induknya X, atau yang punya sub task dengan PIC X. Berlaku di papan, Arsip, dan "Task saya". Keputusan revisi 11 soal "Task saya" (Sprint 11i) diganti.

- [x] S11m-01 `ListBoardTasks` dan `ListArchiveTasks`: kondisi filter PIC menjadi EXISTS ke `task_assignees` OR EXISTS ke `task_subtasks` + `subtask_assignees`, untuk id maupun username. `ListTasks` (`GET /api/tasks?scope=mine`) ikut aturan yang sama.
- [x] S11m-02 Query `ListSubtaskPICMatches`: PIC sub task yang cocok dengan filter dan bukan PIC task induk. Dikirim per task sebagai `subtask_pic_matches` (id dan nama) di papan dan Arsip; kosong bila filter PIC tidak aktif.
- [x] S11m-03 Frontend: kartu papan dan baris Arsip menampilkan "Sub task: <nama>" (`SubtaskPicLine`, nama berupa `PersonChip`). Filter PIC di papan dan Arsip punya teks bantuan "Termasuk task yang sub task-nya dipegang orang ini.", terlihat saat filter dipakai dan selalu terbaca pembaca layar lewat `aria-describedby`.
- [x] S11m-04 Test DB `TestBoardPICFilterIncludesSubtaskPIC` dan `TestArchivePICFilterIncludesSubtaskPIC` (Sprint 13 bagian B).

**Keputusan:**
- Dua EXISTS dalam satu OR, bukan JOIN. Task yang cocok lewat dua jalur tetap satu baris, jadi hitungan kolom papan (panjang tiap kolom) dan `total_count` Arsip (`count(*) OVER ()`) otomatis ikut aturan baru tanpa DISTINCT.
- "Task saya" memakai jalur yang sama dengan filter PIC (`BoardFilter.PICIDs` berisi akun sendiri), jadi angka di kartu profil (`/me/task-counts`) ikut menghitung task yang sub task-nya dipegang orang itu.
- Tidak ada migrasi. `subtask_assignees` sudah punya index `subtask_assignees_user_id_idx (user_id)` dan primary key `(subtask_id, user_id)`; `task_subtasks` dicari per task lewat unique `(task_id, position)`. EXISTS sub task masuk daftar `EXPLAIN ANALYZE` S13-06.
- Hak akses tidak berubah. Task yang masuk hasil filter lewat sub task tetap hanya bisa diubah sesuai hak yang dihitung backend: orang luar yang hanya PIC sub task tetap bukan PIC task, jadi tidak bisa memindah status atau mengubah isi task induk.
- Baris "Sub task: <nama>" tidak muncul untuk orang yang juga PIC task induk, karena namanya sudah ada di daftar PIC. Di "Task saya" baris ini juga muncul (berisi nama sendiri), sebagai penjelasan kenapa task itu ada di daftar.
- Diganti revisi 22 (Sprint 11t): `subtask_pic_matches` dan baris "Sub task: <nama>" di kartu papan dihapus, diganti kelompok "Sub task" yang selalu tampil. Logika filter di atas tidak berubah.

**Selesai kalau:** `./scripts/check.sh` lolos, test DB di atas lolos, dan curl papan dengan `pic=<username>` untuk orang yang hanya memegang sub task mengembalikan task itu dengan `subtask_pic_matches` berisi orang tersebut.

**Hasil:** kode selesai 2026-11-18, `./scripts/check.sh` lolos. Tidak ada migrasi. Perlu restart backend. Test DB menunggu `TEST_DATABASE_URL`.

## Sprint 11n: Form Buat task dan tenggat sub task

**Tujuan:** revisi 16 Manajer. Sub task punya tenggat sendiri, dan form Buat task bisa langsung mengisi sub task lengkap (nama, PIC, deskripsi, tautan acuan, tenggat, lampiran acuan) dengan form yang sama dengan Tambah sub task di detail.

- [x] S11n-01 Migrasi `000013`: kolom `task_subtasks.due_date` (date, nullable). Hanya kolom, tanpa grants baru.
- [x] S11n-02 `SubtaskFields.DueDate`, divalidasi `checkSubtaskDue`: tenggat sub task tidak boleh melewati tenggat task induk, galat 400 "Tenggat sub task tidak boleh melewati tenggat task (<tanggal>)." Dipakai tambah, ubah, dan buat task.
- [x] S11n-03 `POST /api/tasks` menerima sub task berisi lengkap (`title`, `description`, `reference_url`, `pic_id`, `due_date`) dan mengembalikan `subtask_ids` urut sesuai form. Task dan semua sub task dibuat dalam satu transaksi; galat menyebut "Sub task nomor N: ...".
- [x] S11n-04 Komponen bersama `SubtaskFieldsForm`, dipakai Tambah sub task dan Ubah sub task di detail serta tiap baris form Buat task. Lampiran acuan diunggah setelah sub task punya id lewat endpoint lampiran sub task yang sudah ada.
- [x] S11n-05 Form Buat task dan Ubah task: Judul, Tenggat, lalu sisanya. Teks bantuan sub task diganti. Tombol hapus baris sub task memakai `text-danger` (sebelumnya `text-urgent`).
- [x] S11n-06 Detail: tenggat sub task tampil seperti tenggat task (ikon jam dan "Terlambat" menurut hari kantor), dan tanda "Melewati tenggat task" bila tenggat task induk lebih awal. Riwayat menulis perubahan tenggat sub task.
- [x] S11n-07 Garis tebal dobel di isian: cincin fokus isian teks kini menimpa garisnya (`outline-offset: -1px`).

**Keputusan:**
- Tenggat sub task opsional. Batas "tidak melewati tenggat task" dicek di aplikasi, bukan CHECK database, karena tenggat task boleh dimajukan setelahnya. Task tanpa tenggat tidak membatasi tenggat sub task.
- Tenggat task yang dimajukan lebih awal dari tenggat sub task tetap diterima. Sub task itu diberi tanda "Melewati tenggat task" di detail. Tenggat sub task hanya dicek ulang bila tenggatnya sendiri diubah, jadi sub task itu tetap bisa diganti nama, deskripsi, atau PIC-nya tanpa dipaksa memperbaiki tenggat dulu. Aturannya sama dengan PIC lama yang dipertahankan (Sprint 11k).
- Penanda terlambat memakai hari kantor (Asia/Jakarta) di frontend, sama dengan `officeTZ` backend. Ini juga mengubah penanda terlambat task induk, yang sebelumnya memakai tanggal browser.
- Audit: `subtask_added` dan `subtask_renamed` mencatat `due_date`; `task_created` mencatat isi tiap sub task (id, nama, PIC, tenggat). Riwayat lama tanpa `due_date` tetap terbaca seperti sebelumnya.
- Form Buat task memakai isian sub task lengkap. Form Ubah task tetap hanya nama dan urutan, sesuai keputusan Sprint 11e dan 11f: daftar dari form menggantikan daftar lama, jadi isian lain di form itu akan menimpa yang sudah diisi PIC. Isi lain sub task diubah dari detail.
- Yang boleh mengisi sub task saat membuat task sama dengan yang boleh menambah sub task: membuat task hanya untuk Manajer (BR-01), dan Manajer selalu boleh menambah sub task. Validasinya sama persis dengan Tambah sub task (`cleanFieldsOf`, `checkSubtaskDue`, `setSubtaskPIC`); tautan hasil dari form Buat task diabaikan karena diisi saat pengerjaan.
- Satu sub task tidak valid membatalkan seluruh pembuatan task. Galatnya diberi awalan "Sub task nomor N:" supaya Manajer tahu baris mana yang salah.
- Lampiran acuan sub task disimpan sebagai lampiran sub task (`kind` deliverable, keputusan Sprint 11f), diunggah setelah task jadi. Berkas yang gagal tidak membatalkan task: panel detail terbuka dan pesan menyebut nama berkas, sub task ke berapa, dan alasannya.
- Isian teks selalu cocok `:focus-visible`, juga sesudah diklik atau difokus otomatis (Tambah sub task memfokuskan nama baris baru). Diukur di form: tanpa fokus garisnya 1 px `line` sama dengan isian lain, dengan fokus garisnya berubah `mark` dan cincin 2 px berjarak 2 px di luarnya, terbaca sebagai garis dobel. Obatnya di aturan global isian, jadi semua isian teks berubah sama: cincin menimpa garis, fokus tetap terlihat. Kotak centang dan pilihan radio tidak ikut.

**Selesai kalau:** `./scripts/check.sh` lolos, test DB di bawah lolos, migrasi `000013` jalan di dev, dan curl buat task dengan dua sub task (satu bertenggat) berhasil, sedangkan sub task bertenggat setelah tenggat task ditolak 400 tanpa task tersimpan.

**Hasil:** kode selesai 2026-11-18, `./scripts/check.sh` lolos. Migrasi `000013` sudah jalan di dev (Rifky). Curl 2026-11-18: buat task dengan dua sub task 201 (`T-0007`, satu bertenggat 2026-12-15 dan satu tanpa, PIC 4 dan 5 tersimpan), sub task bertenggat setelah tenggat task 400 "Sub task nomor 2: Tenggat sub task tidak boleh melewati tenggat task (20 Desember 2026)." tanpa task tersisa. Audit browser penuh `uji13.mjs` (termasuk form Buat task di semua lebar dan kedua tema) lolos di Sprint 11o. Test DB menunggu `TEST_DATABASE_URL`; `TestCheckSubtaskDue` lolos.

## Sprint 11o: PIC sub task memindahkan task, form tambah sub task dilipat

**Tujuan:** revisi 17 Manajer. (A) PIC sub task boleh memulai task dan menandai Ready, karena dia yang mengerjakan sebagian pekerjaannya. (B) Form tambah sub task di panel detail dilipat di balik satu tombol, supaya panel tidak langsung panjang bagi yang hanya melihat task.

- [x] S11o-01 `CheckTransition` dan `PermissionsFor` menerima `isSubtaskPIC`. Tabel `edges` punya penanda `subtaskPIC` untuk Requested ke On Progress dan On Progress ke Ready. Ready gate dan optimistic locking tidak berubah.
- [x] S11o-02 `Move` menghitung PIC sub task dari `subtask_assignees` di dalam transaksi yang mengunci task. Papan dan Arsip memakai `ListTasksWithSubtaskPIC` (satu query untuk semua kartu, hanya untuk Staf); detail memakai `Detail.IsSubtaskPIC`.
- [x] S11o-03 `can.subtask_pic` dikirim ke frontend hanya untuk teks. Detail menampilkan "Kamu PIC sub task di task ini. ..." untuk PIC sub task; "Kamu hanya bisa melihat task ini." tetap untuk yang tanpa hak apa pun.
- [x] S11o-04 Form tambah sub task di detail dilipat di balik tombol "Tambah sub task" (ikon plus, 44 px, `aria-expanded`, `aria-controls`). Batal dan Escape menutupnya tanpa menutup panel, fokus kembali ke tombol, isian bertahan selama panel terbuka. Transisi 150 ms, mati di `prefers-reduced-motion`.
- [x] S11o-05 Bar progres sub task diberi `role="progressbar"` dengan `aria-valuetext` ("1 dari 3 tercentang"). Bar itu memang bar progres (muncul hanya bila ada sub task, lebarnya persentase tercentang), bukan elemen sisa; sebelumnya `aria-hidden`.
- [x] S11o-06 `uji13.mjs`: bagian `E2E_ONLY=subtaskform`, dan pemilih tombol Detail sub task dipersempit ke `aria-controls^='sub-detail-'` (sejak revisi 14 pemicu profil juga ber-`aria-expanded`).
- [x] S11o-07 Perbaikan popover profil revisi 14 yang ketahuan saat e2e popover pertama kali dijalankan: fokus tidak pindah ke popover, fokus tidak kembali sesudah klik luar, dan Enter pada pemicu tertelan sesaat setelah scrim di ponsel diketuk.

**Keputusan:**
- PIC sub task di sini berarti `user_id` di `subtask_assignees` untuk sub task milik task itu, dihitung backend saat permintaan pindah. Hanya dua perpindahan yang bertambah: Requested ke On Progress dan On Progress ke Ready. Ready ke On Progress (revisi), Ready ke Completed, dan membuka kembali tetap milik PIC task dan Manajer. (Ready ke On Progress dibuka untuk PIC sub task oleh revisi 22, lihat Sprint 11t.)
- Hak lain tidak bertambah. PIC sub task tetap tidak bisa mengubah field task induk, label, deliverable task, atau menambah sub task (`PermissionsFor` hanya memakai `isSubtaskPIC` untuk `MoveTo`). Hak atas sub task-nya sendiri tetap dari Sprint 11i.
- Manajer tidak terpengaruh karena sudah boleh semua perpindahan: cabang `case RoleManager` di `CheckTransition` tidak membaca `isPIC` maupun `isSubtaskPIC`. Karena itu papan dan Arsip melewati query PIC sub task untuk Manajer, dan `TestManagerMoveToIgnoresSubtaskPIC` mengunci anggapan ini: Manajer yang hanya PIC sub task (bukan pembuat, bukan PIC task) mendapat `move_to` yang sama dengan Manajer lain. Bila suatu hari hak Manajer dibatasi per task, test itu gagal dan query-nya wajib dihitung untuk Manajer juga. Admin tetap tidak boleh satu pun.
- Form Buat task tidak berubah: barisnya sudah muncul hanya setelah "Tambah sub task" diklik.
- Tombol pemicu tetap terlihat saat form terbuka (`aria-expanded="true"`), supaya fokus selalu punya tempat kembali dan tombol yang sama bisa menutup form.
- Escape di form tambah dihentikan di form (`keydown` dibatalkan dan tidak diteruskan), jadi `<dialog>` panel tidak menerima Escape yang sama. Terbukti di browser: panel tetap terbuka.
- Popover profil, tiga perbaikan dari pengukuran e2e: (1) fokus dipindah satu render sesudah posisi ditulis, karena sebelum itu popover masih `visibility: hidden` dan browser diam-diam menolak `focus()`; (2) fokus dikembalikan ke pemicu setelah urutan pointer selesai, karena `mousedown` bawaan browser memindahkan fokus ke titik yang diklik setelah `pointerdown`; (3) klik yang ditelan setelah klik luar hanya klik pointer (`detail > 0`), karena di ponsel scrim hilang sebelum `pointerup` dan browser tidak mengirim click, jadi pendengarnya masih terpasang sebentar dan bisa menelan klik dari Enter.

**Selesai kalau:** `./scripts/check.sh` lolos, test di bawah lolos, e2e `subtaskform` dan `popover` lolos, dan curl sebagai `staf.uji` di task yang hanya memberinya sub task: pindah ke On Progress 200, ke Ready saat sub task belum tercentang 400, centang, lalu ke Ready 200.

**Hasil:** kode selesai 2026-11-18, `./scripts/check.sh` lolos. Unit test `TestEveryTransition` (144 kombinasi), `TestPermissionsFor`, dan `TestTaskPICAndSubtaskPIC` lolos. `TestSubtaskPICMoves` (DB) menunggu `TEST_DATABASE_URL`. E2E 2026-11-18: `subtaskform` 18 pemeriksaan 0 gagal, `popover` 27 pemeriksaan 0 gagal, audit penuh 320 pemeriksaan 0 gagal. Perlu restart backend; curl bagian A menunggu restart.

## Sprint 11p: Catatan hasil sub task

**Tujuan:** revisi 18 Manajer. PIC sub task bisa menulis penjelasan hasil kerjanya (output di mana, dikirim ke mana, versi mana yang dipakai), selain tautan hasil dan berkas. Detail sub task dibagi jadi Arahan dan Hasil.

- [x] S11p-01 Migrasi `000014`: kolom `task_subtasks.result_note` (text, nullable). Hanya kolom, tanpa grants.
- [x] S11p-02 `PUT /api/tasks/:id/subtasks/:subtaskId` menerima `result_note` opsional (pointer): tidak dikirim berarti isi lama dipertahankan, jadi klien lama dan form Ubah sub task tidak menghapusnya. Maksimal 2000 karakter (per huruf), spasi tepi dibuang, kosong disimpan NULL. Catatan dan tautan hasil tersimpan dalam satu permintaan; `cleanURL` tetap berlaku.
- [x] S11p-03 Audit `subtask_renamed` mencatat `deliverable_url` dan `result_note` lama dan baru. Riwayat menulis "mengubah catatan hasil", "mengubah tautan hasil", atau keduanya.
- [x] S11p-04 Detail sub task: kelompok Arahan (deskripsi, tautan acuan, berkas acuan) dan Hasil (catatan hasil, tautan hasil, berkas hasil), masing-masing dengan teks kosong. Yang boleh mengisi mendapat textarea "Catatan hasil" dengan penghitung saat mendekati batas, kotak tautan hasil, dan tombol "Simpan hasil". Yang hanya melihat membaca catatan sebagai teks biasa (`whitespace-pre-line`, tanpa `v-html`).
- [x] S11p-05 Berkas sub task punya dua jenis: `brief` (acuan) dan `deliverable` (hasil), lewat `?kind=` di endpoint lampiran sub task. Tanpa `kind` tetap deliverable. Lampiran acuan dari form Buat task dan form Tambah sub task (revisi 16) kini diunggah sebagai `brief`.
- [x] S11p-06 Perbaikan revisi 11 yang ketahuan saat menyusun hak catatan hasil: `CheckUpload` memakai `canAttachSubtask`, detail mengirim `can_edit` dan `can_attach` per sub task dengan aturan backend yang sebenarnya.
- [x] S11p-07 `uji13.mjs`: bagian `E2E_ONLY=subtaskresult`.

**Keputusan:**
- Yang boleh mengisi catatan hasil sama dengan yang boleh mengisi tautan hasil, karena lewat jalur yang sama (`UpdateSubtask`: `lockForSubtask` lalu `editableSubtask`): Manajer, PIC sub task ini (juga dari luar task, revisi 11), dan pembuatnya selama dia masih PIC task, saat task Requested atau On Progress. Tidak ada pemeriksaan baru.
- Endpoint tautan hasil diperluas, bukan endpoint baru: tautan hasil memang disimpan lewat `PUT` sub task. `result_note` dibuat opsional supaya klien yang tidak mengirimnya tidak menghapus catatan.
- Catatan hasil tidak memengaruhi Ready gate maupun aturan centang.
- Berkas acuan dan berkas hasil dibedakan lewat `kind` yang sudah ada di `task_attachments`, bukan kolom baru. Di sub task, hak keduanya sama (PIC task, Manajer, PIC sub task); brief task induk tetap hanya Manajer (FR-TASK-23). Berkas acuan dari revisi 16 yang terlanjur tersimpan sebagai deliverable di data dev tampil di kelompok Hasil; produksi belum punya data.
- Temuan revisi 11, diperbaiki karena revisi ini bergantung padanya: (1) `CheckUpload`, pemeriksaan sebelum isi berkas dibaca, memakai `canAttach` sehingga PIC sub task dari luar task selalu ditolak 403 saat mengunggah berkas sub task-nya, padahal `Upload` sendiri mengizinkan. (2) Detail mengirim `can_edit` hanya bila pelaku PIC task atau Manajer, jadi PIC sub task dari luar task tidak pernah melihat tombol Ubah dan kotak hasil walau backend menerima perubahannya. (3) Tombol berkas sub task mengikuti `can.edit_deliverable` task induk. Sekarang `can_edit` dan `can_attach` dihitung per sub task dengan aturan yang sama dengan backend.
- Penghitung karakter muncul mulai 1800 karakter dan diumumkan lewat `aria-live`. `maxlength` di textarea mencegah ketikan melewati batas; backend tetap menolak 2001 karakter (400).

**Selesai kalau:** `./scripts/check.sh` lolos, test di bawah lolos, migrasi `000014` jalan di dev, curl simpan hasil sebagai PIC sub task 204, sebagai orang tanpa hak 403, dan dengan 2001 karakter 400, serta `E2E_ONLY=subtaskresult` lolos.

**Hasil:** kode selesai 2026-11-18, `./scripts/check.sh` lolos, `TestCleanResultNote` lolos. `TestSubtaskResultNote` (DB) menunggu `TEST_DATABASE_URL`. Migrasi `000014` belum dijalankan; curl dan e2e menunggu migrasi dan restart backend.

## Sprint 11q: Task berulang

**Tujuan:** revisi 19 Manajer. Task bisa diulang harian, mingguan, atau bulanan. Setiap kemunculan adalah task biasa dengan ID sendiri, dibuat otomatis oleh backend pada harinya.

- [x] S11q-01 Migrasi `000015`: tipe `recurrence_freq`, tabel `task_series` (aturan, `start_date`, `end_date`, `active`, `template` JSONB, `created_by`), kolom `tasks.series_id` dan `tasks.occurrence_date`, unique `(series_id, occurrence_date)`. `deploy/sql/10-grants.sql` diberi catatan; daftar DELETE tidak berubah.
- [x] S11q-02 `task/recurrence.go`: aturan hari kemunculan (`Rule.OccursOn`, `Rule.Next`, `Validate`) sebagai fungsi murni dengan tanggal yang disuntikkan.
- [x] S11q-03 `task/series.go`: seri dibuat dari form Buat task atau Ubah task (task itu kemunculan pertama), diperbarui dari Ubah task bila dicentang, dihentikan lewat `POST /api/tasks/:id/stop-series`. `CreateDueOccurrences(now)` membuat kemunculan hari ini.
- [x] S11q-04 Ticker di proses backend setiap 5 menit (`RunOccurrenceTicker`), dan perintah dev `server occurrences [--date YYYY-MM-DD]` yang memanggil fungsi yang sama dengan "hari ini" palsu tanpa mengubah jam sistem.
- [x] S11q-05 API: `schedule` di buat dan ubah task (tidak dikirim berarti seri tidak disentuh), `series_freq` dan `occurrence_date` di papan dan Arsip, `series` (aturan, `next_date`, `due_offset_days`) di detail, filter `series` di Arsip.
- [x] S11q-06 Frontend: bagian Jadwal (`ScheduleFields`) di bawah Deskripsi pada form Buat task dan Ubah task, penanda `RecurrenceBadge` (ikon panah melingkar plus teks) di kartu papan, baris Arsip, dan detail, aturan seri dan "Lihat kemunculan lain" di detail.
- [x] S11q-07 Audit: `series_created`, `series_updated`, `series_stopped` oleh pengguna, `occurrence_created` oleh Sistem (user_id NULL). Masuk kategori Task di Audit Log dan riwayat task.
- [x] S11q-08 Test unit aturan, test DB seri, dan bagian e2e `E2E_ONLY=schedule`.

**Keputusan:**
1. Kemunculan baru tetap dibuat walaupun kemunculan sebelumnya belum Completed.
2. Tanggal bulanan yang tidak ada di bulan itu pindah ke hari terakhir bulan itu (31 jadi 30 April, 28 atau 29 Februari).
3. Lampiran brief tidak ikut diulang. Tautan acuan sub task ikut. Tautan hasil, catatan hasil, deliverable task, dan centang tidak ikut, karena itu hasil pengerjaan kemunculan sebelumnya.
4. Ubah task yang termasuk seri: centang "Pakai perubahan ini untuk kemunculan berikutnya" menyala secara bawaan. Kalau dicentang, aturan dan templat seri diperbarui dari isi task itu sesudah disimpan (judul, deskripsi, urgency, PIC, label, pemohon, jarak tenggat, sub task). Kemunculan yang sudah ada tidak berubah. Memilih "Tidak berulang" dengan centang menyala sama dengan menghentikan.
5. Hak mengatur jadwal mengikuti hak membuat task: hanya Manajer (buat, ubah, hentikan).

Keputusan teknis:
- Templat disimpan sebagai JSONB di `task_series.template`, bukan tabel anak. Templat selalu dibaca dan ditulis utuh, tidak pernah dicari per isinya, dan bentuknya sama dengan input buat task. Tanpa tabel anak, mengganti daftar sub task templat cukup satu UPDATE, jadi app tidak butuh hak DELETE baru.
- Seri tidak pernah dihapus, hanya dihentikan (`active = false`). Tidak ada GRANT DELETE untuk `task_series`.
- Tanggal kemunculan pertama adalah tanggal mulai task, atau hari ini (officeTZ) bila kosong. Tanggal mulai setiap kemunculan berikutnya sama dengan tanggal kemunculannya.
- Jarak tenggat = tenggat dikurangi tanggal kemunculan, dihitung dari task saat seri dibuat atau diperbarui. Tanpa tenggat berarti kemunculan juga tanpa tenggat. Tenggat sub task memakai jaraknya sendiri terhadap tanggal kemunculan yang sama.
- Pembuat kemunculan hanya membuat kemunculan hari ini: hari yang terlewat (server mati, seri baru diaktifkan) tidak disusulkan. Seri yang `end_date`-nya sudah lewat dinonaktifkan oleh pembuat kemunculan.
- Idempoten dan aman untuk dua instance: kemunculan yang sudah ada dilewati lebih dulu (supaya nomor task tidak terpakai sia-sia), dan bentrok unique `tasks_series_occurrence_key` dianggap "sudah dibuat".
- Ticker berjalan di proses backend, bukan cron host, supaya tidak ada langkah deploy tambahan. Dijalankan sekali saat start lalu setiap 5 menit; galat dicatat di log dan tidak menghentikan server.
- Isi templat yang sudah tidak sah tidak menggagalkan seri bila masih bisa dibuat: PIC task yang nonaktif dilewati (divisi mengikuti PIC pertama yang tersisa), label nonaktif dilewati, PIC sub task yang tidak lagi memenuhi syarat dikosongkan. Bila tidak ada satu pun PIC task yang aktif, kemunculan hari itu tidak dibuat dan galatnya tercatat di log.
- Divisi kemunculan tidak disimpan di templat, tetapi dihitung dari PIC pertama seperti task biasa (keputusan Sprint 3), supaya tidak bisa berbeda dengan PIC-nya.
- Kalimat pratinjau di form dihitung di frontend (`utils/recurrence.ts`) karena aturannya belum disimpan. Di detail, kemunculan berikutnya (`next_date`) dihitung server dengan fungsi yang sama dengan pembuat kemunculan.
- "Lihat kemunculan lain" membuka Arsip dengan `series`, jadi hanya kemunculan yang sudah Completed yang tampil. Kemunculan yang masih berjalan ada di papan.

**Selesai kalau:** `./scripts/check.sh` lolos, test di bawah lolos, migrasi `000015` dan `10-grants.sql` jalan di dev, curl membuat task berulang harian, dan `server occurrences --date <besok>` membuat tepat satu kemunculan (dijalankan dua kali tetap satu).

**Hasil:** kode selesai 2026-11-18, `./scripts/check.sh` lolos. Unit test aturan (`TestRuleDaily`, `TestRuleWeekly`, `TestRuleMonthly`, `TestRuleStartEnd`, `TestRuleValidate`, `TestOfficeDate`) lolos. Test DB (`TestSeriesOccurrences`, `TestSeriesEndDate`, `TestSeriesUpdateTemplate`) menunggu `TEST_DATABASE_URL`. E2E `schedule` 20 pemeriksaan 0 gagal. Migrasi `000015` belum dijalankan; curl dan perintah dev menunggu migrasi dan restart backend.

## Sprint 11r: Filter Hari ini dan istilah Deadline

**Tujuan:** revisi 20 Manajer. Papan perlu satu klik untuk melihat yang jatuh tempo hari ini, dan seluruh teks memakai satu kata yang dipakai tim sehari-hari.

- [x] S11r-01 Filter deadline `today` di papan: `due_date` sama dengan hari ini menurut `officeTZ`. Dihitung di backend dengan pola yang sama dengan `overdue` dan `week`, memakai parameter `@today` yang sudah ada. Urutan pilihan: Semua deadline, Terlambat, Hari ini, 7 hari ke depan, Tanpa deadline.
- [x] S11r-02 Kata "Tenggat" diganti "Deadline" di label form task dan sub task, filter, kartu, panel detail, riwayat task, kalimat pratinjau jadwal, penanda "Melewati deadline task", pesan galat backend, dan `docs/panduan-tim.md`.

**Keputusan:**
- "7 hari ke depan" memang menghitung hari ini sebagai hari pertama (`due_date BETWEEN today AND today + 6`), jadi task yang jatuh tempo hari ini muncul di "Hari ini" maupun "7 hari ke depan". Perilaku itu tidak diubah, hanya dicatat supaya tidak dikira bug saat Sprint 13 menguji kedua filter.
- Filter "Hari ini" tidak menyaring status, berbeda dengan "Terlambat" yang mengecualikan Completed. Task yang sudah selesai hari ini tetap muncul, karena yang ditanyakan Manajer adalah apa yang jatuh tempo hari ini, bukan apa yang belum selesai.
- Arsip tidak punya filter deadline, jadi tidak ditambahkan. Arsip disaring dengan tanggal selesai, dan menambahkan filter deadline di sana akan menambah kolom yang tidak ditanyakan siapa pun.
- Penggantian kata berhenti di teks pengguna. Nama kolom (`due_date`), field JSON (`due_date`, `due`), nilai parameter URL (`overdue`, `today`, `week`, `none`), nama variabel, dan nama fungsi tetap, supaya tidak ada migrasi dan tautan papan yang sudah dibagikan tetap berlaku. Komentar kode yang masih memakai kata "tenggat" dibiarkan, karena menyentuhnya hanya memperbesar diff tanpa mengubah apa pun yang dilihat pengguna.

**Selesai kalau:** `./scripts/check.sh` lolos, papan dengan `?due=today` hanya memuat task yang deadline-nya hari ini, dan pencarian kata "tenggat" di teks pengguna tidak menemukan sisa.

**Hasil:** kode selesai 2026-11-18, `./scripts/check.sh` lolos. Tidak ada migrasi. Uji curl dan uji browser menunggu Rifky.

## Sprint 11s: Aktivitas Harian

**Tujuan:** revisi 21 Manajer. Mengganti spreadsheet "Daily Activity Monitoring" dengan tabel di aplikasi: Staf mengisi aktivitas dan output per jam kerja, Manajer melihat semua orang dalam satu tanggal dan mengekspor rentang tanggal ke Excel.

- [x] S11s-01 Migrasi 000016: tabel `daily_activities` (user_id, activity_date, slot, activity, output, updated_by, updated_at), unique (user_id, activity_date, slot), index (activity_date), CHECK panjang 1000 karakter. Sel yang dikosongkan disimpan NULL, tidak pernah dihapus, jadi daftar DELETE di `deploy/sql/10-grants.sql` tidak bertambah.
- [x] S11s-02 Daftar slot tetap di `internal/activity` (09.00-10.00 sampai 17.00-18.00, tanpa 12.00-13.00). Frontend mengambilnya dari API.
- [x] S11s-03 Hak akses di backend: Staf hanya isian miliknya sendiri untuk hari ini (`officeTZ`), Manajer isian siapa pun di tanggal berapa pun, Admin hanya membaca.
- [x] S11s-04 API: `GET /api/activities/me`, `GET /api/activities?date=`, `PUT /api/activities`, dan `GET /api/activities/export?from=&to=` (Manajer, maksimal 31 hari).
- [x] S11s-05 Kartu "Aktivitas hari ini" di papan task untuk Staf: bisa dilipat (status lipat di localStorage), ringkasan "n dari 8 jam terisi", simpan otomatis saat sel kehilangan fokus dengan status Menyimpan, Tersimpan, atau Gagal disimpan beserta tombol Coba lagi.
- [x] S11s-08 Tata letak kartu diputar (revisi 21, permintaan kedua): `<table>` tiga baris, jam sebagai `<th scope="col">`, label Aktivitas dan Output sebagai `<th scope="row">` yang lengket di kiri, satu textarea per sel. Di bawah 768 px tetap kartu per jam.
- [x] S11s-09 Halaman `/aktivitas` lebar penuh (mengubah aturan lebar konten Sprint 11c), tabel bergulir di dalam wadahnya sendiri dengan kepala nama dan kolom jam yang lengket.
- [x] S11s-10 Mode tampilan Minggu (bawaan) dan Hari, tersimpan di query URL (`?mode=minggu&tanggal=2026-11-16`), dengan segmented, navigasi mundur dan maju, pemilih tanggal, tombol hari Sen sampai Jum di mode Hari, filter divisi, dan filter orang (pilih beberapa, dengan pencarian).
- [x] S11s-11 Satu endpoint baca rentang: `GET /api/activities` menerima `week`, `from` dan `to`, atau `date`. Backend mengembalikan daftar tanggal yang tampil beserta sel per tanggal, jadi frontend tidak memanggil API per hari.
- [x] S11s-12 Sel: kosong dibiarkan kosong (teks "kosong" hanya untuk pembaca layar), teks dipotong tiga baris, output URL jadi chip ikon dan nama host, dan klik atau Enter membuka dialog berisi isi penuh. Bagi Manajer dialog itu sekaligus form ubah dengan Simpan dan Batal. Kepala kolom memuat PersonChip, divisi, dan ringkasan "n dari N jam terisi" dengan bar token progres.
- [x] S11s-13 Revisi 21 lanjutan: label hari tetap terlihat saat tabel digeser ke samping, tangga lapisan sel lengket yang opak, jenis tautan Google di Output (chip ikon dan label, thumbnail file Drive di dialog, CSP `img-src`), dan dialog sel yang selalu dibuka dalam mode lihat dengan tombol Ubah untuk Manajer.
- [x] S11s-06 Halaman `/aktivitas`: navigasi tanggal di query URL, filter divisi, kolom Jam lengket, nama orang sebagai PersonChip, tanda "Belum diisi", output berupa URL jadi tautan, dan tombol Ekspor Excel untuk Manajer.
- [x] S11s-07 Test: unit (daftar slot, pembersihan teks, batas karakter, hak akses, pembentukan kolom ekspor), DB (aturan simpan per peran, upsert satu baris, Manajer dan Admin bukan kolom, akhir pekan di ekspor), dan bagian `E2E_ONLY=activity` di `scripts/e2e/uji13.mjs`.

**Keputusan:**
- Yang punya tabel aktivitas hanya akun aktif berperan Staf. Manajer dan Admin tidak punya kolom sendiri, dan isian tidak bisa dititipkan ke mereka: Manajer membagi dan menilai pekerjaan, tidak menerimanya, sama dengan aturan sub task di revisi 12.
- Staf hanya bisa mengisi dan mengubah isian miliknya untuk hari ini menurut `officeTZ`. Manajer bisa mengubah isian siapa pun di tanggal berapa pun. Keduanya diperiksa backend (`activity.CanEdit`), frontend hanya mengikuti `can_edit` dari server.
- Perubahan Manajer pada isian orang lain dicatat di Audit Log kategori baru "Aktivitas" (`activity_updated`), dan sel itu menampilkan "diubah <nama Manajer>". Isian Staf sendiri tidak dicatat, supaya audit log tidak penuh oleh pekerjaan harian biasa.
- Sabtu dan Minggu boleh diisi. Halaman Aktivitas menampilkan tanggal apa pun yang dibuka, sedangkan ekspor melewati Sabtu dan Minggu yang tidak ada isinya, supaya sheet-nya sependek spreadsheet lama.
- Baris "Evaluation Week" dari spreadsheet tidak dibuat. Belum ada aturan siapa yang mengisinya dan apa artinya, jadi menambahkannya sekarang hanya menyisakan kolom kosong.
- Isian kosong disimpan sebagai NULL dan barisnya dibiarkan, bukan dihapus. Dengan begitu app tidak butuh hak DELETE di tabel baru ini, dan "diubah <nama>" tetap terbaca setelah Manajer mengosongkan sebuah sel.
- Ekspor .xlsx ditulis sendiri sebagai zip berisi XML dengan pustaka standar, mengikuti cara `internal/excelimport` membaca .xlsx. Tidak ada dependensi baru. Nama hari dan bulan di sheet memakai bahasa Inggris seperti spreadsheet lama ("Monday - 2", "Daily Activity Monitoring - November 2026"), dan kolom Week memakai nomor minggu ISO.
- Batas 1000 karakter per kolom berlaku di backend dan di CHECK migrasi. Spasi di tepi dibuang dan akhir baris diseragamkan ke LF sebelum disimpan.
- Tata letak kartu di papan diputar atas permintaan Manajer: delapan baris jam yang bertumpuk memakan hampir satu layar, sedangkan jam sebagai kepala kolom membuat kartunya hanya tiga baris tinggi. Mulai 768 px dipakai `<table>` sungguhan supaya pembaca layar menyebut "Aktivitas, 10.00-11.00" dari kepala baris dan kolomnya; di bawah 768 px delapan kolom tidak mungkin muat, jadi tampilan kartu per jam dipertahankan.
- Urutan Tab di tabel itu diatur satu penangan `keydown` pada wadah tabel: per jam, Aktivitas lalu Output. Tabel HTML selalu row-major di DOM, jadi tanpa penangan ini Tab akan melewati seluruh baris Aktivitas dulu. `tabindex` positif sengaja tidak dipakai, dan Tab dari sel terakhir tetap dibiarkan keluar dari tabel seperti biasa.
- Status simpan satu jam ditampilkan satu kali, di bawah textarea Output. Satu kali simpan mengirim Aktivitas dan Output sekaligus, jadi statusnya milik jamnya, bukan milik satu textarea; menampilkannya dua kali di kolom yang sama hanya terbaca seperti galat ganda. Textarea Aktivitas menunjuk ke baris status itu lewat `aria-describedby`, jadi hasil simpan tetap terbaca pembaca layar saat fokus ada di atas.
- Tinggi textarea memakai `field-sizing: content` dengan batas `max-h-40` dan gulir di dalamnya, serta hitungan `scrollHeight` sebagai cadangan untuk browser yang belum mendukung properti itu. Pegangan resize dimatikan, karena tinggi yang diatur tangan merusak kerataan baris tabel.
- Kolom jam minimal 10rem (sekitar 160 px) dan rata lewat `table-fixed`. Di lebar papan yang lebih sempit dari 87rem, termasuk 1280 px, tabelnya bergulir ke samping di dalam wadahnya sendiri sementara kolom label tetap terlihat; halamannya tidak pernah bergulir horizontal.
- Halaman `/aktivitas` memakai lebar penuh, jadi daftar pengecualian di Keputusan Sprint 11c bertambah: bukan hanya papan. Alasannya sama dengan papan, yaitu jumlah kolom yang ditentukan data, satu kolom per Staf (sekitar 8 orang) kali satu minggu. Halaman tabel lain (Arsip, Pengguna, Audit Log) tetap ditengahkan karena kolomnya tetap dan sedikit.
- Mode Minggu jadi bawaan, isinya Senin sampai Jumat seperti spreadsheet lama; Sabtu dan Minggu tetap mengikuti keputusan sebelumnya, yaitu hanya muncul kalau ada isinya. Kecualinya satu: tanggal yang diminta tegas satu hari (mode Hari) selalu ditampilkan, termasuk Sabtu dan Minggu yang kosong. Tanpa itu Manajer tidak punya sel untuk diisi di akhir pekan, karena Staf pun hanya bisa mengisi hari ini dari papan. Aturan penyaringan tetap berlaku untuk rentang, yaitu tampilan mingguan dan ekspor. Mode dan tanggal disimpan di query URL (`?mode=minggu&tanggal=2026-11-16`) supaya tampilan yang sama bisa dibagikan, sama seperti filter papan.
- Satu panggilan API untuk seluruh rentang. `GET /api/activities` menerima `week` (Senin sampai Jumat minggu tanggal itu), `from` dan `to`, atau `date`, dan mengembalikan `dates` berisi tanggal yang benar-benar tampil. Aturan akhir pekan dan aturan kolom dihitung di backend (`collect`), dipakai bersama oleh halaman dan ekspor, supaya keduanya tidak bisa berbeda. Memanggil API per hari dari frontend sengaja dihindari: lima permintaan untuk satu layar membuat gulir terasa patah dan filter kolomnya bisa berbeda antar hari.
- Sel diubah lewat dialog, bukan isian di dalam tabel. Satu minggu berisi sekitar 320 sel (8 orang kali 5 hari kali 8 jam), dan menaruh dua textarea di setiap sel berarti ratusan isian di satu halaman. Dialognya memakai endpoint PUT dan audit yang sama dengan isian di papan, jadi aturan hak akses tidak bercabang. Isian di dalam tabel tetap dipakai di kartu papan, yang hanya delapan jam milik sendiri.
- Area klik sel adalah tombol selebar sel (`absolute inset-0`), bukan `<td>` yang diberi `role`, supaya semantik tabel utuh dan Enter bekerja tanpa penangan sendiri. Chip tautan diletakkan di lapisan `z-10` di atas tombol itu, jadi tautannya tetap bisa dibuka sendiri. Konsekuensinya sel yang punya tautan punya dua perhentian Tab, dan itu diterima: alternatifnya menyarangkan tautan di dalam tombol, yang tidak sah dan membuat pembaca layar bingung.
- Output yang berupa URL hanya ditampilkan sebagai ikon tautan plus nama host. URL penuhnya ada di `title` dan `aria-label`, serta di dialog. Alasannya URL Drive sepanjang 100 karakter merusak lebar kolom dan membuat tabel tidak bisa dibaca.
- Sorotan hari ini memakai token yang sudah ada: latar `canvas` (beda tipis dari `surface`) plus garis kiri `mark` di baris kepala harinya. Tidak ada warna baru, dan rasio kontras teks `ink-muted` di atas `canvas` tetap 5.73 (terang) dan 6.2 (gelap).
- Pemeriksaan E2E untuk halaman ini menunggu jumlah kepala hari di tabel, bukan `networkidle` plus jeda. Tabelnya diganti kerangka memuat setiap kali mode atau tanggal berubah, jadi pemeriksaan yang hanya menunggu jeda kadang membaca kerangka itu dan lulus atau gagal tergantung kecepatan mesin (terbukti: pemeriksaan mode Hari gagal hanya di 1920 px pada uji pertama Rifky).
- Test unit frontend memakai pemeriksa bawaan Node (`node:test`) dan kemampuan Node 24 menjalankan TypeScript langsung, dijalankan dengan `npm run test:unit`. Tanpa dependensi baru, dan sengaja belum dimasukkan ke `scripts/check.sh`: rangka pengujian frontend adalah pekerjaan Sprint 13, yang akan menentukan sendiri alat dan cakupannya.
- Hitungan minggu ada di dua tempat dengan sengaja: `activity.WeekRange` di backend (sumber kebenaran saat membaca data) dan `utils/activityWeek.ts` di frontend (untuk tombol mundur dan maju serta label rentang, yang tidak boleh menunggu jaringan). Keduanya memakai aturan yang sama (Senin sampai Jumat, Sabtu dan Minggu masih masuk minggu Senin yang sudah lewat) dan masing-masing punya test unitnya.
- Revisi 21 lanjutan, label hari: sel kepala hari adalah satu `<th colspan>` selebar tabel, jadi sel itu sendiri ikut bergulir ke samping dan `sticky` pada sel itu tidak berpengaruh. Yang dibuat lengket adalah isinya (`<span class="sticky left-0">`) dengan latar opak yang sama dengan barisnya; garis kiri penanda hari ini ikut pindah ke span itu supaya tetap terlihat.
- Revisi 21 lanjutan, lapisan sel lengket: sudut kiri atas `z-40`, kepala nama `z-30`, kolom jam `z-20`, label hari `z-10`, semuanya berlatar opak (`surface`, atau `canvas` di grup hari ini). Diukur di Edge sebelum perubahan: dengan data dev, tengah sel pojok dan sel jam sudah mengembalikan sel itu sendiri di `elementFromPoint` (terang dan gelap, 360 dan 768 px), jadi tembus pandang yang dilaporkan tidak terulang di sini. Tangganya tetap dibuat tegas dan diperiksa E2E di setiap lebar, supaya kalau terjadi lagi langsung tertangkap.
- Revisi 21 lanjutan, jenis tautan Output dikenali dari URL saja, tanpa memanggil jaringan: `drive.google.com/file/d/<id>` "File Drive", `drive.google.com/drive/folders/<id>` (termasuk `/drive/u/<n>/folders`) "Folder Drive", `docs.google.com/document` "Google Docs", `/spreadsheets` "Spreadsheet", `/presentation` "Slide", selain itu nama host. Host dicocokkan persis, jadi `docs.google.com.contoh.test` tetap dianggap host lain. Ikonnya garis sederhana buatan sendiri, bukan logo Google. Kode ada di `utils/outputLink.ts` tanpa impor, supaya test unit Node bisa memuatnya langsung.
- Revisi 21 lanjutan, thumbnail file Drive di dialog diambil browser pengguna dari `https://drive.google.com/thumbnail?id=<id>&sz=w480` dengan `referrerpolicy="no-referrer"`, jadi alamat aplikasi tidak terkirim ke Google dan server kita tidak pernah menyentuh Drive. Gambar hanya muncul bila pengguna login Google dan punya akses; selain itu permintaannya dialihkan ke halaman login, `<img>` gagal, dan area gambarnya disembunyikan (begitu juga gambar yang lebih kecil dari 64x32 px, yaitu ikon umum Drive). Ruangnya tetap lewat `aspect-video`. CSP `img-src` ditambah `https://drive.google.com` dan `https://*.googleusercontent.com` (tujuan pengalihan thumbnail); direktif lain tidak berubah.
- Revisi 21 lanjutan, dialog sel selalu dibuka dalam mode lihat untuk semua peran, menggantikan keputusan sebelumnya bahwa bagi Manajer dialog itu langsung form. Manajer (`can_edit` dari server) mendapat tombol Ubah; Batal dan simpan yang berhasil kembali ke mode lihat dengan isi terbaru, fokus pindah ke textarea Aktivitas saat masuk mode ubah dan kembali ke tombol Ubah saat keluar. Escape bertahap: di mode ubah dengan perubahan yang belum disimpan, Escape membuang perubahan dan kembali ke mode lihat; Escape berikutnya menutup dialog. Tombol Tutup selalu langsung menutup. Untuk itu `BaseDialog` mengirim alasan `'escape'` pada event `close`; dialog lain tidak memakainya, jadi perilakunya tidak berubah.
- Revisi 21 lanjutan, teks "Ubah isianmu lewat tabel di papan task." di dialog hanya tampil bagi Staf yang membuka isiannya sendiri. Di atas tabel, teks yang sama kini hanya untuk Staf; sebelumnya ikut tampil untuk Admin, yang tidak punya isian.
- Panel filter Orang tertutup kepala tabel (bug, 2026-11-18): wadah tabel tidak membentuk stacking context, jadi sel lengket `z-30` dan `z-40` di dalamnya bersaing langsung dengan panel `z-30` dan menang karena datang belakangan di DOM. Dipilih `isolation: isolate` (`isolate`) pada wadah tabel, bukan teleport ke body: tangga z-index tabel jadi hanya berlaku di dalam tabel, dan panel otomatis di atasnya tanpa bergantung pada angka z-index sel. Teleport butuh hitungan posisi saat gulir dan ubah ukuran, pembalikan ke atas, dan penjepitan tepi, sedangkan masalahnya hanya urutan lapisan; panel tetap ikut aliran halaman di bawah 640 px tanpa kode tambahan. Escape kini dipasang di pembungkus tombol dan panel (sebelumnya hanya di panel, jadi tidak bekerja saat fokus masih di tombol) dan mengembalikan fokus ke tombol; klik di luar menutup panel tanpa memindah fokus. Label tombol "Orang" atau "Orang (2)" sebagai satu teks, supaya nama aksesibelnya tidak terbaca "Orang(2)". E2E `activity` memeriksa `elementFromPoint` di item pertama panel setelah tabel digulir, di 360 dan 1280 px (sebelum perbaikan gagal di 1280 px: tertutup kepala nama).
- Panel filter orang ikut aliran halaman di bawah 640 px dan baru melayang (`absolute`) mulai 640 px. Panel melayang selebar 18rem di layar 360 px akan menambah lebar dokumen dan membuat halaman bergulir horizontal.

**Selesai kalau:** `./scripts/check.sh` lolos, migrasi 000016 jalan di dev, Staf bisa mengisi kartu di papan task dan isiannya muncul di `/aktivitas`, Manajer bisa mengubah isian orang lain (tercatat di Audit Log) dan mengunduh .xlsx, serta tidak ada gulir horizontal halaman di 360 dan 1280 px.

**Hasil:** kode selesai 2026-11-18, `./scripts/check.sh` lolos. Migrasi 000016 dan `10-grants.sql` menunggu Rifky. Menu "Aktivitas" di navbar belum dipasang: `frontend/src/components/AppShell.vue` sedang dipakai perubahan Sprint 13 yang belum di-commit, jadi baris navigasinya diberikan ke Rifky untuk ditempel.

## Sprint 11t: PIC sub task di kartu dan revisi oleh PIC sub task

**Tujuan:** revisi 22 Manajer. (A) Siapa yang memegang sub task terlihat di setiap kartu, bukan hanya saat filter PIC dipakai. (B) PIC sub task bisa mengembalikan task Ready ke On Progress untuk merevisi bagiannya sendiri tanpa menunggu PIC task.

- [x] S11t-01 Query `ListSubtaskPICs`: semua PIC sub task per task, unik, urut menurut posisi sub task pertama yang dipegang, satu query untuk semua kartu (`attachPeopleAndLabels`, jadi papan, Arsip, dan "Task saya" sekaligus). Dikirim sebagai `subtask_pics` (id, nama, username, avatar). `ListSubtaskPICMatches`, `attachSubtaskPICs`, dan `subtask_pic_matches` dihapus.
- [x] S11t-02 Kartu papan: area bawah dibagi dua kelompok berlabel kecil, "PIC" dan "Sub task" (`AvatarGroup`), maksimal 4 bulatan per kelompok lalu "+n" dengan nama lengkap di `title` dan `aria-label`. Arsip tidak memakai komponen kartu yang sama; barisnya tetap "Sub task: <nama>", sekarang berisi semua PIC sub task.
- [x] S11t-03 Aturan: tabel `edges` memberi Ready ke On Progress penanda `subtaskPIC`. `IsSubtaskRevision` menandai jalur Staf yang hanya PIC sub task; di jalur itu catatan tidak wajib.
- [x] S11t-04 `Move` menerima `subtask_ids`. Di jalur revisi PIC sub task: minimal satu, setiap id harus sub task miliknya di task itu yang tercentang (400 bila tidak), centangnya dibuka lagi (`ReopenSubtasks`), dan status, pembukaan centang, serta audit tersimpan dalam satu transaksi yang sudah mengunci task dengan `version` (409 bila basi). Di jalur lain `subtask_ids` yang terisi ditolak 400.
- [x] S11t-05 `MoveOptions` menghitung `move_to` dan `revision_subtasks` dari sub task milik pelaku (`ListMySubtasks`, menggantikan `ListTasksWithSubtaskPIC`), dipakai papan, Arsip, dan detail.
- [x] S11t-06 Frontend: dialog Pindahkan dan dialog revisi di panel detail menampilkan "Sub task yang direvisi" (semua terpilih secara bawaan) dan "Alasan revisi (opsional)" dengan tombol "Kembalikan ke On Progress" bila `revision_subtasks` dari server berisi (`SubtaskRevisionFields`).
- [x] S11t-07 Riwayat task dan Audit Log: "<nama> mengembalikan ke On Progress untuk revisi: <sub task>. Alasan: <alasan>."
- [x] S11t-08 Test: unit (`TestEveryTransition`, `TestTaskPICAndSubtaskPIC`, `TestPermissionsFor`, `TestMoveOptionsSubtaskRevision`), DB (`TestSubtaskPICRevision` baru, `TestSubtaskPICMoves` dan test filter revisi 15 diperbarui), dan bagian `E2E_ONLY=revision` di `scripts/e2e/uji13.mjs`.

**Keputusan:**
- Revisi 22 menambah aturan revisi 17: PIC sub task (Staf yang bukan PIC task) boleh Ready ke On Progress. Ready ke Completed dan membuka kembali Completed tetap milik Manajer.
- Jalurnya ditentukan backend, bukan frontend: Staf, bukan PIC task, memegang minimal satu sub task, dari Ready ke On Progress (`IsSubtaskRevision`). Staf yang sekaligus PIC task dan Manajer yang memegang sub task tetap memakai jalur revisi biasa dengan catatan wajib, karena perilaku mereka tidak boleh berubah. Untuk mereka `revision_subtasks` selalu kosong dan `subtask_ids` yang dikirim ditolak 400, supaya tidak ada sub task yang terbuka diam-diam lewat jalur yang tidak memeriksanya.
- Jalur ini butuh minimal satu sub task milik pelaku yang sedang tercentang. Bila tidak ada (mis. Manajer melepas centangnya, atau sub task baru ditambahkan di Ready), On Progress dibuang dari `move_to`, supaya tombol yang tampil selalu cocok dengan yang diterima `Move`.
- Hanya sub task yang dipilih yang dibuka. Sub task lain, termasuk milik orang lain dan sub task miliknya yang tidak dipilih, tetap tercentang. Akibatnya task kembali bisa ke Ready begitu sub task yang direvisi dicentang lagi, tanpa mengulang pekerjaan orang lain.
- Pilihan yang ditolak menyebut id sub task, bukan judulnya: sub task milik orang lain atau dari task lain tidak boleh dibocorkan judulnya lewat pesan galat.
- Alasan revisi opsional, maksimal 500 karakter (batas yang sama dengan catatan revisi, `MaxNoteLength`). Disimpan di audit `task_status_changed` dengan kunci `reason` dan `reopened_subtasks` (id dan judul saat itu), bukan `note`, karena riwayat menulisnya di dalam satu kalimat; dengan `note` alasannya akan tampil dua kali (kalimat dan baris "Catatan"). Tidak ada action audit baru, jadi filter kategori Task di Audit Log tetap menangkapnya.
- Centang yang dibuka tidak dicatat sebagai `subtask_unchecked` terpisah. Satu kejadian untuk satu keputusan, dan daftar sub task yang dibuka sudah ada di kejadian itu.
- Kelompok "Sub task" di kartu berisi semua PIC sub task, termasuk orang yang juga PIC task induk. Ini berbeda dari revisi 15 yang menyembunyikannya: di sana barisnya menjelaskan hasil filter, di sini kelompoknya menjawab siapa mengerjakan sub task, dan PIC task yang memegang sub task memang mengerjakannya.
- Batas 4 bulatan per kelompok menjaga satu kelompok tetap satu baris di kartu 360 px (4 bulatan plus "+n", sekitar 208 px). Dua kelompok penuh tidak muat sebaris di lebar itu, jadi keduanya memakai `flex-wrap` dan kelompok "Sub task" turun ke baris berikutnya, tanpa gulir horizontal. Kelompok "PIC" memakai aturan "+n" yang sama, walaupun task jarang punya lebih dari 4 PIC.
- Satu query `ListSubtaskPICs` untuk semua kartu, dipanggil dari `attachPeopleAndLabels` bersama PIC dan label. Detail task membentuk daftar yang sama dari sub task yang sudah dimuat (`detailSubtaskPICs`), tanpa query tambahan.
- Tidak ada migrasi. Query baru memakai index yang sudah ada (`task_subtasks_position_key (task_id, position)` dan primary key `subtask_assignees`). Tidak ada tabel baru, jadi grants tidak perlu dijalankan ulang.

**Selesai kalau:** `./scripts/check.sh` lolos, test di atas lolos, curl skenario dua sub task memberi hasil yang diharapkan, dan E2E `revision` lolos di 360 dan 1280 px.

**Hasil:** kode selesai 2026-11-18, `./scripts/check.sh` lolos, unit test lolos. Setelah restart backend: `scripts/uji-revisi22.sh` 26 lolos 0 gagal (setiap penolakan dicek kode dan isi pesannya), E2E `revision` 28 pemeriksaan 0 gagal di 360 dan 1280 px. Test DB menunggu `TEST_DATABASE_URL`.

**Catatan pengujian:** versi pertama `uji-revisi22.sh` membaca detail task di tingkat atas, padahal `GET /api/tasks/:id` membungkusnya di kunci `task`. Akibatnya version dan id sub task kosong, dan kasus 400 ikut "lolos" dengan pesan "Isian tidak terbaca.". Skrip sekarang membaca `d['task']`, mencocokkan pesan galat, dan berhenti bila nilai yang dibutuhkan kosong. E2E `revision` punya kesalahan bentuk yang sama di persiapan datanya dan sudah dibetulkan, ditambah pemilihan kolom Ready di layar sempit.

**Perbaikan kecil (pesan 403 per aksi, 2026-11-18):** ditemukan saat menguji revisi 22. Semua penolakan hak di package task memakai `ErrForbidden` polos, dan `writeTaskError` memberinya satu pesan bersama tentang centang sub task, jadi Staf yang ditolak memindahkan task membaca "Sub task ini hanya bisa dicentang PIC-nya...". Keputusan:

- Penolakan sekarang `ForbiddenError` yang membawa pesan aksinya (`task/forbidden.go`) dan tetap cocok dengan `errors.Is(err, ErrForbidden)`, jadi pemeriksaan lama tidak berubah. Handler membaca pesannya lewat `errors.As`; pesan bersama di `writeTaskError` dan `writeSubtaskError` dihapus.
- Pesan per aksi: membuat, mengubah, menghapus task; label; tautan deliverable; menghentikan pengulangan; pindah status (`Kamu tidak punya hak memindahkan task ini ke <status>.`); menambah, mengubah, menghapus sub task; sub task orang lain (`ErrSubtaskLocked`); centang (`ErrNotSubtaskPIC`); mengunggah dan menghapus lampiran brief, berkas deliverable, dan berkas sub task. Hanya aksi atas sub task yang menyebut sub task.
- Pesan centang mengikuti `canToggle` ("PIC-nya, pembuatnya, atau Manajer (kecuali sub task yang dipegang Manajer lain)"). Pesan lama di handler ("Hanya PIC sub task ini yang bisa mencentangnya.") keliru karena pembuat dan Manajer juga boleh.
- `ErrForbidden` polos yang masih lolos ke handler mendapat pesan cadangan netral "Kamu tidak punya hak melakukan ini pada task ini.". `TestNoBareErrForbidden` memindai kode package task supaya `return ErrForbidden` polos tidak muncul lagi.
- Test: `TestForbiddenMessagesPerAction` (21 jenis penolakan: pesan unik, menyebut sub task hanya untuk aksi sub task) dan `TestWriteTaskErrorForbiddenMessage` (handler meneruskan pesan, juga untuk galat yang dibungkus). Tidak ada perubahan perilaku hak akses; kode HTTP tetap 403.

## Sprint 11u: Halaman Status hanya untuk Admin

**Tujuan:** revisi 23 Manajer. Halaman Status adalah alat pemeriksaan teknis, jadi hanya Admin yang melihatnya.

- [x] S11u-01 Pemeriksaan endpoint: halaman Status (`pages/StatusPage.vue`) hanya memanggil `GET /api/health`, yang mengembalikan `{"status":"ok","database":"ok"}` atau 503 "Database tidak dapat dihubungi." Tidak ada endpoint lain yang memberi versi, waktu jalan, atau detail sistem.
- [x] S11u-02 Navigasi: menu Status diberi `roles: ['admin']`. Top bar desktop dan menu Lainnya di ponsel sama-sama dibangun dari `visibleNav`, jadi keduanya ikut.
- [x] S11u-03 Router: rute `/status` diberi `roles: ['admin']` dan penanda baru `hideFromOthers`. Peran lain yang membuka URL itu langsung melihat halaman 404 yang sudah ada (`not-found`), dengan URL tetap `/status`.
- [x] S11u-04 `scripts/e2e/uji13.mjs`: `/status` keluar dari daftar halaman Manajer dan Staf, dan bagian baru `E2E_ONLY=status` memeriksa Admin (halaman tampil, ada tautan) serta Manajer dan Staf (judul "Halaman tidak ditemukan", URL tetap, tanpa tautan Status) di 360 dan 1280 px.

**Keputusan:**
- Backend tidak diubah. `/api/health` tetap publik dan minimal karena dipakai healthcheck Docker dan pemeriksaan deploy, dan halaman Status memakainya langsung. Isinya tidak membocorkan apa pun yang tidak bisa diketahui dari fakta bahwa aplikasi menjawab, jadi membatasinya hanya merusak healthcheck tanpa menambah keamanan. Karena tidak ada endpoint yang dibatasi, tidak ada test akses per peran di backend.
- Peran lain mendapat 404, bukan dialihkan ke beranda seperti halaman Pengguna, Pengaturan, dan Audit Log, supaya keberadaan halaman ini tidak terungkap. Perilaku itu hanya untuk rute bertanda `hideFromOthers`; rute lain yang dibatasi peran tetap dialihkan ke beranda seperti sebelumnya. Pengalihannya memakai rute `not-found` dengan `pathMatch` dari path asli dan `replace`, jadi URL di bilah alamat tetap `/status` dan tombol Kembali tidak memantul.
- Pengguna yang belum login dan membuka `/status` tetap diarahkan ke halaman masuk dengan `kembali=/status`, sama seperti rute lain. Setelah masuk, Admin sampai ke Status dan peran lain melihat 404.
- `docs/panduan-tim.md` tidak menyebut halaman Status, jadi tidak diubah.

**Selesai kalau:** `./scripts/check.sh` lolos dan E2E `status` lolos.

**Hasil:** selesai 2026-11-18, `./scripts/check.sh` lolos, E2E `status` 22 pemeriksaan 0 gagal. Tidak ada migrasi dan tidak perlu restart backend.

## Sprint 11v: Lampiran Aktivitas Harian

**Tujuan:** revisi 24 Manajer. Bukti kerja sering berupa tangkapan layar atau video pendek, dan sebelumnya hanya bisa ditempel sebagai tautan Drive. Sekarang berkasnya bisa menempel langsung di jamnya. Sekaligus memperbaiki backup storage, yang akan kehabisan disk kalau setiap lampiran 95 MB tersalin ulang setiap hari.

- [x] S11v-01 Migrasi 000017: tabel `activity_attachments` (user_id, activity_date, slot, nama asli, mime, ukuran, checksum, uploaded_by, created_at), index (user_id, activity_date), CHECK ukuran 95 MB. Tanpa `is_deleted`: hapus berarti baris dan berkasnya hilang, jadi tabel ini masuk daftar DELETE di `deploy/sql/10-grants.sql` (sekarang tujuh tabel).
- [x] S11v-02 Daftar jenis yang boleh diperluas (gambar termasuk WebP dan GIF, video MP4, MOV, dan WebM, PDF, dokumen Office, TXT, CSV, ZIP), tetap diperiksa dari magic bytes. SVG dan program ditolak.
- [x] S11v-03 API: `POST /api/activities/files`, `GET /api/activities/files/:id`, `DELETE /api/activities/files/:id`. Hak sama dengan isian aktivitas: Staf hanya miliknya hari ini, Manajer siapa pun dan tanggal berapa pun, Admin hanya membaca. Perubahan Manajer pada jam orang lain tercatat di Audit Log.
- [x] S11v-04 Penyajian: gambar, video, dan PDF boleh inline, sisanya selalu `attachment`, selalu dengan `X-Content-Type-Options: nosniff` dan `Content-Security-Policy: sandbox`. Video memakai `http.ServeContent`, jadi Range (206) jalan.
- [x] S11v-05 Nginx: `client_max_body_size 96m` hanya di `location = /api/activities/files`, dengan timeout lebih panjang dan `proxy_request_buffering off`. Batas lampiran task tetap 51 MB. CSP menambah `media-src 'self' blob:`.
- [x] S11v-06 Kartu di papan task: area lampiran di bawah Output tiap jam, tombol "Tambah berkas" berikon klip, seluruh sel jam menerima drag and drop dengan sorotan, Ctrl+V tangkapan layar mengunggah ke jam itu, bar progres dengan persentase dan tombol Batal, galat per berkas dengan "Coba lagi", thumbnail gambar yang diperkecil di browser, pratinjau, unduh, dan hapus dengan konfirmasi. Tampilan kartu ponsel memakai komponen yang sama.
- [x] S11v-07 Halaman `/aktivitas`: sel menampilkan ikon klip, jumlah berkas, dan thumbnail gambar pertama; dialog sel menampilkan semua lampiran dengan pratinjau, dan Manajer bisa menambah atau menghapus dari mode Ubah.
- [x] S11v-08 Ekspor Excel: nama lampiran ikut di kolom Output, dipisah baris baru. Berkasnya tidak disematkan.
- [x] S11v-09 Backup storage jadi snapshot inkremental berbasis hardlink di `backup/storage-snapshots/<stamp>`, retensi harian dan mingguan sama seperti sebelumnya, langkah restore di runbook diperbarui.
- [x] S11v-10 Test: unit (magic bytes jenis baru, SVG dan EXE yang diganti ekstensi, kolom Output ekspor), DB (hak unggah per peran dan tanggal, hapus menghapus berkas, batas 10 per jam, akhir pekan berisi lampiran tetap tampil), dan bagian `E2E_ONLY=activityfiles`.

**Keputusan:**
- Lampiran menempel pada satu jam (`user_id`, `activity_date`, `slot`), bukan pada kolom Aktivitas atau Output. Yang ditanya Manajer selalu "apa buktinya jam itu", dan memisahkan berkas per kolom hanya menambah pilihan yang tidak berarti apa-apa.
- Tidak ada kolom "nama simpan" tersendiri. Nama berkas di disk adalah `id` barisnya, sama seperti `task_attachments`, dan `storage.PathFor(id)` yang menurunkan path-nya. Kolom kedua yang isinya UUID yang sama hanya menunggu jadi tidak sinkron.
- Berkas lampiran aktivitas dihapus permanen, berbeda dari lampiran task yang hanya ditandai `is_deleted`. Satu video 95 MB yang disimpan terus setelah dihapus akan menghabiskan disk VM, sedangkan lampiran task dibatasi 50 MB dan jumlahnya jauh lebih sedikit. Karena itu tabel ini perlu hak DELETE, dan daftar DELETE di grants naik dari enam menjadi tujuh tabel.
- Berkasnya menumpang folder `attachments/` yang sama dengan lampiran task, jadi `ListAttachmentFileIDs` di job pembersih berkas yatim wajib ikut membaca `activity_attachments`. Tanpa itu job harian akan menghapus seluruh lampiran aktivitas dalam satu malam.
- Batas 95 MB, bukan 100 MB: Cloudflare menolak unggahan di atas 100 MB dengan halaman galatnya sendiri, yang tidak bisa dijelaskan aplikasi. Batas 10 berkas per jam menjaga satu jam tidak dipakai sebagai gudang.
- Jenis berkas diperiksa dari magic bytes dengan daftar izin terpisah (`attachment.DetectActivity`), bukan dengan melonggarkan daftar lampiran task. Lampiran task tetap tanpa video dan ZIP, karena di sana isinya brief dan hasil kerja yang memang dokumen. SVG ditolak di kedua daftar: isinya XML yang bisa memuat skrip.
- Gambar, video, dan PDF disajikan `inline` supaya pratinjau bisa langsung dibuka, tetapi tetap dengan `nosniff` dan `Content-Security-Policy: sandbox`. ZIP dan dokumen Office tidak pernah `inline`. Pratinjau di browser mengambil berkasnya sebagai blob, memakai komponen `AttachmentPreview` yang sudah ada, ditambah cabang video.
- Thumbnail diperkecil di browser (createImageBitmap plus canvas), bukan di server. Membuat thumbnail di server berarti menyimpan berkas kedua, menambah pembersih berkas yatim yang kedua, dan menambah kolomnya di database, hanya untuk gambar yang jumlahnya belasan per hari.
- Nama berkas tangkapan layar yang ditempel mengikuti jamnya (`tangkapan-09.00-1.png`), dan ekstensinya mengikuti jenis gambarnya, karena backend menolak isi yang tidak cocok dengan ekstensi.
- Backup storage memakai klona hardlink (`cp -al`) lalu menyalin hanya berkas baru dan membuang yang sudah hilang, bukan `rsync --link-dest`. rsync tidak ada di image `alpine` yang dipakai skrip, dan memasangnya butuh jaringan, sedangkan job ini sengaja berjalan dengan `--network none`. Berkas storage tidak pernah diubah di tempat (namanya UUID barisnya), jadi hasilnya setara.
- Retensi snapshot dijalankan di dalam container yang sama, sebagai UID pemilik storage, karena folder snapshot dimiliki UID itu dan user VM tidak bisa menghapus isinya. Penanda mingguan berupa berkas `.weekly` di dalam snapshot, bukan salinan terpisah, karena snapshot tidak bisa di-hardlink sebagai satu kesatuan.
- Arsip `storage-*.tar.gz` lama tidak dihapus. Snapshot jalan di sampingnya dan arsip lama hilang sendiri lewat retensi yang sudah ada, jadi tidak ada satu hari pun tanpa cadangan storage yang bisa dipakai.

**Selesai kalau:** `./scripts/check.sh` lolos, migrasi 000017 dan `10-grants.sql` jalan di dev (daftar DELETE tujuh tabel), Staf bisa menarik dan menempel berkas di papan, Manajer bisa menambah dan menghapus dari dialog sel, pratinjau gambar, video, dan PDF terbuka, `E2E_ONLY=activityfiles` lolos, dan `backup.sh` dua kali berturut-turut di VM menghasilkan dua snapshot dengan total ukuran kira-kira sebesar satu snapshot.

**Hasil:** kode selesai 2026-11-18, `./scripts/check.sh` lolos, `go vet ./...` bersih, test unit backend dan frontend lolos. Logika snapshot diuji di laptop dengan dua run berturut-turut: berkas yang tidak berubah punya dua hardlink, berkas yang dihapus dari storage hilang dari snapshot baru, snapshot harian berumur 20 hari dibuang dan yang bertanda mingguan tetap. Migrasi, grants, uji curl, uji browser, dan uji backup di VM menunggu Rifky.

## Sprint 11w: Salinan ke Google Drive

**Tujuan:** revisi 25 Manajer. Semua berkas aplikasi perlu salinan di Google Drive akun tim dengan struktur yang bisa dibuka orang tanpa aplikasi, supaya tim punya arsip yang bisa dibaca dan dicari sendiri. VM tetap tempat utama; Drive adalah arsip.

- [x] S11w-01 Migrasi 000018: `drive_exports` (berkas yang sudah terunggah), `drive_folders` (nama folder task dan sub task yang sudah ditetapkan), `drive_export_runs` (status run terakhir untuk halaman Status). Ketiganya hanya ditambah dan dibaca, jadi daftar DELETE di `deploy/sql/10-grants.sql` tetap tujuh tabel.
- [x] S11w-02 Paket `internal/driveexport`: aturan nama Drive (karakter terlarang dibuang, spasi dirapikan, batas 120 karakter, bentrok diberi " (2)"), nama bulan dan hari bahasa Indonesia, dan penyusun path untuk aktivitas, Brief, Deliverable, sub task, ekspor bulanan, dan dump.
- [x] S11w-03 Perintah `server drive-export stage|record|fail` dengan `--dry-run`: menyusun pohon symlink bernama rapi di folder staging, menulis ekspor Excel bulan berjalan lewat kode ekspor Aktivitas yang sudah ada, dan mengenkripsi dump dengan age.
- [x] S11w-04 `deploy/drive-export.sh`: tiga langkah (stage, rclone copy, record), log ke `backup/drive.log` dengan pola yang sama seperti `backup.log`, dan `--dry-run` untuk memeriksa struktur sebelum unggahan pertama.
- [x] S11w-05 Endpoint `GET /api/drive-export/status` hanya untuk Admin, dan halaman Status menampilkan "Salinan Drive terakhir" berisi waktu, OK atau galat beserta pesannya, dan jumlah berkas.
- [x] S11w-06 Bagian baru "Salinan ke Google Drive" di `docs/runbook-deploy.md`: membuat remote rclone dengan scope `drive.file` lewat `rclone authorize` di laptop, membuat pasangan kunci age dan menyimpan kunci privatnya di dua tempat di luar VM, uji kering, unggah pertama, pemulihan dump, dan baris cron.
- [x] S11w-07 Test: unit (pembersihan nama, penomoran bentrok, path setiap jenis, nama bulan dan hari) dan DB (berkas yang sudah tercatat tidak disusun lagi, `is_deleted` dilewati, ganti judul task tidak mengubah path yang sudah tercatat).

**Keputusan:**
- Rancangannya tetap seperti usulan Manajer: perintah di binary backend menyusun pohon tautan di staging, lalu container rclone resmi yang mengunggah. Alternatif yang dipertimbangkan dan ditolak: memakai Drive API langsung dari Go (menambah pustaka Google, menangani OAuth dan percobaan ulang sendiri, padahal rclone sudah melakukannya), dan memanggil `rclone copyto` satu kali per berkas supaya tidak perlu staging (satu proses rclone per berkas, lambat dan ratusan kali lebih mudah gagal di tengah).
- Hardlink, bukan salinan: staging satu filesystem dengan storage, jadi pohon bernama rapi itu tidak memakan disk sama sekali. Staging ditaruh di `/srv/nvm-internal/drive-staging`, di luar folder storage, supaya tidak ikut terbawa snapshot backup (kalau di dalam storage, berkasnya akan tersalin sungguhan ke snapshot setiap malam).
- `stage` dan `record` dipisah, dengan rclone di antaranya. Dengan begitu "setiap berkas hanya diunggah sekali" tidak berubah jadi "hanya dicoba sekali": baris `drive_exports` baru ditulis setelah unggahan berhasil, jadi unggahan yang gagal dicoba lagi malam berikutnya.
- Dump dikirim ke perintah `stage` sebagai tar lewat stdin, bukan dengan memberi container itu akses ke folder backup. Dump dibuat `backup.sh` dengan mode 0600 milik user VM, dan container ini berjalan sebagai UID 65532 pemilik storage. Lewat pipe, tidak ada izin yang perlu dilonggarkan dan tidak ada salinan plaintext tambahan di disk.
- Yang memilih dump mana yang diunggah adalah skrip (3 harian terakhir dan 8 salinan mingguan), karena skrip yang melihat folder backup. Perintah Go hanya mengenkripsi apa pun yang datang di tar. Dump tidak dicatat di `drive_exports`: rclone sendiri melewati berkas yang sudah sama, dan dump harian yang sama memang diunggah beberapa malam berturut-turut.
- Ekspor Excel bulan berjalan ditimpa setiap malam, jadi juga tidak dicatat di `drive_exports`. Yang dicatat hanya lampiran task dan lampiran aktivitas, yaitu berkas yang memang tidak pernah berubah isinya.
- Enkripsi age dikerjakan di dalam backend dengan pustaka `filippo.io/age`, bukan dengan biner `age` di VM. Image backend distroless tidak punya shell atau biner tambahan, dan menambah image lagi hanya untuk satu langkah enkripsi lebih mahal daripada satu pustaka Go. Kunci privat tidak pernah ada di VM: yang disimpan di `deploy/.env` hanya kunci publik (recipient). Kalau `DRIVE_BACKUP_AGE_RECIPIENT` kosong, berkas aplikasi tetap diunggah dan dumpnya tidak, dengan alasannya tertulis di log.
- Scope rclone `drive.file`, bukan `drive`. rclone hanya bisa melihat dan mengubah berkas yang dia buat sendiri, jadi token yang bocor dari VM tidak membuka seluruh Drive akun tim. Konsekuensinya rclone tidak bisa memakai folder yang dibuat orang lewat browser, karena itu folder induk dibuat rclone sendiri.
- `rclone.conf` disimpan di `/srv/nvm-internal/secrets/`, di luar repo, dimiliki UID 65532 dengan mode 600. Foldernya yang di-mount, bukan berkasnya, karena rclone menulis ulang berkas itu setiap kali token OAuth diperbarui dan penulisan itu lewat rename di dalam folder.
- Nama folder task dan sub task ditetapkan sekali di `drive_folders`. Judul yang diganti di aplikasi tidak membuat folder kedua di Drive, dan berkas lama tidak perlu dipindah. ID task ikut di nama folder, jadi dua task berjudul sama tidak pernah bentrok; penomoran " (2)" hanya diperlukan untuk sub task dan nama berkas di dalam satu folder.
- Berkas yang sudah tercatat tidak disusun lagi, jadi pohon staging setiap malam hanya berisi berkas baru. Akibatnya `rclone copy` hampir selalu memindahkan sedikit data, dan berkas yang dihapus di aplikasi tidak pernah diunggah lagi tanpa harus menghapus apa pun di Drive.
- Berkas yang barisnya ada tetapi berkasnya hilang dari disk dicatat di log dan dilewati, tidak membatalkan seluruh salinan malam itu.
- Halaman Status membaca `drive_export_runs`, bukan `backup/drive.log`. Backend di dalam container tidak punya akses ke folder log host, dan membaca berkas log dari HTTP akan menambah jalan baru ke filesystem host tanpa alasan.
- Status salinan hanya untuk Admin, mengikuti revisi 23 yang menjadikan halaman Status milik Admin. `/api/health` tidak diubah.

**Keputusan tambahan setelah uji kering pertama di VM gagal (2026-11-18):**
- Pemilihan dump memakai `find`, bukan `ls` dengan glob. Pola yang tidak cocok dengan apa pun (belum ada backup mingguan) membuat `ls` keluar dengan kode bukan nol, dan dengan `pipefail` plus `set -E` itu memanggil `fail()` di dalam process substitution. Pemotongan jumlah memakai `awk`, bukan `head`, karena `head` menutup pipa lebih awal sehingga `find` kena SIGPIPE dan pipeline-nya gagal juga.
- `fail()` berhenti di awal bila dipanggil dari subshell (`$BASHPID` berbeda dari `$$`): hanya melaporkan ke stderr lalu keluar, tanpa menghapus berkas sementara, tanpa menulis log, dan tanpa menulis baris status. Tanpa itu, satu kegagalan di process substitution menghapus `$err_file` lebih dulu, sehingga pesan galat langkah berikutnya hilang dan yang tersisa hanya log Compose.
- Pesan galat langkah "susun staging" harus pesan asli perintah Go. Compose menulis baris "Container ... Creating" dan "Created" ke stderr, jadi ikut tersimpan. Sekarang `--progress quiet` dipakai bila Compose mendukungnya (v2.17 ke atas, dicoba dulu supaya versi lama tetap jalan), dan `last_error()` menyaring sisa baris progres apa pun.
- Folder `/srv/nvm-internal/secrets` mode 711, bukan 700. Isinya milik UID 65532, sedangkan skrip berjalan sebagai user VM dan memeriksa dulu apakah `rclone.conf` ada; dengan 700 pemeriksaan itu selalu gagal. 711 memberi hak menelusuri tanpa hak membaca daftar isi, dan berkasnya tetap 600 milik 65532.
- Izin eksekusi `deploy/*.sh` diperiksa `scripts/check.sh` dari indeks git (harus 100755). Di Windows git tidak merekam bit itu sendiri, dan `drive-export.sh` ter-commit 100644 sehingga cron tidak bisa menjalankannya. Perilaku skripnya diuji `scripts/uji-drive-export.sh`, yang mengganti `docker` dengan stub dan menjalankan empat kasus: tanpa berkas `weekly-*`, tanpa `db-*` sama sekali, keduanya ada, dan satu kegagalan buatan di langkah stage.

**Keputusan tambahan setelah langkah stage gagal di VM (2026-11-18):**
- Folder staging tidak pernah dihapus, hanya dikosongkan isinya. Di container folder itu adalah titik mount, dan induknya (`/data`) ada di root filesystem yang `read_only`, jadi `os.RemoveAll` pada foldernya selalu gagal dengan `unlinkat ...: read-only file system`. Pesan itu terbaca seperti mount yang salah, padahal mount-nya benar.
- Pohon staging memakai symlink, bukan hardlink. Storage dan staging adalah dua bind mount berbeda di dalam container, dan kernel menolak hardlink antar mount dengan EXDEV walaupun host-nya satu disk, karena `do_linkat` membandingkan vfsmount, bukan filesystem. Alternatif yang ditimbang: memount folder induk `/srv/nvm-internal` sebagai satu mount (ikut membuka `secrets/` dan `backup/` ke container backend, ditolak), memindahkan storage ke bawah folder induk baru supaya keduanya satu mount (menyentuh data paling berharga di sistem yang sudah dipakai, dan mengubah semua path di runbook), dan menaruh staging di dalam storage (hardlink jalan, tetapi berkas dump terenkripsi di staging akan ikut masuk snapshot backup setiap malam, yang justru masalah yang diperbaiki revisi 24). Symlink menghapus seluruh syarat satu filesystem, dan satu-satunya syarat barunya terlihat jelas: container rclone memount storage di path yang sama (`/data/storage`) dan memakai `--copy-links`. Symlink yang menggantung membuat rclone keluar dengan kode bukan nol, jadi path yang tidak cocok langsung kelihatan.
- Semua log subperintah pindah ke stderr (`newLogger`). `drive-export.sh` membaca stdout sebagai daftar path dan stderr sebagai pesan galat; selama log JSON masih di stdout, daftar path tercemar dan pesan kegagalannya kosong. Ukuran staging dibaca dengan `du -shL`, karena tanpa mengikuti symlink angkanya selalu nol.
- `secrets/` dan `drive-staging/` masuk `.gitignore`: repo di VM ada di `/srv/nvm-internal`, jadi keduanya berada di dalam working tree, dan git memperingatkan karena tidak bisa membaca `secrets/` (mode 711 milik UID 65532).

**Keputusan tambahan setelah uji kering kedua di VM (2026-11-18):**
- Uji kering mencetak ringkasan per jenis (brief, deliverable, sub task, aktivitas, ekspor bulanan, dump), termasuk jenis yang nol, ditambah daftar alasan berkas dilewati beserta jumlahnya. Penyebabnya nyata: di produksi ada 6 lampiran task aktif dan nol yang masuk staging, dan keluarannya tidak menyebut apa pun. Job malam yang tidak menyalin apa pun tidak boleh terlihat sama dengan job yang berhasil.
- Alasan dilewati yang hanya terlihat dari database (lampiran sudah dihapus, task sudah dihapus, sudah tercatat di `drive_exports`) dihitung lewat query sendiri, bukan ditebak dari selisih angka. Alasan yang terlihat dari disk (berkas hilang di storage, id tidak sah, symlink gagal) dihitung di tempat kejadiannya.
- Lampiran milik task yang sudah dihapus memang tidak disalin, dan sekarang jumlahnya tercetak. Kalau Manajer menghapus task, isinya tidak ikut diarsipkan. Mengubah aturan ini satu baris di `ListTaskFilesForDrive`, tetapi tidak diubah tanpa keputusan Manajer.
- Ringkasan dan peringatan ditulis ke stderr, dan cabang uji kering di `drive-export.sh` sekarang mencetak stderr itu. Sebelumnya stderr perintah Go ditangkap ke berkas sementara lalu dibuang, jadi ringkasan sebagus apa pun tidak akan pernah terlihat lewat skrip.
- Penyebab 6 lampiran task tidak masuk staging, ditelusuri dari kode: keluaran stage langsung di VM tidak memuat satu pun peringatan `berkas tidak ada di storage` atau `symlink tidak bisa dibuat`, jadi `link()` tidak pernah dipanggil dan `ListTaskFilesForDrive` memang mengembalikan nol baris. Query itu hanya punya tiga saringan: lampiran `is_deleted`, task `is_deleted`, dan baris `drive_exports` yang sudah ada. Yang pertama gugur karena hitungan produksi sudah `NOT is_deleted`. Join tidak mungkin terlalu sempit (`task_id` NOT NULL dengan FK ke `tasks`, sub task lewat LEFT JOIN), dan kind tidak mungkin tak dikenali (enum hanya `brief` dan `deliverable`, dan kind lain jatuh ke Deliverable, tidak dibuang). Jadi sebabnya tinggal dua: task-nya sudah dihapus (hitungan produksi tidak melihat `tasks.is_deleted`), atau barisnya sudah ada di `drive_exports`. Mana yang benar dibuktikan dengan perintah baca di runbook bagian Salinan ke Google Drive.
- Jalur yang bisa mengisi `drive_exports` tanpa unggahan ditutup: uji kering dulu tetap menulis manifest, jadi `drive-export record` yang dijalankan sesudahnya mencatat semua lampiran sebagai sudah terunggah, dan sejak itu lampiran tersebut tidak pernah masuk staging lagi tanpa peringatan. Sekarang uji kering tidak menulis manifest, sehingga `record` sesudahnya gagal dengan "manifest staging tidak terbaca". Test DB `TestUjiKeringTidakMeninggalkanManifest` gagal pada kode lama.
- Alasan dilewati dicatat sebagai jumlah per alasan, bukan daftar per berkas. Alasan dari database (sudah dihapus, sudah tercatat) terus bertambah seumur aplikasi, jadi daftar per berkas akan memenuhi log setiap malam. Alasan dari disk tetap punya satu baris peringatan per berkas beserta path-nya, dan rincian per berkas dari database ada di query baca runbook.
- Satu tanggal satu dump. Nama di Drive dibuat per tanggal, jadi tiga dump dari tiga kali backup di hari yang sama saling menimpa. Yang dipakai dump terbaru per tanggal, dan jatah harian plus mingguan disaring sekali lagi bersama supaya dump mingguan yang bertanggal sama tidak terkirim dua kali. Stamp lengkap di nama berkas sengaja tidak dipakai: arsip yang dibaca orang lebih berguna satu per hari, dan VM tetap menyimpan semua dump harian apa adanya. Perintah Go juga melewati dump kedua dengan tanggal sama sebagai pengaman.

**Keputusan setelah unggahan pertama di VM (2026-11-19):**
- rclone memakai client OAuth milik tim sendiri (project Google Cloud, OAuth client Desktop app), bukan client bersama bawaan rclone. Client bersama sedang dipensiunkan Google dan kuotanya dibagi dengan semua pengguna rclone, jadi job malam bisa gagal karena kuota orang lain.
- Aplikasi OAuth dipublikasikan **In production tanpa verifikasi merek**: tanpa logo, Branding "not shown", dan verifikasi tidak diajukan. `drive.file` bukan scope sensitif, jadi itu cukup. Status Testing tidak dipakai karena tokennya kedaluwarsa setelah 7 hari. Harganya layar "Google hasn't verified this app" saat izin pertama, yang hanya dilihat Rifky sekali.
- Halaman privasi yang diminta Google adalah berkas statis `frontend/public/privasi/index.html`, di luar aplikasi Vue, supaya tayang tanpa login. Nginx menyajikannya lewat location `/privasi/` sendiri: `try_files $uri` di location `/` tidak pernah mencocokkan folder, jadi `/privasi/` sebelumnya jatuh ke SPA (produksi menjawab dengan title "NVM Internal"). Gayanya dipindah ke `privasi/gaya.css`, karena CSP `style-src 'self'` memblokir `<style>` inline.
- `drive.log` selalu satu baris per run. Ukuran diambil dari `bytes` di log "staging siap" perintah Go, bukan dari `du -shL`: isi staging adalah symlink ke `/data/storage`, path di dalam container, jadi di host semuanya menggantung. `du` tidak menghitung lampiran sama sekali dan keluar bukan nol, lalu dengan `pipefail` "`|| echo -`" menambah baris kedua yang membelah baris log. `log_line` juga mengganti baris baru di nilai mana pun dengan spasi. `scripts/uji-drive-export.sh` meniru `du` yang gagal itu, dan gagal 16 pemeriksaan pada skrip lama.

**Selesai kalau:** `./scripts/check.sh` lolos, migrasi 000018 jalan di dev, `drive-export.sh --dry-run` di VM mencetak struktur yang sesuai, unggahan pertama muncul di Drive akun tim, halaman Status (Admin) menampilkan hasilnya, dan dump yang diunduh dari Drive bisa didekripsi dengan kunci privat age di laptop.

**Hasil:** kode selesai 2026-11-18, `./scripts/check.sh` lolos, `go vet ./...` bersih, test unit dan test DB baru lolos. Satu dependensi Go baru: `filippo.io/age v1.2.1`. Uji kering pertama di VM gagal karena tiga hal di skripnya (pemilihan dump, `fail()` di subshell, dan log Compose yang menutupi pesan galat), ketiganya diperbaiki dan diuji ulang di laptop lewat `scripts/uji-drive-export.sh`. Migrasi, langkah persiapan (rclone authorize, kunci age, secrets, cron), uji kering ulang, dan unggah pertama menunggu Rifky.

## Sprint 11x: Panel Buat task, pemilih berkas, dan halaman penuh

**Tujuan:** revisi 26 Manajer. Dua bug di panel Buat task dan satu fitur: form Buat task bisa dibuka di tab baru sebagai halaman penuh.

- [x] S11x-01 Bug A, pemilih berkas yang dibatalkan menutup panel. Penyebabnya terbukti di browser: input berkas mengirim event `cancel` yang bubble saat pemilih berkas ditutup tanpa memilih, dan `BaseDialog` menangani semua `cancel` yang sampai ke `<dialog>` seolah-olah Escape. Event `cancel` tiruan yang bubble dari input berkas menutup panel di 360 dan 1280 px sebelum perbaikan. Sekarang `onCancel` hanya menangani cancel yang `target`-nya dialog itu sendiri.
- [x] S11x-02 Bug B, isi panel melompat naik saat "pilih berkas" diklik. Diukur di panel sungguhan, 1280 px: wadah gulir isian tetap di `scrollTop` 736, tetapi `<dialog>` sendiri bergeser dari `scrollTop` 48 ke 287. Input berkas `sr-only` (absolute) induknya `<dialog>`, bukan wadah gulir, jadi posisinya tidak ikut tergulir (top 1138 px, padahal label-nya di 339 px). Klik label memfokuskan input itu, dan browser menggulir dialog supaya input terlihat, walau dialognya `overflow-hidden`. Di 360 px sama: 200 ke 439. Sesudah perbaikan, dialog tetap 0 dan wadah gulir tidak berubah di kedua lebar.
- [x] S11x-03 Fitur C: isi form dipisah ke `TaskForm.vue`, dipakai `TaskFormPanel` dan halaman baru `/task/baru` (`TaskNewPage.vue`). Panel Buat task punya tombol "Buka di tab baru" di kepala panel, sama dengan panel detail. Halaman penuh punya PageHeader "Buat task" dan bar aksi Batal dan Buat task di bawah form.
- [x] S11x-04 E2E `E2E_ONLY=createtask` di `scripts/e2e/uji13.mjs`, plus penyesuaian urutan Tab di bagian `activity`.

**Keputusan:**
- Bug B diperbaiki di akarnya, di dua tempat. Wadah gulir `BaseDialog` diberi `relative`, jadi semua elemen absolute di isian dialog (input berkas, radio Urgency dan Jadwal yang `sr-only`) berinduk wadah gulir dan ikut tergulir. Komponen unggah (`FileDropzone`, `ActivityFiles`, `SubtaskFiles`, `ProfileForm`) juga membawa induk `relative` sendiri, supaya aman di wadah gulir mana pun, termasuk di luar dialog. Ini kelanjutan catatan bug bar aksi Sprint 11j: `overflow-hidden` di panel menghentikan gulir oleh pengguna, tetapi tidak menghentikan gulir oleh fokus.
- Bug A diperbaiki di `BaseDialog`, bukan di tiap input, jadi semua input berkas di dalam dialog ikut aman: form Buat task, lampiran di detail task, berkas sub task, dan berkas Aktivitas di dialog `/aktivitas`. Berkas Aktivitas di papan tidak berada di dalam dialog. Escape tetap menutup dialog, dan Escape bertahap di dialog Aktivitas (`close('escape')`) tetap bekerja, karena event cancel dari Escape memang ber-`target` dialog. Efek sampingnya baik: dialog yang bersarang di dalam dialog lain tidak ikut menutup induknya.
- Input berkas tersembunyi di `ActivityFiles` dan `SubtaskFiles` diberi `tabindex="-1"` dan `aria-hidden`, sama dengan `ProfileForm`. Input itu dibuka lewat tombol di bawahnya, jadi dulu ia tab stop tanpa nama dan tanpa fokus terlihat: E2E `activity` menemukan Tab dari Output 09.00 berhenti di elemen tanpa nama. Input di `FileDropzone` tetap bisa di-Tab, karena di sanalah fokusnya (label pembungkus menampilkan focus ring lewat `has-[:focus-visible]`).
- Membuka tab baru dari panel yang sudah terisi tidak membawa isian. Tab baru membuka form kosong, dan panelnya tidak ditutup, jadi isian di panel tidak hilang. Membawa isian butuh jalur penyimpanan sementara (termasuk berkas brief yang belum terunggah), dan itu tidak sebanding dengan kebutuhannya.
- `/task/baru` hanya untuk Manajer, sama dengan `POST /api/tasks`. Peran lain melihat halaman 404 dengan URL tetap (`hideFromOthers`, pola revisi 23). Rute statis selalu menang atas `/task/:id` di vue-router, jadi "baru" tidak pernah terbaca sebagai ID task.
- Setelah task dibuat di halaman penuh, aplikasi pindah ke `/task/<id>` dengan `replace`, supaya tombol Kembali tidak membuka lagi form yang sudah terkirim. Pesan hasilnya (termasuk lampiran yang gagal diunggah) ikut lewat `history.state` dan muncul di halaman task. Batal kembali ke `/task`.
- Halaman penuh memuat daftar akun dan label sebelum form tampil, karena PIC harus dipilih dari daftar itu. Ada state memuat dan state galat dengan tombol Muat ulang.

**Selesai kalau:** `./scripts/check.sh` lolos dan E2E `createtask` lolos.

**Hasil:** selesai 2026-11-18. `./scripts/check.sh` lolos. E2E `createtask` 26 pemeriksaan 0 gagal. Regresi: `activity` 179/0, `activityfiles` 22/0, `schedule` 20/0, `subtaskform` 18/0, `subtaskresult` 10/0. Tanpa perubahan backend dan migrasi.

## Sprint 11y: Semua anggota membuat task, To Do pribadi, dan perbaikan task berulang

**Tujuan:** revisi 27 dan 28 Manajer, ditambah tiga perbaikan bug revisi 19 yang terlihat di produksi.

- [x] S11y-01 Revisi 27: `Create` menerima Staf dan Manajer dengan aturan yang sama persis (PIC, sub task lengkap, jadwal berulang, label, lampiran brief, deadline). Admin ditolak. Middleware peran di rute `POST /api/tasks`, `PUT /api/tasks/:id`, dan `DELETE /api/tasks/:id` dilepas, dan haknya dicek di service yang memegang status serta pembuat task.
- [x] S11y-02 Revisi 27: pembuat task boleh mengubah dan menghapus task buatannya selama status masih Requested (`CanEditOwn`), dicek dari baris yang sudah dikunci di dalam transaksi. `Can.edit` dan `Can.delete` di detail ikut aturan itu.
- [x] S11y-03 Revisi 27: `can_create_task` dikirim server di `/api/auth/me`. Tombol "Buat task" dan rute `/task/baru` memakai nilai itu, bukan daftar peran di frontend. Admin tetap melihat 404 di `/task/baru`. Form task memakai `/api/people` untuk kandidat PIC bila pemakainya bukan pengelola.
- [x] S11y-04 Revisi 28: migrasi 000019 `user_todos` (user_id, title maksimal 200, done_at, position, created_at, updated_at), index (user_id, position), batas 100 item per akun. Tabel ini perlu hak DELETE, jadi daftar DELETE di `deploy/sql/10-grants.sql` jadi delapan tabel.
- [x] S11y-05 Revisi 28: tujuh endpoint `/api/todos` (daftar, tambah, ubah judul, centang, hapus, urutkan ulang, bersihkan yang selesai). Semuanya memakai akun yang login; id milik orang lain dijawab 404.
- [x] S11y-06 Revisi 28: kartu To Do di kolom kiri papan, tombol "To Do (n)" dengan panel di lebar yang memakai bar, dan lembar dari bawah di bawah 640 px. Tambah lewat Enter, ubah di tempat, centang, urutkan dengan drag dan dengan tombol naik dan turun, hapus dengan toast Urungkan 6 detik, bersihkan yang selesai, dan pembaruan optimistis.
- [x] S11y-07 Bug revisi 19 nomor 1: detail task kini mengirim `occurrence_date` dan `series_freq`, jadi "Kemunculan" tidak lagi selalu "-".
- [x] S11y-08 Bug revisi 19 nomor 2: detail seri membawa `template_subtasks` (judul dan PIC dari templat) dan `template_matches`, dan bagian Jadwal menampilkan keduanya.
- [x] S11y-09 Test: DB untuk revisi 27 (Staf membuat task dengan sub task dan jadwal, Admin ditolak, pembuat mengubah dan menghapus task Requested, ditolak setelah On Progress, Staf lain ditolak), DB untuk To Do (hanya pemilik, 404 untuk id orang lain, batas 100, urutkan ulang hanya id sendiri, bersihkan hanya milik sendiri), DB untuk bug revisi 19 (jadwal lewat Ubah mengisi occurrence_date, ticker tidak dobel, sub task templat di detail), dan bagian `E2E_ONLY=todo`.

**Keputusan:**
- Hak membuat task ditentukan satu fungsi, `task.CanCreateTask`, yang dipakai service dan `/api/auth/me`. Frontend tidak pernah menulis daftar perannya sendiri, jadi aturan baru cukup diubah di satu tempat.
- Pengecualian pembuat task sengaja sempit: hanya ubah dan hapus, hanya selama Requested, dan statusnya dibaca dari baris yang sudah dikunci di dalam transaksi. Permintaan yang datang tepat setelah task pindah status tetap ditolak, bukan lolos karena memakai status dari permintaan.
- Pengecualian itu juga dibatasi peran yang boleh membuat task. Task hasil impor Excel bisa punya `requested_by` milik Admin, dan Admin tetap tidak boleh mengubah task; tanpa batasan ini Admin akan mendapat tombol Ubah di task impor yang masih Requested.
- `stop-series` tetap hanya Manajer. Menghentikan pengulangan mengubah kemunculan yang akan datang untuk semua orang, bukan hanya task yang sedang dilihat, jadi itu bukan bagian dari "mengubah task buatannya". Pembuat task yang masih Requested tetap bisa mengganti atau mematikan jadwalnya lewat form Ubah.
- Kartu papan dan Arsip tidak ikut mengirim hak ubah dan hapus per kartu: keduanya hanya memakai `move_to`, dan tombol Ubah serta Hapus ada di detail, yang sudah membaca pembuat task. Dengan begitu query papan tidak perlu menambah kolom.
- Staf memakai `/api/people` untuk kandidat PIC, karena `/api/users` hanya untuk pengelola. Satu tipe `PicCandidate` dipakai form, dan isinya hanya kolom yang benar-benar dipakai, jadi tidak ada kolom karangan untuk menyamakan dua bentuk data.
- To Do memakai tabel sendiri, bukan task dengan penanda. To Do tidak punya PIC, status, divisi, riwayat, atau audit, dan menempelkannya ke tabel tasks akan membuat setiap query papan dan Arsip harus menyaringnya.
- To Do tidak dicatat di audit log, karena ini catatan pribadi. Konsekuensinya item yang dihapus benar-benar hilang, dan itu sebabnya hapus memberi enam detik untuk Urungkan alih-alih meminta konfirmasi.
- Id milik orang lain dijawab 404, bukan 403, dan semua query memakai `WHERE id = ... AND user_id = ...`, jadi jawabannya sama untuk item yang tidak ada dan item milik orang lain. Tidak ada endpoint untuk membaca To Do orang lain, Manajer dan Admin tidak terkecuali.
- Urutkan ulang menolak seluruh permintaan bila ada satu id yang bukan milik pengirim, bukan melewati id itu saja. Melewatinya diam-diam akan membuat urutan berubah separuh dan menjadi cara menebak id orang lain.
- Daftar To Do dipakai bersama satu composable (`useTodos`) dengan state di tingkat modul. Kartu kolom kiri dan panel bar sama-sama ada di DOM di lebar tertentu; tanpa state bersama keduanya akan memuat sendiri dan bisa menampilkan daftar yang berbeda setelah jendela diubah ukurannya.
- Pembaruan optimistis selalu menyimpan salinan daftar sebelum berubah dan mengembalikannya bila server menolak, lalu menampilkan pesan galatnya. Tanpa salinan itu, kegagalan jaringan akan meninggalkan daftar yang berbeda dari database tanpa pemakai tahu.
- Tombol naik dan turun ada di barisnya sendiri, bukan di samping judul. Di kolom kiri papan yang hanya 17rem, tiga tombol 44 px di samping judul membuat judulnya tinggal beberapa karakter, dan menyembunyikan tombol sampai disorot membuatnya tidak bisa dipakai di layar sentuh.

**Keputusan tampilan kartu To Do (revisi 28 lanjutan, 2026-11-19):**
- Kartu memakai permukaan krem dengan teks tinta, token `tag` dan `tag-ink` yang sama dengan label ID task, karena itu satu-satunya aksen identitas yang sudah ada dan nilainya tidak berganti antara mode terang dan gelap.
- Semua warna di kartu itu diturunkan dari `tag-ink`, bukan dari `ink-muted`, `line`, `danger`, atau `mark`, karena token mode-peka dirancang untuk latar yang ikut berganti dan kontrasnya jatuh di atas krem yang tetap terang (`ink-muted` mode gelap di atas krem hanya 1,9:1).
- Angkanya: teks tinta di atas krem 13,31:1; teks redup `tag-ink/70` 5,56:1; garis dan bingkai kotak centang `tag-ink/60` 4,16:1 (ambang non-teks 3:1); cincin fokus `tag-ink` 13,31:1. Semuanya sama di kedua mode karena kreamnya sama.
- Galat dan toast di dalam kartu memakai teks tinta dengan garis kiri tebal, bukan warna `danger`, dengan alasan yang sama; kata "Gagal" dan `role="alert"` yang membawa artinya, bukan warnanya.
- Item jadi baris yang dipisah garis tipis, bukan kotak sendiri, karena dua kotak dengan kontrolnya memakan setengah kolom kiri papan yang hanya 17rem.
- Kontrol baris (pegangan seret, naik, turun, hapus) tersembunyi sampai barisnya disorot atau difokus, dan hanya di pointer presisi (`pointer-fine`), karena layar sentuh tidak punya hover. Di layar sentuh jalannya tombol "Atur" di kepala kartu, yang menampilkan kontrol di semua baris sekaligus (`aria-pressed`).
- Pegangan seret adalah penanda visual (`aria-hidden`, bukan tombol), karena urutan sudah bisa diubah lewat tombol naik dan turun yang masuk urutan Tab; tombol yang tidak melakukan apa-apa hanya menambah perhentian Tab yang membingungkan.
- Baris tambah menjadi baris terakhir daftar dengan isian tanpa kotak, supaya kartunya terbaca sebagai satu lembar, bukan form di atas daftar. Tombol kirim berikon tetap ada karena di layar sentuh tidak ada tombol Enter yang nyaman.
- Item selesai pindah ke kelompok "Selesai (n)" yang bisa ditutup, bawaan tertutup kalau lebih dari tiga, supaya daftar yang aktif tidak terdorong ke bawah oleh pekerjaan yang sudah lewat. Tombol "Bersihkan" ada di kepala kelompok itu, di tempat yang jelas miliknya.
- Centang digambar 150 ms lewat `stroke-dashoffset` yang terikat ke keadaan itemnya, dan pemindahan ke kelompok Selesai ditunda 180 ms supaya goresannya sempat terlihat di tempatnya. Dengan `prefers-reduced-motion` transisinya mati dan centangnya langsung tampil.
- Kontrol baris tidak boleh mengubah lebar atau tinggi baris saat muncul. Di mode biasa kontrolnya adalah lapisan `absolute` di sisi kanan baris dengan latar krem yang sama dan gradasi ke kiri, jadi judul di bawahnya tetap mendapat seluruh lebar baris dan tingginya tetap 44 px. Di mode "Atur" kontrolnya pindah ke barisnya sendiri di bawah judul, karena di situ semua baris menampilkannya sekaligus dan lapisan akan menutupi akhir judul terus-menerus.
- Judul memakai `min-w-0` dan `overflow-wrap: anywhere`, supaya judul yang isinya URL panjang tanpa spasi patah di lebar baris, bukan memaksa barisnya melebar dan memunculkan gulir horizontal.
- Judul yang memuat URL tetap teks biasa, bukan tautan: mengkliknya membuka mode ubah, yang merupakan aksi utama baris itu. Tautannya dibuka lewat ikon 44 px terpisah dengan nama aksesibel "Buka tautan <judul>", jadi tidak ada satu target yang punya dua arti. Ikon itu ikut kelompok kontrol baris, bukan berdiri di samping judul, karena di mode biasa kontrolnya lapisan di sisi kanan baris dan ikon di luar kelompok itu akan tertutup lapisannya saat barisnya disorot. Di layar sentuh jalannya sama dengan kontrol lain, lewat tombol "Atur".
- Centang hanya tampak kalau itemnya selesai. `stroke-dasharray` dan `stroke-dashoffset` sekarang nilai terikat (24 dan 24 atau 0), bukan `path-length="1"`: atribut SVG-nya `pathLength` dengan huruf besar di tengah, jadi versi sebelumnya diabaikan browser dan `stroke-dasharray: 1` justru menggambar centang bertitik yang terlihat terus di item yang belum selesai.
- Kosong: hanya kepala kartu, satu kalimat "Belum ada to do.", dan baris isian. Tanpa garis kosong, karena garis yang tidak bisa diklik terbaca seperti baris isian yang rusak. Kalimatnya adalah `<label>` untuk isian tambah, jadi mengkliknya memfokuskan isian itu; `aria-label` isian tetap menang sebagai nama aksesibelnya.
- Saat kosong, ringkasan dan tombol "Atur" tidak ditampilkan, karena tidak ada yang bisa diringkas atau diatur. Ringkasannya "<n> tersisa" bila masih ada yang belum selesai, dan "Semua selesai" bila semuanya sudah dicentang, bukan "0 tersisa".

- Escape di isian ubah memakai `.stop`, jadi hanya membatalkan perubahan dan tidak sampai ke pendengar `keydown` di document milik TodoPanel. Tanpa itu, Escape di dalam panel atau lembar menutup seluruh panel sekaligus, dan itu yang membuat E2E mati dengan timeout 30 detik di baris setelahnya: judul yang ditunggu memang sudah tidak ada di DOM. Escape kedua, saat tidak sedang mengubah, tetap menutup panel.

- Isian ubah memakai function ref, bukan `ref="editInput"`. Ref bernama yang dipasang di dalam `v-for` dikumpulkan Vue menjadi array, jadi `editInput.value?.focus` bukan fungsi dan `startEdit` berhenti dengan `TypeError` sebelum isian itu mendapat fokus. Akibatnya Escape diterima tombol judul, bukan isian, jadi penghentian perambatan di isian tidak pernah terpakai dan panelnya tetap tertutup. Satu sebab, dua gejala.
- Escape sekarang lewat `cancelEdit`: keluar dari mode ubah (judul semula kembali, karena `editDraft` tidak pernah dikirim), lalu fokus kembali ke tombol judul lewat id `<uid>-title-<id>`, bukan ke body. Tombol itu dibuat ulang setiap keluar dari mode ubah, jadi elemen pemicunya tidak bisa disimpan. `commitEdit` menolak jalan kalau mode ubah sudah tertutup, supaya blur setelah Escape tidak menyimpan apa pun.
- Id sementara hasil tambah optimistis dipetakan ke id dari server (`swapped`), dan setiap perubahan menunggu tambah yang masih jalan (`settled`). Baris hasil tambah sudah tampil sebelum jawabannya datang, jadi urutkan ulang yang menyusul bisa mengirim id negatif; `PUT /api/todos/order` dengan id yang bukan milik pengirim dijawab 404, dan itu asal 404 di console. Dicek langsung ke backend: `order` dengan `[-1]` 404, `[]` 400, dan rute `/todos/order` tidak tertukar dengan `/todos/:todoId`.

**Keputusan perbaikan E2E To Do:**
- Uji pertama gagal dengan timeout 30 detik di isian, karena kartu To Do hanya muncul sendiri di lebar yang punya kolom kiri; di 360 dan 1280 px panelnya harus dibuka dulu. Sekarang test memeriksa apakah tombol To Do terlihat, mengkliknya, lalu menunggu `role="dialog"` bernama To Do benar-benar terlihat sebelum menyentuh isian.
- Semua locator memakai role dan nama aksesibel, bukan placeholder atau teks tampilan, supaya test tidak pecah lagi setiap tampilannya berubah. Kalau isiannya tetap tidak terlihat, test berhenti dengan pesan yang menyebut apakah panel dipakai, bukan menggantung 30 detik.
- Tidak ada `waitForTimeout` di bagian ini: setiap langkah menunggu keadaan yang ditunggu (`waitFor` pada locator yang terlihat, atau `waitForFunction` untuk urutan dan coretan).
- Seluruh badan test ada di dalam `try`, dan `finally` selalu membersihkan item yang judulnya mulai dengan "E2E " lewat `DELETE /api/todos/:id` memakai `ctx.request` (cookie-nya ikut dari context, Origin dikirim manual). Lari yang mati di tengah jalan tidak lagi meninggalkan sampah di akun staf, yang sebelumnya membuat uji keadaan kosong gagal di lari berikutnya.
- Uji keadaan kosong hanya dijalankan kalau setelah pembersihan akun stafnya memang tidak punya to do sungguhan, supaya test tidak gagal karena data orang.
- Permintaan dengan status 400 ke atas dicatat beserta metode dan URL-nya, lalu disertakan di pesan FAIL console. Pesan console browser untuk ini hanya "Failed to load resource" tanpa URL, jadi tanpa catatan itu 404 tidak bisa dilacak dari keluaran test.
- Isian ubah membuka dengan seluruh teks terseleksi, jadi mengetik akan mengganti judul, bukan menambah di akhir. Test menekan End dulu sebelum mengetik tambahan, dan ikut memeriksa seleksinya (0 sampai panjang teks). Sebelum ini test langsung mengetik " diubah" dan judulnya jadi "diubah", lalu langkah centang menunggu judul yang tidak pernah ada.
- Setiap langkah yang mengklik sesuatu diawali pemeriksaan keberadaannya dan berhenti dengan `return` kalau tidak ada. Tampilan yang berubah sekarang berakhir sebagai FAIL dengan nama langkahnya, bukan timeout 30 detik tanpa konteks.
- Centang diklik di labelnya, bukan di input-nya. Input kotak centang `sr-only`, jadi kotaknya tertutup label itu sendiri dan `check()` pada input selalu tertahan ("label intercepts pointer events", dicoba ulang sampai 30 detik). Klik di label adalah yang dilakukan pemakai, dan label itu yang membawa nama aksesibelnya, jadi tidak perlu `force: true` yang hanya mematikan pemeriksaan Playwright. Markupnya tidak diubah.
- Tombol di kepala kartu memakai `min-w-11`, bukan hanya `min-h-11`. "Atur" dengan `px-2` hanya 41 px lebarnya, dan audit E2E menangkapnya sebagai tap target di bawah 44 px. Tombol teks lain di kartu itu sudah lebih lebar dari 44 px, jadi tampilannya tidak berubah.
- Tambahan yang diuji sekarang: centang item baru harus tersembunyi (dibaca dari `stroke-dashoffset` dibanding `getTotalLength`, bukan dari ada atau tidaknya elemennya), dan baris berisi URL panjang harus tetap setinggi semula saat disorot dan difokus dengan judul minimal separuh lebar kartu.

**Keputusan perbaikan bug revisi 19:**
- "Kemunculan: -" bukan soal data. Kolom `occurrence_date` memang terisi di database, dan tidak mungkin kosong untuk task berseri karena CHECK `tasks_series_pair` menuntut `(series_id IS NULL) = (occurrence_date IS NULL)`. Yang hilang adalah dua kolom di jawaban detail: `getTask` menyusun `taskSummary` tanpa `occurrence_date` dan `series_freq`. Jadi gejalanya muncul untuk semua task berulang, bukan hanya yang dijadwalkan lewat Ubah.
- Karena itu tidak ada UPDATE data yang perlu dijalankan di produksi. SELECT pemeriksanya tetap diberikan ke Rifky, dan hasilnya seharusnya nol baris; UPDATE-nya hanya disiapkan untuk kalau hasilnya tidak nol.
- Jadwal yang ditambahkan lewat Ubah sudah menjadikan task itu kemunculan pertama di transaksi yang sama (`startSeries`, `SetTaskSeries`), dengan occurrence_date sama dengan tanggal mulai atau hari ini menurut `officeTZ`. Ticker tidak membuat kemunculan dobel karena unique (series_id, occurrence_date). Keduanya sekarang dijaga test DB, yang sebelumnya tidak ada.
- Sub task templat dikirim server sebagai bagian dari detail seri, beserta nama PIC-nya, dan server juga yang memutuskan apakah daftarnya berbeda dari sub task task ini (`template_matches`). Frontend tidak membandingkan dua daftar sendiri, supaya aturan "apa yang ikut diulang" hanya ada di satu tempat.

**Selesai kalau:** `./scripts/check.sh` lolos, migrasi 000019 dan `10-grants.sql` jalan di dev (daftar DELETE delapan tabel), Staf bisa membuat task lengkap dengan sub task dan jadwal, pembuat bisa mengubah task Requested miliknya, To Do jalan di kolom kiri dan di panel serta lembar, detail task berulang menampilkan kemunculan dan sub task templat, dan `E2E_ONLY=todo` lolos.

**Hasil:** kode selesai 2026-11-19, `./scripts/check.sh` lolos, `go vet ./...` bersih, test unit dan test DB baru terkompilasi dan vet-bersih (test DB skip di laptop karena `TEST_DATABASE_URL` kosong). Migrasi, grants, uji curl, uji browser, dan `E2E_ONLY=todo` menunggu Rifky.

## Sprint 11z: Sub task otomatis masuk templat seri

**Tujuan:** revisi 29 Manajer. Sub task yang ditambahkan lewat panel detail di task berulang tidak masuk templat, jadi tidak ikut kemunculan berikutnya. Contoh produksi: T-0018 punya 3 sub task, tetapi bagian Jadwal menulis "Tidak ada sub task di templat."

- [x] S11z-01 `syncTemplateSubtasks` di `task/series.go`, dipanggil di akhir transaksi tambah, ubah, dan hapus sub task, dan di Ubah task (jalur urutkan, ubah, dan PIC sub task yang ikut lepas). Query baru `IsLatestOccurrence`.
- [x] S11z-02 Detail membawa `series.latest_occurrence`. Bagian Jadwal: kalimat "Perubahan sub task di kemunculan terbaru otomatis dipakai untuk kemunculan berikutnya." di kemunculan terbaru seri aktif, peringatan beda templat hanya di kemunculan lama. Teks audit dan riwayat untuk pembaruan templat dari sub task.
- [x] S11z-03 Test DB `series_subtask_template_db_test.go`; `TestDetailMembawaSubTaskTemplat` disesuaikan dengan aturan baru. Tanpa E2E.

**Keputusan:**
- Kemunculan terbaru adalah kemunculan dengan `occurrence_date` paling akhir di serinya, tidak termasuk yang dihapus. Bila kemunculan terbaru dihapus, kemunculan sebelumnya kembali menjadi yang terbaru.
- Selama seri aktif, setiap tambah, ubah, hapus, dan urutkan sub task di kemunculan terbaru menulis ulang daftar sub task templat dari sub task task itu, di transaksi yang sama. Siapa pun yang boleh mengubah sub task itu (Manajer, PIC task, pembuat atau PIC sub task) ikut memperbarui templat; aturan hak sub task tidak berubah.
- Yang disalin hanya rencana: nama, PIC, deskripsi, tautan acuan, dan jarak deadline terhadap `occurrence_date`. Centang, catatan hasil, tautan hasil, dan berkas tidak. Perubahan yang hanya menyentuh hasil pengerjaan tidak menulis templat dan tidak mencatat audit.
- Kemunculan lama dan seri yang dihentikan tidak memperbarui templat, supaya membereskan task minggu lalu tidak mengubah task minggu depan.
- Field task induk (judul, deskripsi, urgency, PIC, label, deadline) tetap hanya masuk templat lewat Ubah dengan centang "Pakai perubahan ini". Ubah tanpa centang yang mengubah sub task tetap memperbarui sub task templat. Dengan centang, seluruh templat ditulis ulang sekali oleh jalur jadwal, jadi tidak tercatat dua kali.
- Audit: `series_updated` dengan aktor yang mengubah sub task, `new_value.source = "subtask"`, dan daftar judul sub task templat sebelum dan sesudah.
- Data lama tidak diisi ulang lewat migrasi. Task seperti T-0018 (kemunculan terbaru yang sub task-nya sudah beda dengan templat) ikut sinkron pada perubahan sub task berikutnya, atau segera lewat Ubah dengan centang "Pakai perubahan ini".

**Selesai kalau:** `./scripts/check.sh` lolos dan test DB revisi 29 lolos.

**Hasil:** `./scripts/check.sh` lolos. `go test ./internal/task/` lolos, test DB terlewati karena `TEST_DATABASE_URL` kosong di terminal Claude; test DB revisi 29 menunggu dijalankan Rifky. Tanpa migrasi; backend perlu build ulang.

## Sprint 12a: Halaman Timeline

**Tujuan:** revisi 30 Manajer. Satu kalender bersama untuk launching, produksi, dan kehadiran tim, plus kartu Important Notes. Semua peran membaca; hanya Manajer yang menulis.

- [x] S12a-01 Migrasi `000020_timeline`: tabel `timeline_events` dan `timeline_notes`, enum `timeline_category` dan `absence_type`, CHECK rentang dan jenis kartu, index `(start_date, end_date)`, trigger `updated_at`.
- [x] S12a-02 Paket `internal/timeline` dan rute `GET/POST /api/timeline/events`, `PUT/DELETE /api/timeline/events/:id`, `GET/POST /api/timeline/notes`, `PUT /api/timeline/notes/order`, `PUT/DELETE /api/timeline/notes/:id`. Kategori Audit Log baru "Timeline".
- [x] S12a-03 Halaman `/timeline` (menu "Timeline", semua peran, lebar penuh): grid bulan dengan roving tabindex, agenda di bawah 768 px, popover tanggal, panel Buat kartu dan Ubah, Important Notes.
- [x] S12a-04 Test DB `internal/httpapi/timeline_db_test.go` (hak, rentang, privasi, query bulan), test unit `internal/timeline/timeline_test.go` dan `frontend/tests/timeline.test.ts`, E2E `E2E_ONLY=timeline`.

**Keputusan:**
- Hak tulis hanya Manajer, diperiksa satu tempat (`timeline.Service`), jadi rutenya di grup `app` dan jawaban 403 sama untuk Staf dan Admin. Admin tidak ikut menulis walau grup `manage` memberi Admin hak yang sama dengan Manajer: Timeline adalah jadwal kerja tim, bukan pengelolaan sistem. Respons daftar membawa `can_write`, frontend hanya mengikutinya.
- Rentang paling panjang 62 hari termasuk hari pertama dan terakhir (`end_date - start_date <= 61`), dijaga di service dan di CHECK migrasi. Sampai yang sama dengan tanggal mulai disimpan sebagai satu hari (`end_date` NULL).
- Kehadiran memakai `description` sebagai kolom Keterangan, dan `title` dibiarkan NULL. CHECK `timeline_events_kind` memastikan kehadiran selalu punya orang dan alasan, sedangkan launching dan produksi selalu punya judul dan tidak punya orang.
- Pilihan Nama di form memakai `/api/people` (Staf dan Manajer aktif, sama dengan kandidat PIC), dalam `<select>` biasa karena satu kartu satu orang; `PicPicker` dirancang untuk banyak pilihan. Backend menerima akun aktif mana pun. Saat mengubah kartu lama milik akun yang sudah nonaktif, namanya tetap tampil sebagai pilihan dengan tanda "(nonaktif)", tetapi menyimpan tetap ditolak sampai diganti akun aktif.
- Hapus memakai `is_deleted`, jadi daftar DELETE di `10-grants.sql` tetap delapan tabel. Tidak ada pemulihan dari aplikasi; baris yang dihapus tetap ada untuk audit.
- Audit Log kategori "Timeline": buat, ubah, dan hapus kartu dan catatan. Urutan catatan tidak dicatat, karena isinya tidak berubah. Kartu kehadiran dicatat dengan nama orangnya saja, tanpa alasan dan keterangan, karena Audit Log juga dibaca Admin.
- Endpoint daftar menerima satu bulan (`month=YYYY-MM`) dan mengembalikan semua event yang beririsan dengan bulan itu. Tanggal bulan sebelah di grid tampil redup tanpa isi, dan mengkliknya membawa ke bulan itu; event rentang yang melintasi bulan tampil di setiap hari dalam rentangnya di kedua bulan.
- Grid mulai 768 px, dua kolom (kalender dan kolom kanan) mulai 1024 px. Di 768 sampai 1023 px kolom kanan pindah ke bawah kalender, karena tujuh kolom di samping kolom 20rem menyisakan sel sekitar 55 px. Di bawah 768 px grid diganti agenda lewat `matchMedia`, bukan CSS saja, supaya hanya satu daftar tanggal yang bisa difokus ada di DOM.
- Penanda kategori berupa bentuk dan selalu berdampingan dengan teks: Launching kotak penuh `ink`, Produksi kotak garis `ink`, Kehadiran bulatan inisial `tag` dengan teks `tag-ink`. Tidak ada warna baru. Kontras: `ink` di atas `surface` dan `canvas` di atas 14:1 di kedua mode (non-teks, ambang 3:1); inisial `tag-ink` di atas `tag` 13,31:1. Inisial 10 px bersifat dekoratif (`aria-hidden`), nama lengkapnya tertulis di sebelahnya.
- Filter kategori hanya menyaring grid dan agenda. Popover tanggal selalu menampilkan semua entri hari itu. Status toggle ditunjukkan `aria-pressed`, garis `mark`, dan ikon centang, bukan warna saja.
- Hari ini: angka dalam lingkaran `ink` dengan teks `surface`. Sabtu dan Minggu berlatar `canvas`, hari kerja `surface`, sama dengan sorotan hari ini di Aktivitas (Sprint 11s). Tanggal bulan lain memakai `ink-muted` (5,73:1 di atas `canvas` mode terang).
- Popover tanggal mengikuti pola `PersonPopover` (Sprint 11l): `popover="manual"`, menempel di sel mulai 640 px dan lembar dari bawah di bawahnya, Escape dan klik luar menutup lalu fokus kembali ke sel. Escape bertahap: konfirmasi hapus ditutup dulu. Hapus memakai konfirmasi di dalam popover, bukan dialog kedua.
- Membuka Ubah atau Tambah dari popover menutup popover dulu, lalu panel form terbuka. Setelah Batal atau simpan, fokus kembali ke tanggal asalnya, atau ke tombol Buat kartu bila dibuka dari sana.
- Important Notes sengaja dibedakan dari kartu krem To Do (permintaan Rifky 2026-11-19), supaya terbaca sebagai pengumuman penting: garis `urgent` tebal (kiri 8 px), latar `danger-bg`, ikon peringatan dan klip kertas `urgent`, judul `danger`, dan chip "Penting" `on-urgent` di atas `urgent`. Tidak ada warna baru. Kontras terang dan gelap: teks `ink` 14,08:1 dan 14,55:1, `ink-muted` 5,30:1 dan 5,42:1, judul `danger` 6,81:1 dan 8,67:1, chip 6,93:1 dan 7,83:1, garis `urgent` 5,72:1 dan 7,99:1, pemisah `line` 3,19:1 dan 3,54:1. Urutan lewat tombol naik dan turun, tanpa seret. Urutkan mengirim seluruh daftar dan ditolak bila daftarnya sudah berubah di tempat lain. Paling banyak 50 catatan.
- Bulan yang dibuka tersimpan di query `?bulan=YYYY-MM` (kosong untuk bulan ini), supaya tautan dan tombol Kembali membuka bulan yang sama.

**Diganti revisi 38 (Sprint 12e):** alasan dan keterangan kehadiran terbuka untuk semua pengguna yang login; penyamaran "Tidak hadir" di atas tidak berlaku lagi.

**Selesai kalau:** `./scripts/check.sh` lolos, migrasi 000020 jalan di dev, test DB Timeline lolos, dan `E2E_ONLY=timeline` lolos di 360 dan 1280 px.

**Hasil:** kode selesai 2026-11-19, `./scripts/check.sh` lolos, test unit backend dan frontend lolos. Migrasi 000020, restart backend, test DB, dan E2E menunggu Rifky.

## Sprint 12b: Hapus task berulang dan hak PIC atas sub task

**Tujuan:** revisi 31 dan 32 Manajer, dari dua kejadian di produksi: task "TEslagi" dihapus lalu T-0026 tetap dibuat Sistem, dan PIC task T-0029 tidak bisa mengubah sub task kemunculannya.

- [x] S12b-01 Revisi 31 backend: `DELETE /api/tasks/:id?stop_series=true` menghapus task dan menghentikan seri dalam satu transaksi (`Service.DeleteAndStopSeries`), audit `task_deleted` dan `series_stopped`. `can.stop_series` di detail. Pengaman ticker untuk seri yang ditinggalkan. Tanpa migrasi.
- [x] S12b-02 Revisi 31 frontend: dialog hapus task dari seri aktif dengan dua pilihan tanpa bawaan, tanggal kemunculan berikutnya, dan baris "Pengulangan: Aktif atau Dihentikan" di detail.
- [x] S12b-03 Revisi 32: Staf PIC task boleh mengubah dan menghapus semua sub task di task-nya (`editableSubtask`, `can_edit`).
- [x] S12b-04 Test DB `internal/task/series_delete_db_test.go` dan `internal/httpapi/task_delete_series_db_test.go`.

**Keputusan:**
- Revisi 31: dialog hapus task yang termasuk seri aktif wajib memilih "Hapus kemunculan ini saja (pengulangan tetap berjalan)" atau "Hapus dan hentikan pengulangan". Tidak ada pilihan bawaan dan tombol Hapus nonaktif sampai salah satu dipilih. Pilihan pertama menyebut tanggal kemunculan berikutnya dari server. Seri yang sudah berhenti atau task biasa memakai dialog lama.
- Pilihan kedua hanya untuk yang `can.stop_series` (Manajer, sama dengan hak "Hentikan pengulangan" revisi 19). Yang lain melihat pilihan pertama saja dengan teks "Untuk menghentikan pengulangan, minta Manajer.". Server menolak `stop_series=true` dari selain Manajer dengan 403 sebelum apa pun berubah, juga bila pengirimnya pembuat task yang boleh menghapus.
- `stop_series=true` untuk task yang tidak berulang ditolak 400 dan task tidak ikut terhapus, karena hapus dan henti seri satu transaksi.
- Perbaikan bug produksi T-0026 (2026-11-20): rujukan seri dibaca sebelum soft delete, karena `GetTaskSeriesRef` memfilter `NOT is_deleted` dan sebelumnya menghasilkan ErrNoRows (500). Handler task kini memetakan ErrNoRows yang lolos menjadi 404 "Task tidak ditemukan atau sudah dihapus. Muat ulang halaman." dan mencatat WARN. Admin tetap tidak boleh menghapus task (FR-TASK-19), jadi jawabannya 403.
- Pengaman ticker: seri aktif yang tidak punya kemunculan tersisa (semua `is_deleted`) dan kemunculan terakhirnya (termasuk yang dihapus) lebih dari 7 hari sebelum hari ini tidak dibuatkan kemunculan baru, lalu dicatat `WARN` di log backend. Seri tidak dihentikan otomatis, karena menghentikan seri hak Manajer dan bisa saja dihidupkan lagi lewat Ubah. Pengaman ini tidak mencegah kasus "TEslagi" (dihapus kemarin, muncul hari ini); itu tugas pilihan di dialog hapus.
- Detail task berulang menampilkan baris "Pengulangan: Aktif" atau "Dihentikan".
- Revisi 32: Staf PIC task boleh mengubah dan menghapus semua sub task di task-nya, termasuk sub task templat yang tercatat dibuat pembuat seri dan dipegang orang lain, selama task Requested atau On Progress. Hak pembuat, PIC sub task, dan Manajer tidak berubah. Perubahan PIC task di kemunculan terbaru tetap memperbarui templat seri (revisi 29). Menambah sub task sudah boleh untuk PIC task sejak revisi 6.
- Mengurutkan sub task: aplikasi belum punya pengurutan sub task di luar form Ubah task (yang hanya untuk Manajer dan pembuat task selama Requested), jadi revisi ini tidak menambah pengurutan. Masuk "Menunggu jawaban Manajer".
- Centang tidak berubah dan sesuai permintaan: PIC task tidak boleh mencentang sub task yang dipegang orang lain. Satu-satunya pengecualian yang sudah ada sejak revisi 7: sub task tanpa PIC boleh dicentang PIC task mana pun.
- Teks "Kamu hanya bisa melihat task ini" di T-0029: kode tidak membedakan kemunculan buatan Sistem. Hak dihitung dari `task_assignees`, bukan `requested_by` atau `created_by`, dan Staf yang tercatat PIC task selalu mendapat `edit_labels` sehingga teks itu tidak mungkin tampil untuknya. Test `TestTaskPICManagesTickerSubtasks` membuktikan PIC task di kemunculan ticker mendapat On Progress di `move_to` dan tombol Tambah sub task. Jadi saat teks itu tampil, akun yang membuka bukan PIC kemunculan itu di database (misalnya PIC templat lain, atau akun Nala nonaktif saat kemunculan dibuat sehingga dilewati `GetEligiblePICs`). Perlu dicek di data produksi.

**Selesai kalau:** `./scripts/check.sh` lolos dan test DB `-run 'DeleteAndStopSeries|TickerSkipsAbandoned|TaskPICManagesTicker|DeleteStopSeriesHTTP'` lolos.

**Hasil:** kode selesai 2026-11-20, `./scripts/check.sh` lolos. Test DB menunggu Rifky (`TEST_DATABASE_URL`).

## Sprint 12c: Rantai sambung jam di Aktivitas Harian

**Tujuan:** revisi 33 Manajer. Pekerjaan yang berlangsung lebih dari satu jam tidak perlu diisi ulang di setiap jam: jam berikutnya cukup disambung ke jam sebelumnya lewat tombol rantai.

- [x] S12c-01 Migrasi 000021: kolom `continues_previous boolean NOT NULL DEFAULT false` di `daily_activities` dan CHECK `daily_activities_chain_empty` (jam tersambung tidak boleh punya Aktivitas atau Output sendiri). Tanpa tabel baru.
- [x] S12c-02 Backend `internal/activity/chain.go`: `Joinable`, `resolveChains`, `Service.Link`, dan `Conflict` (409). `PUT /api/activities/link` dengan `{user_id, date, slot, linked}`, jawabannya isian sehari penuh. Simpan dan unggah ke jam tersambung ditolak 409.
- [x] S12c-03 Sel API membawa `linked`, `head`, `span`, `joinable`, dan `link_hint`. Hitungan jam terisi, halaman `/aktivitas` (rowspan), ekspor Excel (sel digabung ke bawah), dan nama berkas di ekspor Drive (`09.00-11.00 - bukti.png`) memakai rantai yang sama.
- [x] S12c-04 Frontend: `ActivityChainButton` di kepala kolom jam (layar lebar) dan di kartu jam (di bawah 768 px), jam tersambung tampil readonly dengan label "Lanjutan 09.00-10.00".
- [x] S12c-05 Test DB `internal/activity/chain_db_test.go`, test unit `TestJoinable`, test `TestRantaiBolehMelintasiIstirahat`, dan bagian `E2E_ONLY=chain` di `scripts/e2e/uji13.mjs` (360 dan 1280 px).
- [x] S12c-08 Revisi 33b poles akhir (tampilan saja): keadaan tombol rantai dibedakan warna `accent` dan garis ikon lebih tebal (2,75 lawan 2) ditambah `title`; status simpan pindah ke baris judul jam lewat komponen baru `ActivitySlotStatus`, dipakai kepala kolom tabel dan baris jam kartu jam, jadi Tambah berkas merapat ke bawah Output. E2E `chain` memeriksa warna tombol tersambung berbeda dari yang belum, dan jarak Output ke Tambah berkas maksimal 12 px.
- [x] S12c-07 Revisi 33b lanjutan (tampilan saja): jam lanjutan memakai isian `readonly` yang sama dengan jam kepalanya, jadi tinggi, lebar, dan patahan teksnya sama; label "Lanjutan <jam>" pindah ke kepala kolom; placeholder dimatikan di isian `readonly`; `blur` tidak mengirim simpan untuk jam tersambung. E2E `chain` ikut memeriksa tinggi kotak lanjutan sama dengan kepalanya (selisih maksimal 1 px) dan lebarnya sama dengan kolom lain.
- [x] S12c-06 Revisi 33b (tampilan saja, aturan dan API tidak berubah): tombol rantai pindah ke celah antara dua isian Aktivitas sebagai ikon kecil tanpa bingkai dengan tap target 44 px, hanya dirender saat bisa dipakai atau sudah tersambung, jam tersambung menyatu dengan jam kepalanya, status simpan punya baris tetap di `ActivityBox` dan `ActivityCellFields`, dan "Tambah berkas" di dalam sel tabel tanpa bingkai. Locator `E2E_ONLY=chain` ikut disesuaikan.

**Keputusan:**
- Data: satu penanda di baris jam, bukan salinan isi. Isi jam tersambung diturunkan dari jam kepalanya saat dibaca (`resolveChains`), jadi edit di jam kepala langsung tampil di semua jam yang tersambung. Model data yang ada cocok: satu baris per orang, tanggal, dan slot sudah ada, dan berkas sudah menempel per slot.
- Melepas rantai hanya mengubah penanda menjadi false. Baris tidak dihapus, jadi hak DELETE di `daily_activities` tetap tidak dibutuhkan dan `deploy/sql/10-grants.sql` tidak berubah (tidak ada tabel baru).
- Rantai boleh lebih dari dua jam. Melepas jam di tengah ikut melepas jam setelahnya yang tersambung ke jam itu; jam kepala tidak berubah. Jam yang dilepas kembali kosong dan bisa diisi.
- Ubahan revisi 33 (2026-11-20): rantai boleh melintasi jam istirahat. Jam bersebelahan berarti slot berurutan di `activity.Slots`, jadi 11.00-12.00 dan 13.00-14.00 juga bisa disambung dan tombol rantainya muncul di antara keduanya. Satu-satunya jam tanpa tombol adalah slot pertama (ditolak 400 lewat API). Span tetap dari jam mulai kepala sampai jam selesai ekor (mis. 11.00-14.00), sedangkan hitungan jam terisi tetap per slot (rantai 11.00-12.00 dan 13.00-14.00 dihitung 2 jam). Excel dan nama berkas Drive mengikuti aturan yang sama. Menggantikan keputusan awal revisi 33 yang memutus rantai di jam istirahat.
- Rantai hanya bisa dipasang bila jam itu kosong (Aktivitas, Output, dan berkas). Kalau tidak, server menolak 409 "Kosongkan jam ini dulu untuk menyambung." dan tombolnya `aria-disabled` dengan tooltip yang sama.
- Tambahan dari kami: rantai juga ditolak 409 bila jam sebelumnya (atau kepala rantainya) kosong, "Isi jam sebelumnya dulu untuk menyambung.". Tanpa ini jam kosong bisa disambung ke jam kosong dan terhitung terisi. Masuk "Menunggu jawaban Manajer".
- Jam tersambung tidak bisa diisi atau diberi berkas sendiri: simpan dan unggah ke jam itu ditolak 409 "Jam ini tersambung ke jam sebelumnya. Lepas sambungan dulu untuk mengisinya.". Simpan, unggah, dan sambung untuk orang yang sama berjalan bergiliran karena ketiganya mengunci baris akun pemilik (`GetUserForUpdate`); CHECK migrasi menjadi pengaman terakhir.
- Hak sama dengan mengisi jam (`CanEdit`): Staf hanya jamnya sendiri hari ini, Manajer siapa pun di tanggal berapa pun, Admin tidak. Perubahan Manajer pada jam orang lain dicatat sebagai `activity_updated` dengan `linked` di before dan after, jadi tidak ada aksi audit baru.
- Jam tersambung dihitung terisi selama jam kepalanya berisi. Hitungan frontend (`filledHours`) kini juga menghitung jam yang hanya berisi berkas, sama dengan backend.
- Halaman `/aktivitas`: rantai tampil sebagai satu sel dengan `rowspan`, dan dialognya berjudul rentangnya (mis. 09.00-11.00). Tombol rantai hanya ada di kartu papan task, tempat jam diisi; halaman `/aktivitas` tidak menambah tombol rantai. Backend tetap mengizinkan Manajer lewat API.
- Ekspor Excel: kolom Activity dan Output orang itu digabung ke bawah sepanjang rantai (`mergeCell`), isinya ditulis sekali di jam kepala. Kolom Time tetap per jam, karena kolom itu dipakai bersama semua orang.
- Ekspor Drive: berkas di jam kepala rantai diberi nama dengan rentangnya. Berkas yang sudah terunggah tidak diganti namanya (ekspor satu arah).
- Tombol rantai di layar lebar duduk di garis antara dua kolom jam, 44x44 px. Di bawah 768 px tombolnya ada di baris jam di kartu jam itu. `aria-pressed` menandai keadaan tersambung. Tanpa animasi baru. Tempat dan gaya tombolnya diubah revisi 33b di bawah.

**Keputusan tampilan revisi 33b (2026-11-20):**
- Tombol rantai pindah dari baris judul ke celah antara dua isian Aktivitas yang disambungnya, di tengah tinggi isian itu. Di baris judul tombol itu menempel ke label jam berikutnya, jadi tidak terbaca sebagai penyambung dua isian di bawahnya.
- Semua kolom jam memakai `pl-6 pr-4`, jadi celah tempat tombol itu selalu ada dan kolom tanpa tombol tidak bergeser. Tombolnya absolute di celah itu dan tidak ikut mengubah lebar isian.
- Ikon kecil (`size-4`) tanpa bingkai dan tanpa latar, warna `ink-muted`, dan `accent` saat tersambung. Kotak tombolnya sendiri tetap 44x44 px (`size-11`), jadi tap target tidak berubah meski ikonnya kecil. Keadaan tersambung tidak hanya dibedakan warna: ada `aria-pressed`, label "Lanjutan <jam>", dan latar kotak yang menyatu.
- Tombol hanya dirender kalau jamnya sudah tersambung atau benar-benar bisa disambung sekarang (`joinable` dan `link_hint` kosong). Tombol `aria-disabled` dengan tooltip alasan dihapus: tujuh tombol mati di satu baris lebih ramai daripada manfaat alasannya, dan alasan yang sama tetap muncul sebagai galat 409 bila dicoba lewat API. Aturan dan API tidak berubah, hanya yang dirender.
- `aria-label` kini menyebut jamnya ("Sambungkan jam 14.00-15.00 ke jam sebelumnya", "Lepas sambungan jam 14.00-15.00"). Di satu kartu ada sampai tujuh tombol ini, dan label yang sama persis membuat daftar tombol pembaca layar tidak bisa dibedakan. E2E `chain` ikut memakai nama itu, bukan posisi DOM-nya.
- Jam tersambung tidak lagi kotak putus-putus terpisah: isiannya sama persis dengan isian jam biasa, hanya `readonly`, dengan tepi kiri lurus tanpa garis dan latar `canvas`. Karena komponen dan lebarnya sama, tinggi dan titik patah teksnya otomatis sama dengan jam kepalanya, tanpa aturan tinggi tambahan. Kotaknya tidak lagi ditarik ke kiri menutupi celah: lebarnya sama dengan kolom lain, dan efek menyatu cukup dari tepi lurus plus latar tipis itu.
- Label "Lanjutan <jam>" pindah ke kepala kolom jamnya. Di dalam sel ia memakan satu baris, jadi jam lanjutan selalu lebih tinggi daripada jam kepalanya. Sebagai `th scope="col"` ia juga ikut dibacakan untuk kedua sel di bawahnya, jadi tidak ada informasi yang hilang. Di kartu jam (di bawah 768 px) label itu tetap di baris jam kartunya, karena di sana isian bertumpuk dan tidak ada tinggi yang perlu disejajarkan.
- Placeholder dimatikan di isian `readonly`, jadi jam kepala yang Output-nya kosong tidak tampak berisi di jam lanjutannya, dan tidak ada tanda "-" yang perlu ditambahkan. Teks "-" di `ActivityCellFields` mode `readonly` dibiarkan: komponen itu juga dipakai halaman `/aktivitas` untuk baris orang lain, yang memang tabel teks, bukan kotak isian.
- Simpan saat kehilangan fokus dilewati untuk jam tersambung (`blur` berhenti di awal). Isiannya `readonly` jadi tidak bisa berubah, dan server menolaknya 409, jadi permintaannya tidak perlu dikirim sama sekali.
- Keadaan tombol rantai tidak dibedakan warna saja: yang tersambung memakai `accent` dengan garis ikon lebih tebal (2,75 lawan 2), ditambah `aria-pressed`, `title` ("Lepas sambungan" atau "Sambungkan ke jam sebelumnya"), label "Lanjutan <jam>" di kepala kolom, dan latar kotak isiannya. Kontras ikon `accent` di latar kartu: 17,06:1 terang dan 14,97:1 gelap (ambang non-teks 3:1), 15,23:1 dan 15,54:1 saat selnya disorot drag dan latarnya jadi canvas; `accent-hover` 13,02:1 dan 16,86:1; ikon yang belum tersambung (`ink-muted`) 6,42:1 dan 5,97:1.
- Status simpan pindah ke baris judul jam, satu tempat dengan label "Lanjutan", lewat komponen `ActivitySlotStatus` yang dipakai kepala kolom tabel dan baris jam kartu jam. Satu baris kecil (tinggi `1.125rem`, sama dengan `leading` teks 13 px) dicadangkan di semua kolom, jadi status yang datang dan hilang tidak mengubah tinggi baris judul, dan di bawah Output tidak ada celah kosong yang menunggunya. Tambah berkas kini 4 px di bawah Output di tabel dan 8 px (gap grid) di kartu jam.
- E2E `chain` gagal satu pemeriksaan di 1280 px setelah kotak lanjutan menjadi textarea: `getByText` tidak bisa melihat isi textarea, karena nilainya properti DOM, bukan teks anak. Isinya memang dirender, hanya tidak terlihat oleh locator lama, dan di 360 px pemeriksaan itu tetap lolos karena di kartu jam isinya masih teks biasa. Sekarang test membaca `inputValue()` kalau ada isian jam itu, dan teks kartunya kalau tidak, jadi yang diperiksa tetap isi jam kepala di kotak lanjutan.
- ActivityBox tidak lagi punya baris status sendiri: kodenya ikut dibuang setelah pindah, karena komponen itu hanya dipakai kartu papan task dan tidak ada pemanggil lain yang memerlukannya.
- Id baris itu yang dipakai `aria-describedby` kedua isian, jadi hasil simpan tetap terbaca saat fokus ada di Aktivitas. `aria-live="polite"` menempel di baris yang permanen itu, dan `role="alert"` ditambahkan hanya saat galat.
- Galat: teksnya dipotong elipsis dengan `title`, tetapi isi penuhnya tetap ada di DOM, jadi pembaca layar ikut membacakannya lewat `role="alert"`. Teks itu sendiri yang menjadi tombol coba simpan lagi (dengan keterangan `sr-only`), dan tinggi kliknya 44 px dari `py-[13px]` yang ditarik balik `-my-[13px]`, jadi baris judul tidak ikut tinggi. Kontras `danger` di surface 8,25:1 dan 9,56:1.
- `ActivityCellFields` dapat prop `hideStatus`, dipakai hanya kartu papan task yang sudah menampilkan status di baris jam. Halaman `/aktivitas` tidak mengirimnya, jadi baris status di sana tetap seperti sebelumnya.
- Tombol lepas sambungan duduk di celah yang sama dengan tombol sambung, di luar kotak isian, jadi tidak ada ikon yang berdempetan dengan teks. Kotak lanjutan yang tidak lagi melebar ke celah itu yang membuat ini mungkin.
- Status simpan ("Menyimpan", "Tersimpan", galat) punya baris tetap setinggi satu baris teks kecil di `ActivityBox` dan `ActivityCellFields`, selalu dirender. Sebelumnya baris itu muncul dan hilang, jadi Tambah berkas terdorong turun dan tinggi kolom melompat setiap simpan. Karena baris itu permanen, `aria-live="polite"` pindah dari span status ke baris itu: live region yang lahir bersama isinya sering tidak diumumkan. Baris galat masih bisa lebih tinggi karena membawa tombol "Coba lagi", dan itu dibiarkan: galat jarang dan tombolnya memang perlu.
- Kontras dihitung, dua mode: ikon rantai mati (ink-muted) 6,42:1 terang dan 5,97:1 gelap di atas surface, 5,73:1 dan 6,2:1 di atas canvas saat sel disorot drag (ambang non-teks 3:1); ikon tersambung (accent) 15,23:1 dan 15,54:1; status simpan 13px (ink-muted) 6,42:1 dan 5,97:1; teks "Lanjutan" di latar canvas 5,73:1 dan 6,2:1; isi jam tersambung 15,23:1 dan 16,66:1. Semua lolos AA.
- Tombol "Tambah berkas" di dalam sel tabel (`compact`) kehilangan bingkainya dan memakai warna `ink-muted`; di luar sel (kartu jam) tetap berbingkai. Delapan tombol berbingkai di satu baris membuat tabelnya terbaca seperti deretan kotak, padahal isian yang utama.
- Test `TestLampiranManajerDanHapus` gagal sebelum revisi ini: test menghitung aksi `activity_updated`, sedangkan unggah mencatat `activity_file_added`. Diperbaiki 2026-11-20 bersama gerbang `scripts/pra-deploy.sh`: kodenya yang benar (aksi khusus berkas terdaftar di kategori Aktivitas Audit Log sejak Sprint 11v), jadi test kini menghitung `activity_file_added` dan `activity_file_deleted` masing-masing dan memastikan unggah tidak tercatat sebagai `activity_updated`.

**Selesai kalau:** `./scripts/check.sh` lolos, migrasi 000021 jalan di dev dan test DB, test DB `-run 'Rantai|Joinable'` lolos tanpa SKIP, dan `E2E_ONLY=chain` lolos di 360 dan 1280 px.

**Hasil:** kode selesai 2026-11-20, `./scripts/check.sh` lolos, migrasi 000021 jalan di dev dan test DB, test DB rantai lolos. E2E menunggu backend di-restart.

## Sprint 12d: Hak centang sub task dan Output beberapa tautan

**Tujuan:** revisi 34 dan 35 Manajer.

- [x] S12d-01 Revisi 34 backend: `canToggle` baru (PIC sub task itu, PIC task mana pun, pembuat task `requested_by`, Manajer), `CheckableSubtasks` jadi fungsi murni tanpa query, query `ListSubtasksWithOtherManagerPIC` dan `CountSubtaskAssignees` dibuang. Pesan 403 "Sub task ini hanya bisa dicentang PIC sub task, PIC task, pembuat task, atau Manajer.".
- [x] S12d-02 Revisi 34, jalur revisi: `Move` menerima `subtask_ids` opsional dari PIC task dan Manajer saat Ready ke On Progress (sub task tercentang mana pun, catatan tetap wajib). Detail mengirim `revision_subtasks` dan `revision_note_required`; dialog revisi di detail memakai `SubtaskRevisionFields` dengan `noteRequired`.
- [x] S12d-03 Revisi 35: `outputParts` di `utils/outputLink.ts` dengan test unit `tests/outputParts.test.ts`, komponen `OutputLinks.vue`, dipasang di sel tabel /aktivitas, dialog detail jam, kartu Aktivitas di papan (tampilan bertumpuk), dan Output jam lanjutan.
- [x] S12d-04 Test DB `TestCentangSubTaskPerPeran` dan `TestRevisiPICTaskMembukaSubTask`, unit `TestCanToggle` ditulis ulang; test lama yang menguji aturan yang diganti disesuaikan.

**Keputusan:**
- Revisi 34: yang boleh mencentang dan membatalkan centang sub task mana pun di task itu adalah PIC sub task itu, semua PIC task, pembuat task (`requested_by`), dan Manajer, hanya saat task On Progress. Admin selalu ditolak, juga bila tercatat sebagai peminta task hasil impor Excel. Staf yang tidak terlibat ditolak 403.
- Aturan lama yang diganti: (1) Sprint 11g, pembuat sub task boleh mencentang: dihapus atas jawaban Rifky 2026-11-23, karena pembuat sub task hampir selalu PIC task atau Manajer; orang yang dicopot dari task tidak lagi bisa mencentang. (2) Revisi 4, Manajer tidak boleh mencentang sub task yang dipegang Manajer lain: dihapus, Manajer boleh semua. (3) Sprint 11e, PIC task hanya boleh mencentang sub task tanpa PIC: dihapus, PIC task boleh semua. (4) Revisi 32 mencatat "PIC task tidak mencentang sub task milik orang lain": diganti revisi ini.
- "Dicentang <nama>, <waktu>" dan audit `subtask_checked` memakai pelaku (`done_by` diisi `actor.ID`), bukan PIC sub task. Ini sudah begitu sejak awal; revisi ini menambah testnya.
- Berlaku sama untuk kemunculan seri: aturan dihitung dari `task_assignees` dan `requested_by` kemunculan itu.
- Perluasan revisi 22 (jawaban Rifky 2026-11-23): PIC task dan Manajer boleh memilih sub task tercentang mana pun untuk dibuka lagi saat mengembalikan Ready ke On Progress. Pilihannya opsional dan tidak ada yang terpilih secara bawaan, supaya pekerjaan orang lain tidak terbuka tanpa dipilih sendiri; catatan revisi tetap wajib. Audit memakai `note` seperti revisi biasa ditambah `reopened_subtasks`. Jalur PIC sub task tidak berubah (wajib memilih, sub task miliknya, alasan opsional). Pembuat task Staf tetap tidak bisa memindahkan Ready ke On Progress, karena itu hak pindah status, bukan hak centang.
- Pilihan sub task untuk PIC task dan Manajer hanya dikirim di detail task. Papan dan Arsip tidak mengirimnya, karena butuh semua sub task setiap kartu; di sana dialog Pindahkan tetap revisi biasa dengan catatan. `revision_note_required` dikirim di detail, papan, dan Arsip supaya frontend tidak menebak jalurnya.
- Revisi 35: semua URL http dan https di Output dikenali, dipisah spasi, baris baru, atau koma. Tanda baca di ujung (titik, koma, titik koma, titik dua, tanda seru, tanda tanya) dibuang; kurung tutup dan kurung siku hanya dibuang bila tidak punya pasangan di dalam URL, supaya tautan seperti `.../Foo_(bar)` tetap utuh. Label mengikuti aturan lama (File Drive, Folder Drive, Google Docs, Spreadsheet, Slide, atau nama host); label yang sama diberi nomor mulai dari yang kedua ("File Drive 2"). Potongan teks yang isinya hanya pemisah dibuang, teks lain tetap di urutan aslinya.
- Tombol tautan: tab baru dengan `rel="noopener noreferrer"`, `title` berisi URL lengkap, `aria-label` "<label>, buka di tab baru", tinggi `min-h-11` seperti chip File Drive sebelumnya, focus ring `mark`. Di sel tabel tombolnya `z-10` di atas tombol selebar sel. Di dialog detail setiap tautan punya barisnya sendiri dengan URL lengkap di bawahnya, karena `title` tidak terlihat di layar sentuh.
- Dialog detail jam tidak boleh digulir ke samping: teks Output, URL di bawah tombol, dan teks Aktivitas memakai `overflow-wrap: anywhere`. Thumbnail Drive diambil dari file Drive pertama di Output.
- Output jam lanjutan (rantai revisi 33) di tabel kartu papan yang berisi tautan tampil sebagai kotak hanya baca berisi tombol, bukan textarea readonly. Tanpa tautan tetap textarea readonly yang sama dengan jam kepalanya. Akibatnya tinggi kotak lanjutan yang berisi tautan bisa berbeda sedikit dari textarea jam kepalanya.
- Isi Output yang tersimpan tidak diubah. Ekspor Excel dan salinan Drive tetap memakai teks aslinya.
- Tidak ada migrasi dan tidak ada tabel baru, jadi grants tidak perlu dijalankan ulang.

**Revisi 36 (tempel dan seret berkas di lampiran task dan sub task):**

- [x] S12d-05 `utils/pasteFiles.ts` (penamaan tempelan, dengan test unit `tests/pasteFiles.test.ts`), composable `useUploadQueue` (antrean yang diangkat dari `AttachmentSection`, ditambah coba lagi), komponen `UploadQueueList` dan `FileDropTarget`. Dipasang di Arahan dan Hasil sub task serta Lampiran brief dan Deliverable task. E2E `E2E_ONLY=paste`.

**Keputusan revisi 36:**
- Bagian tujuan ditentukan dari tempat event terjadi, bukan dari logika terpisah: setiap bagian lampiran (Arahan sub task, Hasil sub task, Lampiran brief, Deliverable) dibungkus `FileDropTarget` yang menangkap `paste` dan `drop` dari elemen mana pun di dalamnya, termasuk isian Catatan hasil, Tautan hasil, dan tautan deliverable, lalu menyerahkan berkasnya ke komponen lampiran bagian itu. `stopPropagation` menjaga bagian luar tidak ikut menerima berkas yang sama.
- Tempelan tanpa berkas dibiarkan, jadi Ctrl+V berisi teks tetap menempel teks ke isian yang difokus.
- Gambar dari clipboard yang tidak bernama atau bernama generik (`image.png`) diberi nama `tempel-YYYYMMDD-HHMMSS.<ext>` menurut jam perangkat, dengan `-2`, `-3` untuk beberapa gambar sekaligus; ekstensi mengikuti jenis gambar (png, jpg, webp, gif) karena backend menolak isi yang tidak cocok dengan ekstensi. Berkas bernama yang disalin dari penjelajah berkas tetap memakai namanya. Form Buat task memakai fungsi yang sama, jadi tempelannya juga bernama `tempel-...` (sebelumnya `screenshot-...`). Aktivitas Harian tetap memakai nama per jam (`tangkapan-09.00-1.png`, keputusan Sprint 11v).
- Batas ukuran, tipe berkas, dan hak akses sama dengan tombol Tambah berkas di bagian itu, karena berkas tempelan dan seretan masuk ke antrean yang sama (`precheck` 50 MB dan daftar tipe lampiran task, `can_attach` untuk sub task, `can.edit` untuk brief, `can.edit_deliverable` untuk deliverable). Backend tidak berubah.
- Antrean bersama dipakai lampiran task dan sub task: satu berkas per waktu, progres per berkas, beberapa berkas sekaligus, galat per berkas. Galat jaringan dan penolakan server (tidak berhak, terlalu besar menurut server) punya tombol "Coba lagi"; penolakan sebelum unggah (tipe atau ukuran) hanya "Tutup", karena mengulang tidak mengubah hasilnya. Hasil unggah dan galat dibacakan lewat `aria-live`.
- Lampiran sub task sekarang menerima beberapa berkas sekaligus lewat tombolnya juga (`multiple`), dengan progres yang sama. Tombolnya tetap aktif selama mengunggah, berkas tambahan masuk antrean.
- Petunjuk "Atau tempel (Ctrl+V) atau seret berkas ke sini." hanya tampil di perangkat dengan mouse (`pointer-fine`), karena Ctrl+V tidak ada di layar sentuh. Tombol Tambah berkas selalu ada. Sorotan saat berkas diseret di atas bagian memakai `bg-canvas` dan garis `mark`, sama dengan Aktivitas Harian, tanpa animasi baru.
- Dropzone lampiran task di dalam bagian sudah menangani drop sendiri; `FileDropTarget` melewati event yang sudah ditangani (`defaultPrevented`) supaya berkas tidak terunggah dua kali.

**Selesai kalau:** `scripts/pra-deploy.sh` LOLOS dan E2E revisi 34, 35, dan 36 lolos di 360 dan 1280 px.

## Sprint 12e: Aktivitas mundur 2 hari kerja dan akses tambahan per pengguna

**Tujuan:** revisi 37 dan 38 Manajer.

- [x] S12e-01 Revisi 37 backend: `activity.EditableDays` (satu-satunya hitungan rentang), `CanEdit` memakainya untuk isian, lampiran, dan rantai jam; `GET /api/activities/me?date=` mengirim `today` dan `editable_dates`; pesan 403 menyebut batasnya. Tanpa migrasi.
- [x] S12e-02 Revisi 37 frontend: kartu Aktivitas di papan dengan tombol Sebelumnya dan Berikutnya, judul per tanggal, dan keterangan saat mengisi tanggal lalu.
- [x] S12e-03 Revisi 37 ekspor Drive: `ExportMonths` menulis ulang Excel setiap bulan yang masih bisa menerima isian.
- [x] S12e-04 Revisi 38: migrasi `000022_user_access`, akses dibaca bersama data sesi, Timeline dan Important Notes memeriksa akses masing-masing, alasan Kehadiran tidak lagi disamarkan, drawer Ubah pengguna dan form Tambah pengguna punya bagian "Akses tambahan", Audit Log menyebut perubahan akses.
- [x] S12e-05 Test: unit `TestEditableDays`, `TestExportMonths`, `TestCanEdit`; DB `TestStafMengisiDuaHariKerjaKeBelakang`, `TestEksporMengambilIsianTerlambat`, `TestAksesTambahanPerPengguna`; E2E `E2E_ONLY=aktivitasmundur` dan `E2E_ONLY=akses`.

**Keputusan revisi 37:**
- Staf boleh mengisi dan mengubah aktivitas, lampiran, dan rantai jam miliknya untuk hari ini dan 2 hari kerja Senin sampai Jumat sebelumnya, menurut zona Asia/Jakarta. Senin bisa mengisi Jumat dan Kamis. Hari ini selalu boleh, juga bila hari ini Sabtu atau Minggu; Sabtu dan Minggu sebelumnya tidak ikut dihitung dan tidak bisa diisi mundur. Tanggal masa depan ditolak. Manajer semua tanggal, Admin hanya membaca.
- Hitungan rentangnya satu fungsi di backend (`activity.EditableDays`), dikirim lewat `editable_dates` di `GET /api/activities/me`. Frontend hanya mengikuti daftar itu untuk tombol Sebelumnya dan Berikutnya; kata "kemarin" di judul hanya label tampilan untuk tanggal kalender kemarin.
- Pesan 403: "Kamu hanya bisa mengisi aktivitasmu sendiri untuk hari ini dan 2 hari kerja sebelumnya."
- Judul kartu: "Aktivitas hari ini", "Aktivitas kemarin", atau "Aktivitas", diikuti tanggal lengkap. Tombol Sebelumnya nonaktif di batas paling awal, Berikutnya nonaktif di hari ini, dengan `aria-label` yang menyebut tanggalnya (atau batasnya bila nonaktif). Hitungan "x dari 8 jam terisi" mengikuti tanggal yang tampil. Saat bukan hari ini, keterangan "Kamu sedang mengisi aktivitas <hari, tanggal>." tampil di atas tabel.
- Simpan otomatis yang terpicu saat kotak kehilangan fokus bisa selesai setelah tanggal diganti. Composable simpan meneruskan tanggal jawaban server, dan kartu mengabaikan jawaban untuk tanggal yang sudah ditinggalkan, supaya isian tanggal lama tidak menimpa tampilan tanggal baru.
- Ekspor Drive: Excel bulanan ditulis ulang utuh setiap malam, jadi isian terlambat di bulan yang sama selalu ikut di malam berikutnya. Celahnya di pergantian bulan: dulu hanya bulan berjalan yang ditulis, sehingga isian akhir bulan lalu (termasuk hari terakhir bulan, karena job berjalan 03.45 WIB) tidak pernah terkirim. Sekarang `ExportMonths` menulis setiap bulan dari tanggal terawal yang masih boleh diisi Staf, atau kemarin bila lebih awal, sampai hari ini. Lampiran aktivitas tidak butuh perubahan: yang disusun adalah semua lampiran yang belum tercatat di `drive_exports`, apa pun tanggalnya.

**Keputusan revisi 38:**
- Dua kolom boolean di `users` (`can_manage_timeline`, `can_manage_notes`), bukan tabel izin terpisah: aksesnya tetap dua, dan kolom di `users` ikut terbaca oleh `GetSessionUser` yang sudah dijalankan di setiap permintaan. Akses tidak disimpan di token sesi, jadi pemberian dan pencabutan langsung berlaku di permintaan berikutnya tanpa login ulang.
- Hanya Staf yang menyimpan akses. Manajer selalu punya keduanya (dihitung di kode, tidak disimpan). Admin tetap hanya membaca Timeline dan Important Notes sesuai aturan Admin yang ada; aksesnya tidak bisa diberikan. CHECK `users_access_staff_only` menolak akses tersimpan untuk peran selain Staf, dan backend mengosongkannya saat peran diganti dari Staf.
- Drawer Ubah pengguna dan form Tambah pengguna punya bagian "Akses tambahan" dengan dua kotak centang dan satu baris keterangan masing-masing. Untuk Manajer keduanya tercentang dan nonaktif dengan teks "Manajer selalu punya akses ini.", untuk Admin tidak tercentang dan nonaktif dengan alasannya. Yang boleh mengubah akses sama dengan yang boleh mengubah pengguna: grup rute `manage` (Admin dan Manajer).
- `PUT /api/users/:id` yang tidak mengirim kolom akses tidak mengubah akses, supaya klien lama atau skrip yang hanya mengubah nama tidak mencabutnya diam-diam.
- Penegakan di backend: tambah, ubah, dan hapus agenda memakai Kelola Timeline; tulis, ubah, urutkan, dan hapus catatan memakai Kelola Important Notes. `can_write` di daftar agenda dan daftar catatan dihitung dari akses masing-masing. Pesan 403: "Akunmu tidak punya akses untuk mengubah bagian ini. Minta Manajer menambahkan aksesnya di halaman Pengguna."
- Audit Log: perubahan akses tercatat di `user_updated` dengan kunci per akses beserta nilai sebelum dan sesudah; Audit Log menulisnya "akses "Kelola Timeline" nonaktif menjadi aktif". Akun baru mencatat aksesnya di `user_created`.
- Grants: tidak ada tabel baru, dan hak SELECT, INSERT, UPDATE pada `users` diberikan per tabel (`ALTER DEFAULT PRIVILEGES` di `00-bootstrap.sql`), jadi kolom baru otomatis tercakup. `10-grants.sql` tidak berubah dan tidak perlu dijalankan ulang.
- Alasan Kehadiran terbuka untuk semua pengguna yang login (transparan): Staf, Manajer, dan Admin menerima jenis alasan dan keterangannya lengkap, termasuk sakit, izin pribadi, dan cuti. Penyamaran "Tidak hadir" dihapus dari backend dan frontend. Keputusan ini menggantikan keputusan privasi revisi 30 yang menunggu Manajer, dan pertanyaannya dihapus dari daftar "Menunggu jawaban Manajer". Form Kehadiran menyebut "Alasan dan keterangan terlihat oleh semua anggota tim."
- Akses "Lihat alasan Kehadiran" yang sempat dirancang dihapus sebelum commit, karena alasan kini terbuka untuk semua. Migrasi 000022 diubah langsung (belum pernah naik ke produksi); DB dev dan test DB lokal diturunkan satu versi lalu dinaikkan lagi dengan migrasi yang baru.

**Revisi 39 (urutan navbar dan halaman awal):**

- [x] S12e-06 Urutan menu dan halaman awal Timeline, logo ke Timeline, E2E `E2E_ONLY=beranda`. Hanya frontend.

**Keputusan revisi 39:**
- Urutan menu top bar: Timeline, Task, Aktivitas, Arsip, lalu sisanya dengan urutan lama (Pengguna, Pengaturan, Audit Log, Status, Profil). Aturan peran tiap menu tidak berubah.
- Bar bawah ponsel paling banyak empat item termasuk Lainnya (keputusan Sprint 9 dan 11). Timeline masuk bar bawah, jadi isinya Timeline, Task, Arsip, dan Lainnya. Profil pindah ke menu Lainnya karena urutan prioritasnya paling akhir; Aktivitas tetap di Lainnya seperti sebelumnya, di urutan pertama.
- Halaman awal semua peran adalah Timeline (`homeFor`), termasuk Admin yang sebelumnya diarahkan ke Pengguna. Semua peran sudah bisa membuka Timeline (rutenya tanpa batas peran, dan backend mengizinkan semua pengguna yang login membaca), jadi tidak ada aturan yang diubah. Yang memakai `homeFor`: "/" untuk pengguna yang sudah login, setelah login dan ganti password paksa tanpa tujuan, dan rute yang tidak boleh dibuka peran itu.
- Tujuan asal tetap dihormati lewat `?kembali=` yang sudah ada sejak S12-16. Validasinya tidak berubah: hanya path internal (diawali "/", bukan `//` atau `/\`, dan bukan halaman login itu sendiri) atau host https yang tercantum di `VITE_ALLOWED_RETURN_HOSTS`. Tujuan ke situs lain diabaikan dan pengguna masuk ke Timeline.
- Logo "Internal" di kiri atas menjadi tautan ke Timeline dengan nama aksesibel "NVM Internal, ke Timeline", tinggi minimal 44 px, dan focus ring.
- E2E `beranda` login lewat form dua kali per run. Limiter login (10 percobaan per IP, satu kembali tiap 30 detik) bisa habis bila E2E dijalankan berulang dalam beberapa menit; test menunggu sesuai `Retry-After` bila mendapat 429.

**Revisi 40 (kolom Completed maksimal 10):**

- [x] S12e-07 `Service.Board` memotong kolom Completed menjadi 10 task terbaru setelah semua filter, `count` kolom berisi total yang cocok; keterangan dan tautan Arsip di papan; test DB `TestKolomCompletedMaksimalSepuluh` dan `TestSortCompletedPemecahSeri`; E2E `E2E_ONLY=completed`. Tanpa migrasi.

**Keputusan revisi 40:**
- Kolom Completed di papan (Task saya dan Semua task) paling banyak 10 task (`task.CompletedLimit`). Kolom lain tidak berubah.
- Urutan memakai `solved_date`. Kolom itu `timestamptz` yang diisi `now()` setiap kali task masuk Completed dan dikosongkan saat keluar (`SetTaskStatus`, CHECK `tasks_solved_only_when_completed`), jadi nilainya sudah waktu presisi perpindahan terakhir ke Completed, termasuk task yang pernah dikembalikan lalu Completed lagi. Riwayat status tidak perlu dibaca. Pemecah seri: nomor task terbesar dulu, dibandingkan sebagai angka supaya T-10000 tetap di atas T-9999.
- Batas 10 diambil setelah aturan 7 hari dan semua filter papan (cari, orang, divisi, label, urgency, deadline), karena semuanya sudah diterapkan di query `ListBoardTasks`. Pemotongan dilakukan di backend sebelum PIC, label, dan `move_to` dimuat, jadi kartu yang tidak tampil tidak ikut dikirim.
- `count` tiap kolom adalah total yang cocok dengan filter; di Completed bisa lebih dari jumlah kartu yang dikirim. Lencana jumlah di judul kolom, tombol pilih kolom di ponsel, dan jumlah total papan memakai `count`, jadi semuanya menampilkan total.
- Bila total Completed lebih dari 10, di bawah kartu terakhir tampil "Menampilkan 10 dari <total> task." dan tautan "Lihat semua di Arsip". Arsip memang hanya berisi task Completed; tautannya membawa filter papan yang juga dikenal Arsip (cari, cakupan, orang, divisi, label, urgency). Filter deadline tidak ada di Arsip, jadi tidak dibawa. Bila total 10 atau kurang, keterangannya tidak tampil.
- Seret ke Completed dan pindah lewat dialog tetap memuat ulang papan dari server setelah berhasil, jadi kartu yang baru selesai langsung muncul di paling atas dan yang ke-11 tergeser keluar.

**Selesai kalau:** `scripts/pra-deploy.sh` LOLOS dengan migrasi 000022 di test DB, DB dev di versi 22, dan E2E `aktivitasmundur`, `akses`, `beranda`, dan `completed` lolos di 360 dan 1280 px.

**Catatan infrastruktur dev (2026-11-24):** database dev `nvm_internal` dan test `nvm_internal_test` pindah dari Postgres di VM (lewat tunnel SSH port 15432) ke PostgreSQL 18 lokal di laptop, `localhost:5432`, role `nvm_dev` yang bukan superuser dan memiliki kedua database. Database dev dan test lama di VM sudah dihapus. Cara membuat role dan database serta format `DATABASE_URL` dan `TEST_DATABASE_URL` ada di `README.md`. `scripts/pra-deploy.sh` memigrasikan test DB sendiri. Penyebutan tunnel SSH di retro Sprint 0 dan S7-00 adalah catatan sejarah. Waktu `go test` sebelum tanggal ini (sekitar 330 sampai 1046 detik) diukur lewat tunnel, jadi tidak bisa dibandingkan langsung dengan run sesudahnya.

## Sprint 12f: Whiteboard (revisi 41)

**Tujuan:** Whiteboard setara Milanote atau whiteboard Canva dengan tema NVM, dalam empat tahap W1 sampai W4. Rancangan, alasan, dan alternatif yang ditolak ada di `docs/whiteboard.md`; bagian ini mencatat langkah dan keputusan yang perlu diuji Sprint 13.

**W1 (fondasi, catatan, Tempat sampah):**

- [x] S12f-01 Migrasi `000023_whiteboard`: `wb_boards` (satu papan akar `is_root`, papan lain selalu punya induk), `wb_cards` (catatan, judul, checklist, dan kartu papan dengan `ref_board_id`), `wb_activity`, kolom Tempat sampah, indeks trigram. Papan akar dibuat migrasi. `deploy/sql/10-grants.sql` berubah: DELETE di `wb_boards` dan `wb_cards`, daftar jadi sepuluh tabel.
- [x] S12f-02 Backend `internal/whiteboard`: validasi isi dari daftar putih (`content.go`), papan akar dan kartu papan, ganti judul papan, kartu dengan patch per field dan 409 isi, operasi checklist per butir, Tempat sampah per batch (kartu papan membawa seluruh subtree), riwayat per papan, audit `wb_deleted`, `wb_restored`, `wb_purged`. Rute `/api/whiteboard/*` (`boards/root` untuk papan akar), tulis hanya Staf dan Manajer.
- [x] S12f-03 Job `maintenance` menghapus permanen batch Tempat sampah lebih dari 30 hari.
- [x] S12f-04 Frontend: satu halaman kanvas `/whiteboard` dan `/whiteboard/:id` selebar sisa layar tanpa gulir halaman; bar atas (breadcrumb, judul yang bisa diubah, cari judul papan, penanda simpan, riwayat); sidebar alat (Catatan, Judul, Checklist, Papan, Tempat sampah) yang bisa diklik atau diseret; tombol Tambah dan lembar alat di bawah 768 px; kartu catatan dengan TipTap, judul, checklist, kartu papan; warna, seret, ubah ukuran, kotak seleksi, keyboard; panel Tempat sampah dan riwayat; menu Whiteboard sesudah Aktivitas.
- [x] S12f-05 Test: unit `content_test.go` dan `tests/whiteboard.test.ts`; DB `TestWbPapanBersarang`, `TestWbKartuPerFieldDanKonflikIsi`, `TestWbChecklistPerButir`, `TestWbTempatSampah`, `TestWbHapusPermanen`, `TestWbRiwayatDigabung`, `TestWhiteboardAdminHanyaMembaca`, `TestWhiteboardKonflikDanTidakAda`; E2E `E2E_ONLY=wb1` (`wbdasar`, `wbkeyboard`, `wbponsel`, `wbadmin`).
- [x] S12f-06 Navigasi: menu atas desktop mulai 1280 px; di bawahnya bar bawah dan menu Lainnya, karena menu Admin dan Manajer meluap di 768 dan 1024 px.

**Keputusan W1:**
- Semua adalah papan (keputusan Manajer, seperti Milanote). Satu papan akar bersama "Whiteboard" dibuat migrasi dan langsung terbuka di menu Whiteboard; tidak ada halaman daftar dan tidak ada folder. Papan akar tidak bisa dihapus atau dipindah (CHECK dan indeks unik parsial); judulnya boleh diubah.
- Papan di dalam papan tampil sebagai kartu jenis `board` dengan judul dan ringkasan isi ("2 papan, 4 kartu"). Klik atau Enter membukanya di `/whiteboard/:id`; di layar sentuh ketukan membukanya. Papan baru dari alat Papan langsung meminta judul di kartunya. Kedalaman maksimal 8 (papan akar 0).
- Staf dan Manajer membuat, mengubah, dan menghapus semua papan dan kartu. Admin hanya membaca, dijaga satu middleware `wbWriter` di semua rute tulis (403 "Admin hanya bisa melihat Whiteboard, tidak bisa mengubahnya."). `can_write` di respons memakai aturan yang sama. Admin tetap bisa melihat Tempat sampah, tanpa tombol Pulihkan.
- Posisi, ukuran, urutan tumpuk, dan warna kartu digabung per field (yang terakhir sampai menang). Isi kartu dijaga `content_version`: versi basi ditolak 409 dan klien menyimpan ketikan sebagai kartu salinan di sebelahnya. Checklist diubah per butir lewat `POST .../checklist`, jadi centang tidak pernah bentrok. Ganti judul papan memakai `version`; versi basi 409.
- Isi catatan disimpan sebagai JSON ProseMirror yang dibangun ulang dari daftar putih (node, mark, atribut). Tautan hanya http, https, dan mailto; lainnya ditolak 400. Dirender tanpa `v-html`. TipTap dengan `injectCSS: false`; CSS ProseMirror yang dibutuhkan disalin ke `style.css`. Gapcursor dan dropcursor dimatikan (kedip tanpa henti dan seret di dalam teks tidak dibutuhkan).
- Hapus selalu ke Tempat sampah, satu batch per aksi; menghapus kartu papan membawa papan anak dan seluruh turunannya. Pulihkan per batch, ditolak bila papan asal kartunya masih di Tempat sampah (dengan nama papan itu). Hapus permanen hanya oleh job harian setelah 30 hari, tanpa tombol. Riwayat papan ikut hilang saat papannya dihapus permanen (ON DELETE CASCADE); audit log mencatat hapus, pulihkan, dan hapus permanen.
- Riwayat per papan: seret, ubah ukuran, urutan, dan warna beruntun dari orang yang sama digabung ke satu baris per 5 menit dengan daftar kartu; edit isi kartu yang sama digabung per 5 menit dengan isi terakhir.
- Warna kartu memakai token latar yang sudah ada (abu, hijau, kuning, biru, krem, merah), tanpa warna baru. Isi kartu krem memakai token mode terang di kedua mode, karena krem tetap terang di mode gelap (ink-muted gelap di atas krem hanya 2,51:1).
- Sidebar alat mulai 768 px: klik menaruh kartu di tengah layar, seret menaruhnya di titik lepas. Di bawah 768 px: tombol Tambah membuka lembar alat, breadcrumb menjadi tombol kembali plus judul, cari dibuka dari ikon. Alat tahap berikutnya baru ditampilkan saat tahapnya selesai.
- Di layar sentuh, sentuhan pada kartu yang belum terpilih menggeser kanvas; ketukan memilih kartu, ketukan pada kartu terpilih membukanya. Di mouse, seret area kosong membuat kotak seleksi; geser kanvas dengan roda, Spasi atau tombol tengah, atau tombol mode geser.
- Perubahan orang lain baru terlihat setelah muat ulang sampai W2.

**Selesai kalau:** `scripts/pra-deploy.sh` LOLOS dengan migrasi 000023 (versi papan akar) di test DB, DB dev di versi 23, `./scripts/check.sh` lolos, dan E2E `wb1` lolos di 360 dan 1280 px.

**W2 (kolaborasi langsung dan edit lanjutan):**

- [x] S12f-07 Backend: hub kolaborasi di memori (`internal/whiteboard/hub.go`) untuk kehadiran, kursor, pilihan, gerakan seret (`grab`, `move`, `release`), kunci lunak dengan draf, dan pindah kunci antar tab akun yang sama; WebSocket `GET /api/whiteboard/ws` (`github.com/coder/websocket` v1.8.15, ISC) dengan cek Origin sendiri, ping tiap 30 detik, cek ulang sesi tiap 60 detik (fungsi `sessionFor` yang sama dengan `requireSession`), batas 64 KB per pesan, 32 koneksi per pengguna, antrean keluar 512 pesan; siaran `changed`, `deleted`, `renamed`, dan `resync` setelah transaksi REST commit; sinkron ulang `join` dengan `last_seq`. Tanpa migrasi.
- [x] S12f-08 `frontend/nginx.conf`: `location = /api/whiteboard/ws` dengan Upgrade, Connection, Host, dan batas baca satu jam. `vite.config.ts`: proxy `ws: true`.
- [x] S12f-09 Frontend: avatar kehadiran (redup saat diam 3 menit, tombol ikuti kursor), kursor bernama, bingkai dan label orang lain, gerakan seret langsung, kunci lunak dengan teks langsung, dialog kunci di tab lain, dialog kartu dihapus saat diedit, dialog papan dihapus, penanda "Terputus, menyambung ulang" dengan antrean perubahan, undo dan redo milik sendiri, salin, tempel, duplikat, rata dan sebar, minimap.
- [x] S12f-10 Test: unit `tests/whiteboard.test.ts` (jeda sambung ulang, undo melewati field orang lain); DB `TestHubKehadiran`, `TestHubKunciLunak` (100 putaran bersamaan), `TestHubPindahKunciDuaTab`, `TestHubSeret`, `TestHubAdminHanyaMelihat`, `TestHubHapusSaatDiedit`, `TestHubSinkronUlang`, `TestHubAntreanPenuh`, `TestWhiteboardWSHandshake`, `TestWhiteboardWSSiaranSetelahCommit`, `TestWhiteboardWSSesiDicabut`; E2E `E2E_ONLY=wb2` (tujuh skenario multi-konteks plus audit navbar), `scripts/e2e/uji-wss.mjs` (wss di Chrome, Edge, Firefox di bawah CSP produksi), `scripts/e2e/uji-restart.mjs` (backend dimatikan dan dinyalakan lagi seperti rilis.sh).

**Keputusan W2:**
- Perubahan data tetap lewat REST; WebSocket hanya membawa hal sementara dan pemberitahuan. Siaran `changed` dikirim setelah commit ke semua koneksi papan itu, termasuk tab lain milik pengirim; perubahan yang ditolak (400, 403, 409) tidak pernah disiarkan. Klien mengabaikan kartu yang versinya tidak lebih baru.
- Kunci lunak per kartu saat mengetik (catatan, judul, checklist, judul kartu papan). Pemegang mengirim draf paling sering tiap 150 ms; hub memvalidasinya dengan pembersih yang sama dengan REST sebelum meneruskan. Kunci lepas saat editor ditutup, koneksi putus, atau tanpa detak 30 detik (klien berdetak tiap 10 detik). Orang lain melihat teks berubah dan label "<nama> sedang mengedit"; mereka tetap boleh memindah, mengubah ukuran, dan mewarnai kartu itu.
- Dua tab akun yang sama: kunci milik pasangan pengguna dan tab. Tab kedua menawarkan "Edit di sini"; tab pertama menyimpan dulu lalu melepas, atau kunci pindah setelah 3 detik bila tab pertama tidak menjawab.
- Kartu yang sedang diedit dihapus orang lain: hub menyimpan draf terakhir pemegang, dan handler hapus menyimpannya sebagai isi kartu di transaksi hapus yang sama (tanpa meminta dan menunggu draf seperti rancangan W0, yang lebih rumit dan tidak lebih aman). Pemegang mendapat dialog dengan nama penghapus dan tombol Pulihkan kartu.
- Seret: `grab` pertama yang sampai ke server menang; seret yang kalah dibatalkan dengan label "<nama> sedang memindahkan kartu ini". Gerakan disiarkan 25 kali per detik, dibatasi 30 per detik per koneksi di server. Kartu orang yang putus di tengah seret kembali ke posisi tersimpan setelah 3 detik.
- Terputus: perubahan REST yang gagal karena jaringan atau backend tidak menjawab masuk antrean memori (bukan localStorage), penanda "Terputus, menyambung ulang" tetap tampil, dan antrean dikirim berurutan setelah tersambung dan `join` menyusulkan perubahan orang lain. Perubahan yang ditolak setelah tersambung tidak diulang; papan dimuat ulang. Sambung ulang 1, 2, 4, 8, 16, lalu 30 detik ditambah acak sampai 20%.
- Sesi diperiksa ulang tiap 60 detik selama koneksi hidup; sesi yang dicabut menutup koneksi dengan kode 4401 dan klien diarahkan ke halaman masuk. Admin boleh terhubung, terlihat di kehadiran, dan kursornya terlihat; `lock`, `grab`, `draft`, dan `select` dari Admin ditolak.
- Undo dan redo hanya aksi sendiri, per tab, maksimal 100 langkah, hilang saat pindah papan. Tata letak hanya membatalkan field yang nilainya masih hasil aksi sendiri; isi hanya dibatalkan bila versi isinya belum diubah orang lain; checklist memakai operasi kebalikan per butir. Bila ada yang dilewati: "Sebagian langkah tidak dibatalkan karena sudah diubah orang lain."
- Salin dan tempel memakai teks clipboard berawalan `nvm-whiteboard:` (maksimal 100 kartu), jadi bisa antar papan dan antar tab. Kartu papan ditempel sebagai papan baru kosong berjudul "<judul> (salinan)"; isinya tidak ikut. Duplikat (Ctrl+D) sama dengan salin lalu tempel 24 px ke kanan bawah.
- Warna kolaborasi: token baru `--color-collab-1` sampai `--color-collab-8` plus `--color-on-collab` untuk terang dan gelap, karena token status hanya empat latar pucat. Terhitung: label minimal 5,48:1, warna terhadap canvas dan surface minimal 4,9:1. Warna per orang tetap (`user_id mod 8`), selalu disertai nama.
- Build produksi tidak lagi menyisipkan font sebagai `data:` URI (`build.assetsInlineLimit` di `vite.config.ts`). CSP produksi `font-src 'self'` memblokirnya, jadi subset kecil JetBrains Mono gagal dimuat di produksi sejak sebelum Whiteboard; ditemukan oleh `uji-wss.mjs`.
- Uji restart memakai backend kedua (127.0.0.1:8081) dan Vite kedua (5174, `VITE_API_TARGET`), supaya backend dev yang sedang dipakai tidak ikut dimatikan.

**Selesai kalau:** `scripts/pra-deploy.sh` LOLOS, `./scripts/check.sh` lolos, E2E `wb1` dan `wb2` lolos di 360 dan 1280 px (audit navbar di 768, 1024, 1279, dan 1280 px), `uji-wss.mjs` lolos di Chrome, Edge, dan Firefox, dan `uji-restart.mjs` lolos.

**W3 (berkas, tautan, kolom, garis, gambar bebas, tabel, komentar, pindah papan):**

- [x] S12f-11 Migrasi `000024_whiteboard_media`: `wb_files`, `wb_link_previews`, `wb_connectors`, `wb_comments`; `wb_cards` mendapat jenis `image`, `file`, `link`, `column`, `drawing`, `table` serta kolom `file_id`, `column_id`, `col_order` dengan CHECK; `wb_boards.resync_seq`. `ListAttachmentFileIDs` mencakup `wb_files`, dan pembersih berkas yatim memperlakukan `<id>.thumb.jpg` sama dengan berkas aslinya. `deploy/sql/10-grants.sql`: DELETE jadi tiga belas tabel (tambah `wb_connectors`, `wb_files`, `wb_link_previews`).
- [x] S12f-12 Backend: paket `internal/linkpreview` (pengambil pratinjau anti SSRF), layanan W3 di `internal/whiteboard/w3.go` dan `content_w3.go` (unggah dengan thumbnail, buka berkas, pratinjau dengan cache dan batas per orang, operasi tabel, garis, komentar, pindah papan, keluar masuk kolom), hub `Connectors`, `ConnectorsGone`, `Moved`, `LockedByOther`, rute di `whiteboard_w3_handlers.go`. Job harian juga membuang berkas tanpa kartu (lebih dari 1 hari), cache pratinjau lama (lebih dari 7 hari, tidak dipakai kartu), dan garis di Tempat sampah lebih dari 30 hari.
- [x] S12f-13 Frontend: alat Berkas, Tautan, Kolom, Gambar, dan Tabel di sidebar dan lembar alat (ditampilkan setelah alatnya berfungsi penuh); tempel tangkapan layar dan seret berkas dari komputer ke kanvas; kartu gambar dari thumbnail, kartu berkas dengan tombol unduh, kartu tautan dengan pratinjau; kolom yang menumpuk isinya; lapisan garis penghubung dan dialog garis; panel komentar dan lencana jumlah; dialog Pindahkan ke papan dan lepas kartu di atas kartu papan; mode pena gambar bebas tanpa kartu dengan toolbar melayang (revisi A); editor tabel per sel dengan judul (revisi B).
- [x] S12f-14 Test: unit Go `linkpreview_test.go` (semua kelas alamat terlarang, IP langsung, alamat campuran, IPv4-mapped, NAT64, redirect ke IP dan ke nama internal, lebih dari tiga redirect, host IPv6 saja gagal di bawah 500 ms, dual stack lewat IPv4, bukan HTML, 404, body dan gambar besar, batas waktu, nama tidak ditemukan), `content_w3_test.go` (validasi gambar bebas dan tabel, operasi tabel); DB `TestRunMenjagaBerkasWhiteboard` (job harian menjaga berkas Whiteboard dan `.thumb.jpg`-nya, termasuk kartu di Tempat sampah, dan menghapus berkas yatim sungguhan beserta thumbnail yatim), `TestWbUnggahBerkas`, `TestWbKolomDanPenghubung`, `TestWbKomentar`, `TestWbPindahPapan`, `TestWbTabel`, `TestWbPratinjauDiblokir`, `TestWhiteboardWSTabelDanGambarBebas`, `TestWhiteboardWSGarisDanPindah`, rute W3 di `TestWhiteboardAdminHanyaMembaca`; unit frontend (layout kolom, urutan kolom, ujung garis, penyederhanaan goresan); E2E `E2E_ONLY=wb3` (wbberkas, wbtautan dengan server lokal, wbkolom, wbpenghubung, wbkomentar, wbgambar, wbtabel, wbpindah).

**Keputusan W3:**
- Berkas: maksimal 50 MB; JPG, PNG, WebP, GIF, PDF, DOC, DOCX, XLS, XLSX, PPT, PPTX, TXT, dan CSV, diperiksa dari isi berkas (SVG selalu ditolak, juga yang berkedok PNG). Gambar mendapat thumbnail JPEG paling besar 1200 px per sisi yang disimpan ulang dari piksel (EXIF dan lokasi foto tidak ikut, arah putar diterapkan); resolusi di atas 40 megapiksel ditolak. Tinggi kartu gambar baru mengikuti perbandingan sisi gambar. Berkas disajikan dengan `nosniff` dan `Content-Security-Policy: sandbox`; gambar dan PDF tampil di tab, sisanya diunduh. Tempelan tangkapan layar diberi nama `tempel-<waktu>`.
- Salin tempel kartu gambar dan berkas memakai berkas yang sama (`file_id`); jenis kartu harus cocok dengan jenis berkasnya. Berkas tanpa kartu (termasuk di Tempat sampah) dihapus job harian setelah 1 hari.
- Pratinjau tautan: server me-resolve nama sendiri dan memeriksa setiap alamat (loopback, privat, link-local termasuk metadata cloud, CGNAT, multicast, dokumentasi, IPv4-mapped, NAT64 ke alamat terlarang); satu alamat terlarang saja sudah membuat pratinjau diblokir. Koneksi hanya lewat IPv4 karena VM produksi tanpa rute IPv6; host yang hanya punya IPv6 gagal seketika dengan pesan, tidak menunggu batas waktu. Port hanya 80 dan 443, tanpa nama pengguna di URL, paling banyak 3 redirect yang masing-masing diperiksa ulang, 5 detik, 512 KB HTML, tanpa cookie dan proxy. og:image diambil lewat pemeriksaan yang sama lalu disimpan ulang sebagai JPEG, jadi peramban tidak pernah memuat gambar dari situs luar dan CSP tidak dilonggarkan. Hasil diblokir dan gagal ikut disimpan beserta alasannya (dicoba lagi paling cepat 1 jam); hasil berhasil dipakai ulang 7 hari. Batas 30 pengambilan per orang per jam. Kartu tautan tetap dibuat tanpa pratinjau bila pratinjau gagal.
- Kolom: satu tingkat (kolom tidak bisa masuk kolom). Isi kolom ditumpuk di bawah judul selebar kolom, dan kolom memanjang mengikuti isinya; posisi tersimpan isi kolom diabaikan. Seret kartu ke kolom menyisipkannya di titik lepas; seret keluar menaruhnya di titik lepas; Shift dan panah atas atau bawah mengubah urutan. Hapus kolom ikut menghapus isinya di batch yang sama.
- Garis penghubung: dibuat dari dua kartu terpilih (urutan pilihan menentukan arah), diubah lewat dialog (label maksimal 200 karakter, ujung polos atau panah, garis utuh atau putus-putus, warna). Garis ikut ke Tempat sampah bersama kartu ujungnya dan kembali saat dipulihkan.
- Model data garis sudah memuat kebutuhan diagram W5 (revisi C), supaya W5 tidak memigrasi ulang data garis: `from_card_id` dan `to_card_id` boleh NULL (ujung bebas, titiknya di `geometry`); `geometry` jsonb berisi port per ujung (auto, empat sisi, empat sudut), jalur `straight`, `orthogonal`, atau `curved`, dan paling banyak 50 titik belok; `style` menerima tujuh jenis ujung (`none`, `arrow`, `open_arrow`, `triangle`, `diamond`, `diamond_open`, `circle`) dan tiga gaya garis (`solid`, `dashed`, `dotted`); label tiga posisi (`label` di tengah, `label_start`, `label_end`). Semuanya divalidasi server. UI W3 baru memakai garis lurus antar dua kartu.
- Pindah papan: lewat dialog atau melepas kartu di atas kartu papan. Kartu papan membawa papan anaknya; papan tidak bisa dipindah ke dalam dirinya sendiri atau melewati batas kedalaman. Garis yang kedua ujungnya ikut pindah ikut pindah; garis yang tertinggal satu ujungnya masuk Tempat sampah (batch sendiri, bisa dipulihkan). Kartu yang dikunci orang lain tidak bisa dipindah (409). Rombongan kartu ditaruh di kanan isi papan tujuan. Papan asal mencatat `resync_seq`, jadi klien yang tersambung ulang memuat ulang papan utuh; klien papan tujuan menerima `resync`.
- Gambar bebas tanpa kartu (revisi A): alat Gambar menyalakan mode pena; coretan langsung di kanvas, di atas kartu, tanpa bingkai atau latar. Keluar lewat Escape, alat lain, atau Selesai di toolbar pena melayang (warna, ketebalan, penghapus per goresan, hapus goresan terakhir). Satu sesi pena orang yang sama menjadi satu objek (kartu jenis `drawing`); lewat 300 goresan, 6.000 titik, atau lebar 4.000 px, sesi otomatis memulai objek baru. Klien mengirim titik koordinat dunia; server menghitung kotak batas (ditambah setengah tebal pena plus 2 px, minimal 40 px) dan menyimpan titik bulat relatif ke pojok kotak itu. Isi dan ukuran objek hanya berubah lewat operasi pena (`add`, `remove`, `color`); objek bisa dipindah, diwarnai ulang, dan dihapus ke Tempat sampah. Coretan dirender di atas kartu; di luar mode pena hanya garisnya (tebal pena plus 8 px layar) yang menangkap klik, sisanya tembus ke kartu di bawah. Goresan yang sedang digambar disiarkan lewat WebSocket (`ink`, sekitar 15 kali per detik, divalidasi hub), lalu disimpan lewat REST saat goresan selesai. Objek sesi dikunci selama sesi; penghapus akun lain di objek itu ditolak 409 dengan nama pemegangnya. Di ponsel satu jari menggambar, dua jari menggeser dan zoom. Mode pena diumumkan lewat aria-live; objek gambar bisa difokus dengan panah, dihapus dengan Delete, dan berlabel "Gambar bebas oleh <nama>". Warna dari daftar putih (tinta plus enam nama warna kartu), ketebalan 2, 4, atau 8. Warna pena dan garis memakai token tinta gelap, bukan latar kartu yang pucat: terhitung minimal 4,95:1 di mode terang (terendah `high` di atas krem) dan 7,83:1 di mode gelap. Kartu krem di mode gelap memakai nilai mode terang untuk warna pena juga.
- Tabel: 1 sampai 100 baris, 1 sampai 20 kolom, 500 karakter per sel, teks polos. Perubahan per sel, baris, atau kolom lewat operasi di server. Judul tabel (revisi B) maksimal 200 karakter di atas grid, diubah lewat operasi `title` selama kartu dikunci, dan ikut teks pencarian (`plain`).
- Kunci lunak ditegakkan di server untuk tabel dan gambar bebas: operasi tabel dan patch isi gambar atau tabel dari akun lain ditolak 409 "<nama> sedang mengubah kartu ini." Tab lain milik akun yang sama tetap boleh. Catatan tetap dijaga `content_version` seperti W2.
- Komentar: 1 sampai 2.000 karakter teks polos. Mengubah hanya penulisnya; menghapus penulisnya atau Manajer (pilihan bawaan pertanyaan 1). Komentar dihapus lewat `deleted_at`. Jumlah komentar ikut di kartu dan disiarkan, jadi lencana dan panel orang lain langsung berubah. Admin bisa membaca komentar, tanpa form.
- Zoom roda (perbaikan bug di W3): listener wheel native `{ passive: false }` di kanvas; Ctrl atau Cmd (juga pinch touchpad) zoom di sekitar kursor dengan faktor eksponensial terhadap deltaY yang dinormalisasi untuk deltaMode baris (16 px) dan halaman (tinggi kanvas), satu tingkat roda sekitar 22%, dibatasi 240 px per event; batas 10% sampai 400%. Tanpa Ctrl roda menggeser kanvas, Shift ke samping; halaman tidak pernah tergulir dan zoom browser tertahan. Wheel biasa di dalam editor atau bagian yang bisa digulir menggulir bagian itu. Event digabung per frame (requestAnimationFrame), tanpa transisi. Persentase zoom menjadi tombol kembali ke 100%.
- Dobel klik untuk mengedit (tambahan W3): dobel klik (dan dobel ketuk dalam 350 ms di layar sentuh) membuka editor catatan, judul, butir checklist yang diklik, dan sel tabel yang diklik, dengan kursor di posisi klik (offset karakter dari teks tampilan; untuk catatan dipetakan ke posisi dokumen TipTap). Klik sekali hanya memilih; centang tetap sekali klik. Kartu papan: klik sekali memilih, dobel klik membuka papannya (Enter juga). Dobel klik area kosong tidak membuat apa pun. Dobel klik kartu yang dikunci orang lain membuka tampilan hanya baca berlabel "Hanya baca: <nama> sedang mengedit"; teksnya tetap berubah langsung lewat draf. Tombol Ubah isi di bar pilihan tetap ada. Dobel klik ditangani di kanvas (kartu dicari dari titik klik), karena pointer capture saat klik mengalihkan dblclick ke elemen kanvas.
- Tabel memakai draf lokal per sel dan judul: jawaban server untuk operasi lain tidak lagi menimpa ketikan yang belum dikirim.
- Checklist (judul dan butir) dan label garis memakai draf lokal yang sama (`frontend/src/components/whiteboard/drafts.ts`): siaran perubahan orang lain atau jawaban operasi lain yang datang saat mengetik tidak lagi menimpa ketikan; dibuktikan E2E `wbketik` yang gagal di kode sebelumnya (butir kembali ke teks lama, label garis jadi kosong). Isian lain sudah aman karena memakai v-model ke state lokal sejak awal: judul papan, judul kartu papan, judul, judul kolom, keterangan berkas, komentar (baru dan ubah), alamat tautan, dan cari. Catatan memakai TipTap yang tidak membaca ulang isi dari server selama editor terbuka.
- Area klik garis penghubung dan coretan memakai atribut SVG `pointer-events="stroke"`; kelas Tailwind `pointer-events-[stroke]` tidak menghasilkan CSS, jadi sebelumnya garis tidak bisa diklik.
- Undo dan redo W3: buat kartu (termasuk unggah, tempel, dan objek gambar baru) dan isi (keterangan, judul kolom, tabel) ikut riwayat. Goresan tambahan di objek yang sama dibatalkan lewat Hapus goresan terakhir. Keluar masuk kolom dan pindah papan tidak ikut riwayat; pindah papan menampilkan tombol "Buka papan".

**Selesai kalau:** `scripts/pra-deploy.sh` LOLOS, `./scripts/check.sh` lolos, DB dev di versi 24, dan E2E `wb1`, `wb2`, dan `wb3` lolos di 360 dan 1280 px.

**W4 (template, ekspor, pencarian isi, Tampilan Daftar, level of detail, performa):**

- [x] S12f-15 Backend tanpa migrasi (kolom `wb_boards.is_template` sudah ada sejak `000023`): `internal/whiteboard/w4.go` dengan `SetTemplate`, `Templates`, `FromTemplate` (salin subtree papan, kartu, isi kolom, dan garis dalam satu transaksi), dan `SearchCards`; rute `GET /api/whiteboard/templates`, `PUT /api/whiteboard/boards/:id/template`, `POST /api/whiteboard/boards/:id/from-template`; `GET /api/whiteboard/search` kini menjawab `results` (papan) dan `cards` (kartu). Query baru di `whiteboard.sql`: `SetWbTemplate`, `ListWbTemplates`, `ListWbSubtreeBoards`, `ListWbCardsForCopy`, `InsertWbCardCopy`, `ListWbConnectorsForCopy`, `SearchWbCards`.
- [x] S12f-16 Frontend: tombol Jadikan template di bar atas (lembar Tambah di ponsel) dan alat Template yang membuka dialog pilih template; kotak cari mencari papan dan isi kartu, hasil kartu membuka papannya dengan kartu itu terpilih dan terfokus (`?kartu=` dibuang setelah dipakai); dialog Ekspor (PNG atau PDF, seluruh papan atau kartu terpilih) dengan penggambar kanvas `components/whiteboard/exportBoard.ts` dan penulis PDF `utils/wbExport.ts`; Tampilan Daftar (`WbListView.vue`); level of detail di `WbCard.vue`.
- [x] S12f-17 Poles yang ditemukan uji W4: bar atas dua baris di bawah 1280 px (lokasi dan judul di baris pertama, kontrol di baris kedua), Riwayat dan Daftar hanya ikon di bawah 768 px, label simpan saat putus cukup "Terputus", Pas ke layar di ponsel mengosongkan 160 px atas dan 120 px bawah.
- [x] S12f-18 Test: unit Go `TestSnippet`, `TestSearchTerm`; DB `TestWbTemplate`, `TestWbCariIsi`, `TestWbExplain5000` (EXPLAIN ANALYZE SQL sqlc asli dengan 5.000 kartu), `TestTemplateBerkasBertahanSetelahTemplateDihapusPermanen` (template berisi gambar dihapus permanen setelah 31 hari: kartu salinan, baris `wb_files`, berkas, dan thumbnail bertahan dan masih bisa dibuka; setelah salinan juga dihapus permanen barulah berkas dibuang sebagai yatim; aturan FK dikunci di test), rute W4 di `TestWhiteboardAdminHanyaMembaca`; unit frontend `tests/wbExport.test.ts` (skala, struktur PDF: header, offset xref ke tiap objek, startxref, panjang stream, byte JPEG utuh, batas halaman; nama berkas; pemecah baris) dan `cardTitle`, `cardLines`, `fitView` dengan jarak atas dan bawah di `tests/whiteboard.test.ts`; E2E `E2E_ONLY=wb4` (`wbtemplate`, `wbekspor`, `wbcari`, `wbdaftar`, `wbperforma`).
- [x] S12f-19 Audit aksesibilitas dan kontras W4 (lihat keputusan di bawah). Ekspor Drive diperiksa: hanya lampiran task dan Aktivitas, jadi berkas Whiteboard tidak ikut (pertanyaan 2, bawaan "tidak", tanpa perubahan kode).

**Keputusan W4:**
- Template: papan mana pun kecuali papan akar boleh ditandai template oleh Staf dan Manajer (pertanyaan 4, bawaan). Papan akar ditolak 400. Membuat dari template menyalin papan beserta semua papan turunannya, kartu, isi kolom, dan garis yang kedua ujungnya ikut tersalin; komentar, riwayat, dan kartu di Tempat sampah tidak ikut. Salinan bukan template. Kartu gambar dan berkas memakai berkas yang sama (`file_id`), seperti salin tempel, bukan hardlink seperti rancangan W0: tidak ada byte yang disalin, dan job harian sudah menjaga berkas yang masih dipakai kartu mana pun. Batas 2.000 kartu per salinan dan batas kedalaman papan tetap berlaku. Template yang sedang dibuka tidak bisa dipilih di papannya sendiri, supaya template tidak membesar berisi salinan dirinya.
- Pencarian isi: kata kunci 2 sampai 100 karakter (`%` dan `_` dicari apa adanya), mencari teks kartu (`plain`, sudah mencakup catatan, judul, checklist, tabel, keterangan berkas) dan judul pratinjau tautan, hanya kartu yang tidak di Tempat sampah, paling banyak 50 hasil. Tiap hasil membawa jalur papannya dan cuplikan sekitar kata kunci.
- Ekspor: dikerjakan di peramban, tidak ada yang dikirim ke server. Digambar ulang dari data kartu (bukan tangkapan DOM, karena kartu di luar layar tidak di-mount dan tangkapan DOM butuh pustaka tambahan): teks polos per baris tanpa format huruf, gambar dari thumbnail, tabel sebagai kisi, goresan, dan garis dengan panah dan label. Warna selalu token mode terang supaya layak dicetak. Skala 2x, diperkecil bila melewati 16 juta piksel atau 16.384 px per sisi, dengan keterangan skala di dialog. PDF satu halaman berisi JPEG (DCTDecode) dengan ukuran halaman 96 dpi, maksimal 14.400 poin per sisi. Nama berkas dari judul papan.
- Tampilan Daftar: tombol Daftar (toggle) mengganti kanvas menjadi daftar per kolom lalu "Di luar kolom", urutan baca. Aksi sama dengan kanvas lewat fungsi halaman yang sama: Lihat di kanvas, Buka papan, Ubah, centang butir checklist, Pindah ke papan lain, Hapus (ke Tempat sampah dengan Urungkan). Kanvas tetap terpasang, jadi kembali ke kanvas tidak memuat ulang papan.
- Level of detail: di bawah zoom 40% kartu hanya menampilkan judul ringkas dan warnanya, huruf dibagi zoom supaya sekitar 14 px di layar. Gambar tetap menampilkan pratinjaunya; kartu yang sedang diedit selalu utuh. Label aksesibel kartu tidak berubah.
- Performa (build produksi, `vite preview`, 1280 px, papan 500 kartu termasuk 50 gambar dan 20 gambar bebas, 100 garis): muat 187 sampai 223 ms, p95 frame 13,4 ms saat geser 3 detik (batas 2 detik dan 20 ms). Lewat Vite dev p95 berkisar 13 sampai 27 ms karena Vue mode pengembangan, jadi E2E hanya menilai batas frame di build produksi. EXPLAIN 5.000 kartu di test DB: muat papan 15,5 ms memakai `wb_cards_board_idx`; cari 3,1 ms dengan seq scan (planner tidak memilih indeks trigram di 5.000 baris; ditinjau ulang bila papan jauh lebih besar).
- Kontras W4 (terhitung): teks redup Tampilan Daftar dan dialog 6,42:1 (terang) dan 5,97:1 (gelap) di surface; tombol toggle tertekan memakai pasangan accent yang sudah ada, 10,16:1 dan 14,99:1 saat hover; judul ringkas level of detail 13,87:1 di kartu kuning terang dan 13,67:1 gelap. Tanpa warna baru.
- Aksesibilitas W4: tombol aksi Tampilan Daftar memakai `aria-label` yang diawali teks terlihatnya (label in name) lalu judul kartu, karena span `sr-only` yang absolut dihitung sebagai blok dan menambah spasi di nama tombol. Dialog Ekspor tidak memakai wadah `aria-live` tersendiri; pesan memakai `role="status"` dan `role="alert"`. Tombol penutup dialog Ekspor bernama Selesai supaya tidak kembar dengan tombol Tutup bawaan dialog. Toggle Jadikan template dan alat Template berbeda nama karena aksinya berbeda.
- Bar atas: di bawah 1280 px dua baris (lokasi dan judul, lalu kontrol), karena satu baris tidak cukup untuk judul di 360, 768, dan 1024 px (judul tinggal 0 sampai 45 px) dan pergantian teks penanda simpan memindah bar ke dua baris di tengah klik, sehingga klik mendarat di elemen lain. Tinggi bar kini tetap untuk Tersimpan, Menyimpan..., Gagal disimpan, dan Terputus.
- Pas ke layar di bawah 768 px mengosongkan 160 px atas (bar pilihan dua baris) dan 120 px bawah (kontrol zoom dan Tambah), supaya kartu yang baru diketuk tidak tertutup bar pilihannya sendiri.

**Selesai kalau:** `scripts/pra-deploy.sh` LOLOS, `./scripts/check.sh` lolos, unit test frontend lolos, E2E `wb1` sampai `wb4` lolos di 360 dan 1280 px, dan `wbperforma` lolos di build produksi.

**W5 (bentuk dan diagram, revisi C):**

- [x] S12f-20 Migrasi `000025_whiteboard_shapes`: CHECK jenis kartu menerima `shape`. Down menghapus kartu bentuk dulu (garisnya ikut lewat ON DELETE CASCADE) lalu memasang CHECK W3. Tanpa tabel baru, jadi `deploy/sql/10-grants.sql` tidak berubah (tetap tiga belas tabel DELETE).
- [x] S12f-21 Backend: `internal/whiteboard/content_w5.go` (validasi isi bentuk). Bentuk memakai jalur kartu yang sudah ada: buat, patch isi dengan `content_version` dan 409, kunci lunak dan draf hub (`CleanContent`), siaran, undo, salin tempel, Tempat sampah, pencarian (`plain`), dan template. Garis tidak berubah di backend: port, jalur, titik belok, tujuh ujung, dan tiga label sudah divalidasi sejak `000024`.
- [x] S12f-22 Frontend: alat Bentuk dan palet (`WbShapePalette.vue`), kartu bentuk (`WbShape.vue`) dengan delapan pegangan ubah ukuran dan pegangan putar di `WbCard.vue`, panel gaya bentuk (`WbShapeStyle.vue`), lapisan garis baru (`WbConnectorLayer.vue`), pegangan diagram (`WbDiagramHandles.vue`: port, titik belok, titik sisip, garis sementara), panel gaya garis lengkap (`WbConnectorDialog.vue`), garis terpilih dengan Delete dan Enter, ekspor bentuk dan jalur garis. Logika murni di `utils/wbShapes.ts`.
- [x] S12f-23 Test: unit Go `TestCleanShape`; DB `TestWbBentuk`; unit frontend `tests/wbShapes.test.ts` (path semua bentuk, port setelah putar, jalur siku tidak menembus kotak asal dan tujuan termasuk bentuk diputar dan port saling membelakangi, titik belok, melengkung, label, ujung, ubah ukuran tegak dan diputar, sudut dengan snap, `routeConnector`); E2E `E2E_ONLY=wb5` (flowchart dan diagram kelas, dua konteks); `wbperforma` kini 500 kartu dengan 100 bentuk dan 100 garis.
- [x] S12f-24 Gerbang akhir: `uji-wss.mjs` diperluas dengan WebSocket papan task (revisi 47) di Chrome, Edge, dan Firefox; `uji-restart.mjs` dengan papan task terbuka; audit luapan baru `scripts/e2e/uji-lebar.mjs` (768 dan 1024 px, Staf dan Manajer, halaman utama dan semua panel Whiteboard); regresi `wb1` sampai `wb5`.

**Keputusan W5:**
- Satu jenis kartu `shape` dengan 22 bentuk dari daftar putih: Dasar (persegi, persegi sudut bulat, elips, segitiga, belah ketupat, segi enam, bintang), Flowchart (proses, keputusan, mulai atau selesai, data, database, dokumen), UML (kelas, aktor, catatan UML, paket), dan Panah blok (kanan, kiri, atas, bawah, dua arah). Isi: `rotation` 0 sampai 359 (bilangan bulat), `fill` nama warna kartu atau "putih" atau kosong, `stroke` warna pena, `stroke_width` 1 sampai 8, `stroke_style` utuh atau putus-putus, `align`, `font_size` 12, 14, 16, 20, 24, atau 32, `text` maksimal 2.000 karakter dengan baris baru tetap. Kelas UML memakai `name` (maksimal 200), `attributes`, dan `methods` (masing-masing maksimal 2.000, satu per baris) dan `text` kosong. Nilai di luar daftar ditolak 400, tidak dibetulkan diam-diam.
- Bentuk digambar SVG dari angka (`shapePath`), teks lewat interpolasi Vue, tanpa `v-html`. Isian krem memakai token mode terang seperti kartu krem. Bentuk baru dari palet hanya dipilih (tidak langsung diedit); teks diketik dengan dobel klik, Escape atau keluar dari isian menyimpan.
- Putar: seluruh kartu bentuk diputar terhadap pusatnya, termasuk garis pilihan dan pegangannya; posisi, culling, dan seleksi kotak tetap memakai kotak tegak. Pegangan putar 100 px layar di atas bentuk; Shift kelipatan 15 derajat. Ubah ukuran: delapan pegangan; geser diproyeksikan ke sumbu bentuk yang diputar dan titik seberang pegangan tetap; Shift menjaga proporsi; snap grid hanya untuk bentuk tegak. Jalur keyboard untuk putar dan ukuran: panel Gaya bentuk (Aksi lain di bar pilihan).
- Port: tengah keempat sisi untuk semua kartu, ditambah empat sudut untuk bentuk; dihitung ulang dari posisi, ukuran, dan sudut. Port "otomatis" memilih sisi yang paling menghadap ujung lain. Pegangan port tampil 48 px layar di luar tepi kartu terpilih, dengan garis tipis ke titik sambungnya, karena di tepi ada pegangan ubah ukuran yang sama letaknya (ditemukan E2E: klik pegangan kiri menarik garis, bukan mengubah ukuran). Tarik dari port ke kartu lain membuat garis; ke area kosong membuat ujung bebas. Garis dari atau ke bentuk memakai jalur siku dan ujung panah.
- Jalur siku keluar lurus dari port sampai lewat kotak batas (untuk bentuk diputar, kotak batas setelah diputar), lalu memilih rute terpendek dengan belokan paling sedikit yang tidak menembus kotak asal dan tujuan; bila rute langsung menembus, rute keliling dicoba. Titik belok dilewati berurutan; seret titik belok untuk memindah, titik sisip di tengah tiap bagian untuk menambah, dobel klik untuk membuang (juga tombol Hapus titik belok di panel). Jalur melengkung memakai Bezier dari arah port, atau kurva halus melewati titik belok.
- Garis bisa dipilih dengan klik (area 16 px layar); Delete menghapus ke Tempat sampah, Enter atau dobel klik membuka panel gaya, Escape melepas. Panel gaya: tiga label (tengah, dekat awal, dekat akhir), tujuh jenis ujung per sisi, utuh, putus-putus, atau titik-titik, warna, jalur, dan port per ujung.
- Ekspor PNG dan PDF menggambar bentuk dengan path yang sama dan garis dengan `routeConnector` yang sama dengan layar.
- Level of detail tidak berlaku untuk bentuk: bentuknya sendiri sudah ringkas.
- Performa (build produksi, 1280 px, 500 kartu termasuk 100 bentuk sebagian diputar dan 100 garis sebagian siku): muat 155 sampai 190 ms, p95 frame 13,4 ms.

**Selesai kalau:** `scripts/pra-deploy.sh` LOLOS dengan migrasi 000025 di test DB, DB dev di versi 25, `./scripts/check.sh` lolos, E2E `wb1` sampai `wb5` lolos di 360 dan 1280 px, `wbperforma` lolos di build produksi, `uji-wss.mjs` lolos di Chrome, Edge, dan Firefox, `uji-restart.mjs` lolos, dan `uji-lebar.mjs` lolos di 768 dan 1024 px.

**Perbaikan setelah rilis W5 (label kursor kolaborasi tidak terbaca di tema terang):**
- Penyebab: token `--color-collab-1` sampai `8` ada di blok `@theme`, dan Tailwind v4 hanya mengeluarkan variabel `@theme` yang dirujuk utilitas atau CSS. Warna kolaborasi dipilih per orang lewat indeks di inline style, jadi nilai mode terang indeks 1 dan 3 sampai 8 tidak pernah ada di CSS hasil build; hanya indeks 2 yang dirujuk `.wb-card-krem`. Label tampil transparan dengan teks putih (1,12:1), bingkai kartu orang lain tidak tergambar. Mode gelap aman karena tokennya di blok CSS biasa. Hitungan kontras W2 lolos karena memakai nilai di berkas sumber, bukan warna yang terpasang di halaman.
- Perbaikan: token kolaborasi dan token teks per warna `--color-on-collab-1` sampai `8` di blok `:root` biasa untuk kedua tema; aturan `[data-collab='n']` memasang `--wb-remote` (latar solid) dan `--wb-remote-on` (teks) per indeks, dipakai kursor, label kursor, label dan bingkai kartu, gambar bebas, dan avatar kehadiran. Label diberi tepi tipis warna kanvas supaya tetap terbaca di atas gambar. Label kursor pindah ke kiri panah di dekat tepi kanan dan naik di dekat tepi bawah, supaya tidak terpotong kanvas.
- Terhitung: teks label mode terang putih 5,48 sampai 7,10:1, mode gelap tinta 7,45 sampai 11,24:1; panah dan bingkai terhadap kanvas minimal 4,90:1 (terang) dan 8,70:1 (gelap).
- Test: `scripts/e2e/uji-kursor.mjs` membaca `getComputedStyle` (latar, teks, isian panah, warna bayangan bingkai, opacity efektif termasuk induk) untuk kedelapan indeks di tema terang dan gelap, di 360 dan 1280 px, lalu menghitung kontrasnya; juga label kursor di tepi kanan dan bawah. Gagal di kode sebelum perbaikan (71 dari 200 pemeriksaan), lolos sesudahnya.

## Sprint 12g: Task saya untuk peminta, edit To Do, dan papan task realtime (revisi 45 sampai 47)

**Tujuan:** tiga revisi papan task yang menyela Whiteboard W4. Kodenya sudah di-commit dan dirilis per revisi; bagian ini mencatat aturan yang diuji Sprint 13.

- [x] S12g-01 Revisi 45: "Task saya" (papan `scope=mine` dan angka kartu profil) memuat task yang dipegang sebagai PIC task, PIC sub task, atau diminta (`requested_by`), juga untuk Manajer dan kemunculan task berulang. Penanda "Kamu peminta" bila hanya peminta. Test `TestTaskSayaMemuatPeminta`, E2E `scripts/e2e/uji-r45.mjs`.
- [x] S12g-02 Revisi 46: ubah To Do di kartu profil lewat dobel klik atau dobel ketuk dengan kursor di posisi klik. E2E `scripts/e2e/uji-r46.mjs`, bagian todo `uji13.mjs` disesuaikan.
- [x] S12g-03 Revisi 47: papan task realtime lewat WebSocket `GET /api/tasks/ws`, tiga keputusan tambahan Manajer (lihat keputusan). Test `TestTaskLiveEvents`, tambahan Arsip dan `/tasks` di `TestTaskSayaMemuatPeminta`, E2E `scripts/e2e/uji-r47.mjs`, skenario papan task di `uji-restart.mjs`, `frontend/nginx.conf` mendapat `location = /api/tasks/ws`.

**Keputusan:**
- Revisi 45: satu baris per task walau cocok lewat beberapa jalur. Hak akses tidak berubah: peminta Staf tetap tidak bisa memindahkan task orang lain (`move_to` kosong, 403). Filter Orang tidak berubah: `pic=<nama>` hanya PIC task dan PIC sub task. Batas 10 kartu Completed tetap. `requester_only` dihitung server; penanda "Kamu peminta" kontras 15,23:1 (terang) dan 16,66:1 (gelap).
- Revisi 46: klik sekali tidak membuka isian (klik kedua dobel klik akan jatuh di isian dan menyeleksi kata); Enter atau Spasi di judul membuka isian dengan kursor di akhir. Enter dan blur menyimpan, Escape membatalkan dan fokus kembali ke judul. Isian textarea selebar baris yang tumbuh dan membungkus teks; baris baru di isian diganti spasi saat disimpan. Di layar sentuh posisi kursor dipasang ulang selama 600 ms karena gestur dobel ketuk bawaan menyeleksi satu kata.
- Revisi 47, event: setelah handler task menjawab 2xx (sesudah commit) server menyiarkan `{task_id, version, aksi, pelaku}` ke semua koneksi, tanpa isi task. Aksi: `baru` (buat, kemunculan dari ticker dengan pelaku "Sistem"), `ubah` (task, label, deliverable, hentikan seri, sub task, lampiran), `pindah`, `centang`, `hapus`. 4xx termasuk 409 tidak menghasilkan event. Semua yang login boleh membaca semua task, jadi siaran ke semua koneksi hanya membuka id task dan nama pelaku; akses tetap dicek saat klien mengambil ulang lewat REST.
- Revisi 47, klien: satu koneksi per tab dipakai bersama papan, Arsip, detail, dan dialog Pindahkan; ditutup sedetik setelah halaman terakhir pergi. Tab yang mengubah mengirim `X-Tab-Id` dan tidak memberi tahu dirinya sendiri. Papan dan Arsip mengambil ulang lewat REST, digabung 300 ms. Selama drag papan tidak dimuat ulang; setelah dilepas, kartu yang berubah tidak ikut dipindah dan toast "<ID> sudah dipindah <nama> ke <status>". Detail dan dialog Pindahkan menampilkan penanda dengan tombol Muat ulang, tanpa memuat ulang sendiri; catatan revisi di dialog tetap ada. Sambung ulang memuat ulang papan sekali; penanda "Terputus" kecil hanya saat koneksi putus.
- Revisi 47, 409: pesan menyebut orang yang lebih dulu mengubah dari baris audit terakhir task itu ("sudah diubah <nama> lebih dulu", atau "sudah kamu ubah di tab lain"), field `by` di galat. Papan dimuat ulang otomatis.
- Revisi 47, keputusan Manajer 1: klik angka di kartu profil membuka Task saya untuk Staf dan Manajer. Manajer mendapat saklar Task saya dan Semua task di papan dan Arsip; bawaannya Semua task, jadi `scope=mine` miliknya ditulis ke URL. Filter Orang nama sendiri di akun Manajer tetap filter Orang.
- Revisi 47, keputusan Manajer 2: Arsip `scope=mine` dan `GET /api/tasks?scope=mine` memakai aturan revisi 45 (PIC task, PIC sub task, atau peminta); penanda "Kamu peminta" juga di Arsip.
- Revisi 47, keputusan Manajer 3: kontrol To Do (naik, turun, hapus) hanya tampil di mode Atur, di baris sendiri di bawah teks. Ruang tetap di kanan tidak dipakai karena kartu 17rem tidak muat lima tombol 44 px. Ikon buka tautan di slot 44 px tetap di samping teks. Teks tidak pernah tertutup dan baris tidak bergeser saat hover.

**Tindak lanjut (belum dikerjakan):** realtime untuk Aktivitas Harian dan Timeline memakai koneksi yang sama (event jenis baru di hub `taskLive`). Arsip sudah ikut di revisi 47.

## Sprint 12: Persiapan deploy dan migrasi data

**Tujuan:** semua yang dibutuhkan untuk go-live sudah tertulis, tetapi belum dijalankan di VM.
**SRS:** FR-TASK-20, BR-09, NFR-SEC-07, 08, NFR-REL-01 sampai 05, NFR-RET-03.

Migrasi data:
- [x] S12-01 CLI impor Go dari sheet Task Tracker dan Completed Task di `NVM - Marketing & Sales.xlsx`. ID asli dipertahankan, `is_migrated = true`. Setiap baris Excel menjadi satu task tanpa sub task.
- [x] S12-02 Pemetaan nama di Request By dan Solved By ke username lewat file CSV. Nilai "Creative" dipetakan ke akun anggota divisi Creative.
- [x] S12-03 Task Completed di Excel tanpa Solved Date ditolak CHECK database. Putuskan saat dry-run: isi dari tanggal lain di baris itu, atau tandai untuk diisi manual.
- [x] S12-04 Mode `--dry-run` dengan laporan baris terbaca dan baris gagal beserta alasannya. Setelah impor, `setval` sequence ke ID terbesar.

Operasional (hanya ditulis):
- [x] S12-05 `Dockerfile` backend, `deploy/docker-compose.yml` (backend dan Nginx; PostgreSQL memakai container bersama di VM), dan `nginx.conf` dengan `client_max_body_size 51m` serta `proxy_set_header Host $host`.
- [x] S12-06 Skrip backup harian (database dan `/srv/nvm-internal/storage`), penyalinan ke hard disk eksternal, PC IT, dan laptop IT (salinan laptop dienkripsi).
- [x] S12-07 Pembersih berkas yatim (berkas di `attachments/` dan `avatars/` tanpa baris database) dan job harian penghapus audit log login yang lebih tua dari 1 tahun, serta baris `user_sessions` yang sudah kedaluwarsa (`created_at` lebih dari `auth.SessionMaxAge` lalu, 9 jam sejak revisi 8; baris yang sudah dicabut lewat `token_version` ikut terhapus karena tidak lagi diperbarui).
- [x] S12-08 Panduan satu halaman untuk tim (login, memindahkan task, sub task, unggah, profil) dan catatan operasional di README (backup, restore, update versi).
- [x] S12-09 Role database terpisah seperti di compro dan doomsday: `nvm_internal_migrator` untuk DDL (dipakai `migrate`) dan `nvm_internal_app` untuk aplikasi. Hak DELETE hanya untuk tabel yang memang dihapus permanen oleh kode. Hasil audit query DELETE di `backend/internal/db/queries` (2026-11-16): `task_subtasks` (`DeleteSubtasks`), `subtask_assignees` (`RemoveSubtaskAssignees`, juga lewat `ON DELETE CASCADE`), dan `task_labels` (`DeleteTaskLabels`). Task dan lampiran memakai `is_deleted`, jadi tidak butuh DELETE. Penghapus audit log login di S12-07 akan butuh DELETE di `audit_logs`; tambahkan saat job itu ditulis. Tabel `user_sessions` (migrasi 000006): role aplikasi butuh SELECT, INSERT, UPDATE, dan DELETE. Pencabutan sesi tidak memakai DELETE (lewat UPDATE `users.token_version`); DELETE dipakai job pembersih sesi kedaluwarsa di S12-07, yang berjalan di dalam backend dengan role aplikasi. Jadi daftar tabel dengan DELETE menjadi empat: `task_subtasks`, `subtask_assignees`, `task_labels`, dan `user_sessions` (ditambah `audit_logs` saat job audit S12-07 ditulis). Audit ulang daftar ini bila ada query DELETE baru.
- [x] S12-10 Binary backend dibangun statis (`CGO_ENABLED=0`) supaya jalan di image distroless. Di compro, binary yang tidak statis gagal jalan di server.
- [x] S12-11 Skrip backup memperhitungkan UID pemilik folder storage di container, karena di compro izin folder sempat membuat backup gagal diam-diam. Uji restore sebelum butir ini dianggap selesai.
- [x] S12-12 Cache: endpoint yang dibaca halaman pengelola (Pengguna, Pengaturan, Audit Log) tidak boleh tertahan cache di Nginx atau browser. Di compro, halaman admin sempat menampilkan data basi.

Revisi 5 (kode, dikerjakan 2026-11-16 di luar urutan karena dibutuhkan sebelum pemasangan dashboard):
- [x] S12-13 `GET /api/auth/verify` untuk Caddy forward_auth: 204 tanpa body bila sesi sah dan password sudah diganti, 401 bila tidak ada sesi atau sesi tidak sah, 403 bila password masih harus diganti. Selalu `Cache-Control: no-store`; `X-NVM-User` berisi username pada 204. Memakai `requireSession` dan `requirePasswordChanged` yang sudah ada. Uji: `scripts/uji-verify.sh`.
- [x] S12-14 Cookie sesi lewat env `COOKIE_DOMAIN` (kosong di dev) dan `COOKIE_SECURE` (wajib true bila `APP_ENV=production`, backend menolak start bila tidak). Atribut yang sama dipakai saat cookie dihapus (logout dan sesi berakhir).
- [x] S12-15 Pemeriksaan sameOrigin menerima origin produksi tanpa hardcode: host dari header `Origin` (atau `Referer`) dibandingkan dengan header `Host` permintaan. Syaratnya proxy meneruskan `Host` asli (`proxy_set_header Host $host` di S12-05, dan Cloudflare Tunnel tanpa `httpHostHeader`).
- [x] S12-16 `?kembali=` di halaman login dan setelah ganti password paksa hanya menerima path internal (diawali `/`, bukan `//` atau `/\`, dan bukan halaman login itu sendiri) atau URL `https` tanpa port dan userinfo yang host-nya tercantum di `VITE_ALLOWED_RETURN_HOSTS`. Selain itu diabaikan dan pengguna masuk ke halaman awal.
- [x] S12-17 Tombol Dashboard di top bar desktop dan menu Lainnya di ponsel, alamat dari `VITE_DASHBOARD_URL` (kosong berarti tidak tampil), buka di tab baru, untuk semua akun yang login.
- [x] S12-18 Idle timeout ditegakkan di server lewat tabel `user_sessions` (migrasi `000006`, index per akun di `000007`), supaya verify tidak perlu menerbitkan cookie. Cookie dan JWT diterbitkan sekali saat login dan ganti password, dengan umur mutlak 12 jam. Lihat Keputusan revisi 5.

**Butuh dari Manajer:** pemetaan nama di Excel ke username akun.

**Keputusan (revisi 5, 2026-11-16):**
- `/api/auth/verify` memakai `requireSession`, jadi setiap pemanggilan membaca ulang akun dari database: akun nonaktif dan logout langsung berlaku di dashboard.
- Sesi (diubah 2026-11-16, migrasi `000006_user_sessions`). Sebelumnya sesi berupa JWT tanpa catatan di server: setiap request yang lolos menerbitkan JWT baru (`iat` dan `exp` baru, jadi nilai token berganti) dengan cookie `Max-Age` 30 menit, dan idle timeout ditegakkan oleh `exp` di dalam token. Token lama tidak dicabut, tetap sah sampai `exp`-nya sendiri; pencabutan hanya lewat `token_version`. Karena Caddy forward_auth tidak meneruskan `Set-Cookie` dari jawaban 2xx, cara itu membuat pemakaian dashboard tidak pernah memperpanjang sesi. Sekarang:
  - Login dan ganti password membuat satu baris `user_sessions`, dan JWT membawa `sid`. Hanya dua titik itu yang menerbitkan cookie. Tidak ada lagi terbit ulang atau rotasi token di endpoint mana pun, termasuk verify.
  - Idle timeout 30 menit (`auth.IdleTimeout`) ditegakkan di server dari `last_seen_at`. Setiap request yang lolos, termasuk `/api/auth/verify`, memperbarui `last_seen_at` paling sering sekali per 60 detik (`sessionTouchEvery`), supaya permintaan dashboard yang beruntun tidak menulis ke database setiap kali. Pemakaian dashboard dihitung sebagai aktivitas.
  - JWT dan cookie memakai umur mutlak 12 jam sejak login (`auth.SessionMaxAge`), lebih panjang dari idle timeout, sehingga cookie tidak kedaluwarsa duluan selama sesi masih aktif di server. Setelah 12 jam pengguna login ulang walaupun terus aktif.
  - Waktu sesi diambil dari jam aplikasi, bukan `now()` database, supaya selisih jam dua mesin tidak memengaruhi idle.
  - Token tanpa `sid` (terbit sebelum migrasi) ditolak, jadi semua pengguna login ulang sekali setelah pemasangan.
  - Logout tetap menaikkan `token_version` (semua perangkat), tanpa menghapus baris sesi.
  - Yang disimpan di `user_sessions` hanya id sesi acak (UUID v4 dari `gen_random_uuid()`, 122 bit acak), `user_id`, `created_at`, dan `last_seen_at`. Tidak ada token mentah atau data yang bisa dipakai login: id sesi hanya berguna di dalam JWT yang ditandatangani `JWT_SECRET`, jadi bocornya isi tabel tidak cukup untuk masuk.
  - Batas mutlak 12 jam dicek dua kali di server: dari `exp` JWT (`auth.Parse`) dan dari `created_at` di database (`touchSession`). Cookie `Max-Age` sama dengan batas itu (12 jam), tidak lebih pendek.
  - Pencabutan memakai `token_version` di tabel `users` (UPDATE), bukan menghapus baris sesi. Yang menaikkannya: logout (`BumpTokenVersion`, berlaku untuk semua perangkat sekaligus, jadi "keluar dari semua perangkat" adalah logout biasa), ganti password (`ChangePassword`; perangkat yang mengganti langsung mendapat sesi baru), reset password oleh pengelola (`ResetUserPassword`), dan akun dinonaktifkan (`SetUserActive`, dan `requireSession` juga menolak akun nonaktif). Perubahan peran atau divisi (`UpdateUserProfile`) tidak menaikkannya, sama seperti sebelum revisi 5: hak akses dibaca ulang dari database di setiap request, jadi perubahan langsung berlaku tanpa login ulang.
  - Index: pencarian per sesi memakai primary key `id`. Pencarian per akun memakai `user_sessions_user_id_idx` di migrasi `000007`, dipisah dari `000006` karena `000006` sudah terpasang di database dev. `last_seen_at` ber-index untuk pembersihan.
- Tidak ada rate limiter umum. Limiter per IP (`limitLogin`, 10 percobaan lalu 1 per 30 detik) hanya dipasang di `POST /api/auth/login` dan `POST /api/auth/change-password`, jadi verify yang dipanggil Caddy di setiap permintaan dashboard tidak ikut tertahan. Bebannya dua query baca per permintaan dashboard (akun dan sesi) ditambah satu UPDATE paling sering sekali per 60 detik per sesi, dapat diterima untuk tim kecil.
- Cookie lama dengan Domain berbeda (mis. cookie dev tanpa Domain, lalu produksi memakai `novoidminds.xyz`) dianggap cookie lain oleh browser. Mengganti `COOKIE_DOMAIN` berarti pengguna perlu login ulang sekali.
- Env frontend memakai awalan `VITE_` (`VITE_DASHBOARD_URL`, `VITE_ALLOWED_RETURN_HOSTS`) karena Vite hanya menanam variabel berawalan itu ke bundle. Nilainya terlihat di browser, jadi tidak boleh berisi rahasia.
- Tujuan `?kembali=` yang berupa halaman login itu sendiri (`/`) diabaikan, supaya pengguna yang sudah login tidak berputar di halaman login.

**Keputusan (deploy dan impor, 2026-11-16):**
- Ditiru dari compro: role migrator dan app dengan `00-bootstrap.sql` yang meminta password interaktif dan menutup database dari role lain; migrasi lewat image `migrate/migrate` di jaringan Postgres; Nginx dari template dengan `NGINX_ENVSUBST_FILTER`; `set_real_ip_from` gateway jaringan Docker dengan `real_ip_header CF-Connecting-IP`; `X-Forwarded-For` ditimpa, bukan ditambah; header keamanan di berkas yang di-include per location; folder storage milik UID 65532; format satu baris per run di `backup.log`; cron dalam UTC.
- Ditiru dari doomsday: healthcheck `wget /api/health` di container Nginx (backend distroless tidak punya shell); batas log Docker 10 MB x 3; `resolver 127.0.0.11` dengan variabel backend; pemeriksaan `CGO_ENABLED=0` lewat `go version -m` saat build; `pg_restore --list` untuk membuktikan dump bisa dibaca; `trap ERR` di skrip backup.
- Sengaja berbeda dari keduanya:
  - Port Nginx diikat ke `127.0.0.1:8094`, bukan semua antarmuka (S14-03). Subnet `172.30.94.0/24` dan port 8094 meneruskan urutan compro (`.92`, 8092) dan doomsday (`.93`, 8090).
  - Hak DELETE tidak diberikan lewat default privileges, tetapi per tabel di `deploy/sql/10-grants.sql` (dijalankan ulang setelah migrasi yang menambah tabel), sesuai audit S12-09. compro memberi DELETE ke semua tabel.
  - CSP lebih ketat: `style-src 'self'` tanpa `unsafe-inline`. Hasil pemeriksaan build sungguhan: `index.html` tanpa script atau style inline, bundle tanpa `eval`, font dan foto dari origin sendiri, dan binding `:style` Vue memakai CSSOM yang tidak diatur `style-src`. Belum diuji di browser; cek galat CSP di console masuk daftar cek runbook dan S13-09.
  - Backup storage dibuat di container `alpine` yang berjalan sebagai UID 65532 dengan mount baca-saja, karena backend menulis berkas 0600 dan folder 0750. compro memilih mengubah izin berkas (`chmod a+rX`).
  - Rotasi 14 harian dan 8 mingguan (salinan hari Minggu lewat hard link); compro dan doomsday hanya 14 harian.
  - Satu image backend berisi `server` (dengan subperintah `maintenance`), `seed`, dan `import-excel`. compro dan doomsday memakai binary admin terpisah.
- Backend memercayai `X-Forwarded-For` dan `X-Real-IP` hanya dari `TRUSTED_PROXIES` (subnet Nginx di produksi, kosong di dev). Limiter login dan kolom `ip_address` audit log memakai `ClientIP` hasil itu. Uji di dev: `scripts/uji-ip.sh`.
- Semua jawaban `/api` memakai `Cache-Control: no-store` lewat middleware grup `/api`; foto profil menimpanya dengan `immutable` bila `?v=` cocok. Unduhan lampiran diubah dari `private, no-cache` ke `private, no-store`.
- Password minimal 12 karakter (sebelumnya 8) untuk akun baru, password sementara dari reset, ganti password sendiri, dan `seed`. Password lama tetap berlaku sampai diganti.
- Job harian `server maintenance [--dry-run]` dijalankan cron host lewat `docker compose run --rm`: menghapus sesi kedaluwarsa, audit log `login_success`, `login_failed`, `account_locked`, dan `logout` yang lebih tua dari 1 tahun (perubahan password dan akun tidak termasuk), serta berkas yatim di `attachments/` dan `avatars/` (termasuk sisa `.upload-*`). Berkas yang lebih muda dari 1 jam tidak disentuh, karena lampiran sudah ada di disk sebelum transaksinya selesai. Berkas lampiran yang `is_deleted` bukan yatim dan tetap disimpan. Nama berkas yang bukan bentuk nama aplikasi hanya dilaporkan.
- Impor Excel:
  - Berkas `.xlsx` dibaca dengan pustaka standar (zip dan XML), tanpa dependensi baru. Kolom dicari dari baris header berisi ID Task dan Judul Task (SRS 2.1, sembilan kolom), bukan dari posisi.
  - Satu baris menjadi satu task tanpa sub task (S12-01), jadi tanpa PIC. Solved By hanya menentukan divisi task (divisi akun hasil pemetaan) dan dicatat di deskripsi ("Solved By: ..."). Request By menjadi `requested_by`.
  - Ditolak dan dilaporkan, tidak diperbaiki diam-diam: ID bukan format T-nnnn atau kembar di berkas atau sudah ada di database, judul kosong atau lebih dari 200 karakter, status atau urgency di luar daftar, nama yang belum dipetakan, tanggal yang tidak terbaca, Completed tanpa Solved Date (S12-03), dan Solved Date pada status selain Completed.
  - Deliverable yang bukan tautan `http(s)` dipindah ke deskripsi (constraint menolaknya sebagai tautan). Status kosong di sheet Completed Task dianggap Completed, dengan catatan di laporan.
  - Tanggal dibaca sebagai hari WIB: nomor seri Excel, `dd/mm/yyyy`, atau `yyyy-mm-dd`.
  - Impor sungguhan menolak jalan bila masih ada baris yang ditolak, dan semua task ditulis dalam satu transaksi beserta audit `task_created` (pelaku "Sistem"), lalu `task_number_seq` disetel ke ID terbesar tanpa pernah mundur. Impor memakai role migrator karena `setval` tidak diizinkan untuk role aplikasi.

**Butuh dari Manajer (tambahan):**
- Isi CSV pemetaan `deploy/import/pemetaan-nama.example.csv` (nama di Request By dan Solved By, termasuk "Creative", ke username).
- Task hasil migrasi tanpa PIC tidak tampil di "Task saya" Staf. Setuju, atau Solved By dijadikan satu sub task dengan PIC?
- Task Completed tanpa Solved Date (S12-03): isi dari Request Date, atau diisi manual sebelum impor?

**Hasil:** kode dan berkas Sprint 12 selesai 2026-11-16; belum ada yang dijalankan di VM. `./scripts/check.sh` lolos. Pembaca `.xlsx` dan aturan baris diuji lewat test sementara (dihapus lagi): teks bersama dan inline, header yang tidak di baris pertama, nomor seri tanggal, penolakan urgency dan tanggal, pemetaan nama, normalisasi ID. Uji dengan berkas Excel sungguhan menunggu berkasnya (tidak ada di `~/Documents/NVM`). Docker tidak ada di PC kantor, jadi build image, Nginx, CSP di browser, dan healthcheck pertama kali berjalan di VM (runbook langkah 4 dan 7). Uji API setelah restart backend: lihat laporan Sprint 12.

**Retro:**
- Macet: (isi sendiri)
- Lancar: (isi sendiri)
- Diubah untuk Sprint 13: (isi sendiri)

---

## Sprint 13: Pengujian

**Tujuan:** semua yang dibangun di Sprint 7 sampai 12 teruji, dikerjakan Claude Code secara otomatis, lalu direview.
**Aturan:** Claude Code tidak boleh commit dan push. Commit dan push dijalankan Rifky.

- [x] S13-01 Kunci Claude Code lewat `.claude/settings.json` (dimajukan ke 2026-11-13, lihat Pembagian kerja).
- [x] S13-02 Tulis `CLAUDE.md` (dimajukan ke 2026-11-13). Perbarui bila ada aturan baru saat Sprint 13.
- [ ] S13-03 Prompt Claude Code: perbaiki test lama yang rusak karena Sprint 7, lalu tulis test backend untuk izin, transisi status, sub task dan PIC turunan, profil dan foto, arsip dan pencarian, lampiran, centang sub task beserta syarat Ready.
- [ ] S13-04 Test frontend untuk komponen dengan logika (pemilih PIC ganda, filter di URL, kartu profil).
- [ ] S13-05 Peta AC-01 sampai AC-15 ke test atau langkah uji manual, dengan hasil PASS atau FAIL.
- [ ] S13-06 Data dummy sekitar 2.000 task, ukur p95 dari dalam VM (bukan dari PC), cek `EXPLAIN ANALYZE` bahwa index pencarian terpakai. Target p95 di bawah 1 detik. Blok kata kunci di `ListArchiveTasks` dan `ListBoardTasks` memakai `OR` dengan banyak `EXISTS`, yang kemungkinan membuat index trigram tidak terpakai. Kalau lambat atau tidak memakai index, tulis ulang jadi `t.id IN (SELECT ... UNION SELECT ...)` di kedua query sekaligus. Sejak revisi 15 daftar `EXPLAIN ANALYZE` juga memuat filter PIC (`pic=<username>`) di papan dan Arsip, karena kondisinya kini dua EXISTS (PIC task dan PIC sub task), serta `ListSubtaskPICMatches`.
- [ ] S13-07 Review hasil bersama, perbaiki temuan, commit.
- [ ] S13-08 Uji yang tidak bisa otomatis: ponsel sungguhan (kamera atau galeri, Safari, tempel gambar), pembaca layar, cek keyboard saja.
- [ ] S13-09 Audit visual dan aksesibilitas hasil Sprint 10, Sprint 11, dan tombol Dashboard (dipindah dari S11-07), dengan Playwright dan `VITE_DASHBOARD_URL` terisi: `document.documentElement.scrollWidth <= window.innerWidth` di 360, 768, 1280, dan 1920 px untuk Admin, Manajer, dan Staf di semua halaman (perhatikan top bar 768 px: tombol Dashboard menambah sekitar 60 px), Tab dan Escape di panel, dialog, dan menu Lainnya, emulasi `prefers-reduced-motion` (tidak ada transisi), kontras dihitung, galat console 0, dan screenshot untuk dibandingkan dengan versi sebelum Sprint 11 (commit `8b36389`).
- [ ] S13-10 Test otomatis model sesi (revisi 5 dan 8, migrasi 000006 dan 000007): tabel hanya menyimpan id acak dan waktu (tanpa token); batas mutlak 9 jam ditolak dari `exp` JWT dan dari `created_at`; cookie `Max-Age` 9 jam; sesi yang lama tidak dipakai tetap sah selama belum 9 jam (idle timeout sudah dihapus); password 6 karakter diterima, 5 ditolak, dan kuncian akun setelah gagal berulang tetap jalan; verify tanpa `Set-Cookie` dan tidak merotasi token; `last_seen_at` hanya ditulis bila aktivitas terakhir lebih dari 60 detik lalu (verify beruntun tidak menulis); token tanpa `sid` atau dengan `sid` akun lain ditolak; pencabutan setelah logout (semua perangkat), ganti password (perangkat lain 401, perangkat yang mengganti 204), reset password, dan nonaktif; perubahan peran tidak memaksa login ulang. `scripts/uji-verify.sh` sudah mencakup sebagian lewat API.

---

## Sprint 14: Deploy, UAT, dan go-live

**Tujuan:** tim memakai sistem ini, bukan Excel.

Deploy (dijalankan Rifky di VM):
- [x] S14-01 Deploy ke VM `nvm-server` 192.168.68.69 di `/srv/nvm-internal`. Cek dulu lokasi disk data untuk `STORAGE_DIR` di VM baru; mount hard disk baru ke `/srv/nvm-internal/storage` bila sudah ada.
- [x] S14-02 Jalankan `deploy/docker-compose.yml`; container `restart: unless-stopped`. Uji VM dan container menyala sendiri setelah Windows restart dipindah ke S14-18.
- [ ] S14-03 Port database tidak dibuka keluar VM; firewall hanya izinkan port 80 dari subnet kantor. Pengikatan port Postgres, Redis, dan RabbitMQ ke 127.0.0.1 masih tertunda sejak Oktober: cek keadaannya di VM baru dan selesaikan sebelum go-live. Sejak revisi 5 akses publik lewat Cloudflare Tunnel (koneksi keluar dari VM), jadi tidak ada port masuk yang perlu dibuka untuk publik; tinjau ulang aturan port 80 subnet kantor saat pemasangan.
- [x] S14-04 Backup harian berjalan, dan uji restore pertama ke database kosong dicatat waktunya (target di bawah 4 jam).
- [ ] S14-05 Diagnosa jaringan PC tim 192.168.68.68, host Hyper-V untuk VM baru (`Get-NetAdapter`, `Get-VMSwitch`, ping ke gateway), dan catat hasilnya.
- [ ] S14-10 Aktifkan GitHub secret scanning, push protection, dan Dependabot untuk repo ini, lalu pilah peringatannya menjadi runtime dan dev-only (seperti di compro).
- [x] S14-11 HTTPS, lalu `COOKIE_SECURE=true` dan `COOKIE_DOMAIN=novoidminds.xyz` (dipindah dari "Setelah go-live" karena revisi 5 membuat sistem publik).
- [x] S14-12 `novoidminds.xyz` sebagai public hostname di tunnel `nvm-server`.
- [x] S14-13 Tunnel baru di PC tim 192.168.68.68 untuk `dash.novoidminds.xyz` (NVM Launcher) dan `studio.novoidminds.xyz` (Remotion Studio), keduanya lewat Caddy dengan forward_auth ke `https://novoidminds.xyz/api/auth/verify`. Saat 401 atau 403, arahkan ke `https://novoidminds.xyz/?kembali=<URL asli>`. Isi `VITE_DASHBOARD_URL` dan `VITE_ALLOWED_RETURN_HOSTS` saat build frontend produksi.
- [x] S14-14 Caddy dan NVM Launcher menyala otomatis di PC tim.
- [x] S14-15 Rate limit Cloudflare untuk `POST /api/auth/login`, di atas limiter per IP di backend (backend hanya melihat IP Cloudflare atau proxy, kecuali header IP asli dipercaya).
- [ ] S14-16 Password awal akun tim kuat (bukan pola mudah ditebak), karena halaman login sekarang terbuka ke internet.
- [ ] S14-17 Perbarui SRS: sistem tidak lagi LAN-only (revisi 5).
- [ ] S14-18 Uji restart Windows di PC tim 192.168.68.68: VM `nvm-server`, container NVM-Internal, compro, doomsday, Postgres, tunnel `nvm-server` dan `nvm-tim-pc`, Caddy, dan NVM Launcher menyala sendiri tanpa login manual, lalu `https://novoidminds.xyz` dan `https://dash.novoidminds.xyz` terbuka.
- [ ] S14-19 Salinan backup ke luar VM: hard disk eksternal, PC IT, dan laptop (terenkripsi), sesuai runbook bagian 8. Tentukan jadwal dan siapa yang menjalankan.
- [ ] S14-20 Ulangi uji restore setelah ada data sungguhan (akun tim, task, lampiran, foto), catat waktunya. Uji pertama 3 detik dengan data hampir kosong.

UAT:
- [ ] S14-06 Checklist AC-01 sampai AC-15 bersama Manajer, hasil PASS atau FAIL.
- [ ] S14-07 Perbaiki temuan UAT. Temuan yang bukan bug masuk backlog setelah go-live.
- [ ] S14-08 Migrasi final dari Excel versi terakhir, lalu Excel dibekukan (hanya baca).
- [ ] S14-09 Buat akun semua anggota tim, bagikan username dan password awal secara langsung.

**Selesai kalau:** semua AC PASS dan tim sudah memakai sistem minimal satu minggu tanpa kembali ke Excel.

**Hasil (deploy 2026-11-17):** NVM-Internal jalan di VM `nvm-server` 192.168.68.69 (`/srv/nvm-internal`) dan terbuka di `https://novoidminds.xyz`. Selesai:
- Deploy, migrasi sampai `000007`, dan akun Admin pertama `adminit`.
- Tunnel `nvm-server` dengan route `novoidminds.xyz` ke `127.0.0.1:8094`.
- HTTPS (Always Use HTTPS, HSTS 6 bulan) dan cookie Secure dengan Domain `novoidminds.xyz`.
- Rate limit Cloudflare untuk `POST /api/auth/login`: 5 per 10 detik, blok 10 detik (paket Free).
- Tunnel `nvm-tim-pc` di PC tim dengan route `dash.novoidminds.xyz` dan `studio.novoidminds.xyz` ke Caddy di `127.0.0.1:8088`. Caddy dan NVM Launcher menyala otomatis.
- Backup harian. Uji restore pertama selesai dalam 3 detik dengan data hampir kosong; diulang setelah ada data sungguhan (S14-20).

Masih terbuka: uji restart Windows di PC tim (S14-18), salinan backup ke luar VM (S14-19), uji restore ulang (S14-20), port database dan firewall (S14-03), diagnosa jaringan (S14-05), GitHub secret scanning dan Dependabot (S14-10), akun tim dengan password awal kuat (S14-09, S14-16), SRS (S14-17), impor Excel menunggu pemetaan dan keputusan Manajer (S14-08, butir "Butuh dari Manajer" Sprint 12), UAT (S14-06, S14-07), dan seluruh Sprint 13.

**Catatan deploy (sudah dimasukkan ke `docs/runbook-deploy.md`):**
- Migrate gagal "permission denied for schema public": template database di cluster ini hasil restore, jadi skema `public` di database baru tidak otomatis milik pemilik database. `00-bootstrap.sql` menjalankan `ALTER SCHEMA public OWNER TO nvm_internal_migrator` dan `REVOKE CREATE ON SCHEMA public FROM PUBLIC` di dalam `nvm_internal_prod`, lalu mencetak pemilik skema sebagai bukti.
- Variabel yang sudah di-export di shell mengalahkan `--env-file`, dan compose tidak membuat ulang container hanya karena isi env berubah. Setelah `deploy/.env` diubah: source ulang atau buka sesi SSH baru, lalu `up -d --force-recreate`.
- Sebelum migrasi: cek panjang `JWT_SECRET` (64) dan user di `MIGRATOR_URL` dan `DATABASE_URL` dengan password disamarkan.
- `git clone` gagal bila `/srv/nvm-internal` sudah ada; periksa isinya dulu, folder kosong dihapus dengan `rmdir`.

---

## Daftar revisi

| No | Tanggal | Isi | Masuk ke |
|---|---|---|---|
| 1 | 2026-11-09 | Task punya sub task; PIC dipilih per sub task, bisa lebih dari satu | Sprint 7 dan 8 |
| 2 | 2026-11-09 | Area kiri papan diisi kartu profil (foto, nama, jabatan, divisi, bisa diubah); poles UI/UX papan dan semua halaman | Sprint 10 dan 11 |
| 3 | 2026-11-09 | Sub task bisa dicentang oleh PIC-nya saat On Progress; Ready butuh semua sub task tercentang; task penuh bisa dibuka di tab baru; di form, Deskripsi menggantikan posisi Lampiran brief, dan Lampiran brief masuk ke Detail tambahan | Sprint 8 |
| 4 | 2026-11-10 | Manajer boleh mencentang dan melepas centang sub task siapa pun, kecuali sub task yang dipegang Manajer lain | Sprint 8 |
| 6 | 2026-11-17 | PIC kembali dipilih di task induk oleh Manajer. Sub task disusun sendiri oleh PIC yang mengerjakannya; Manajer juga boleh menambah sub task | Sprint 11d |
| 7 | 2026-11-17 | Sub task bisa ditugaskan ke PIC task yang lain, bukan hanya ke diri sendiri. Satu sub task satu PIC, dan PIC itu yang mencentang | Sprint 11e |
| 8 | 2026-11-17 | Sub task menampilkan pembuat dan PIC-nya, serta punya deskripsi dan lampiran sendiri. Password minimal 6 karakter dan sesi paling lama 9 jam | Sprint 11f |
| 9 | 2026-11-17 | Sub task punya tautan deliverable sendiri, bukan hanya berkas. Yang boleh mencentang: PIC sub task, yang membuatnya, dan Manajer | Sprint 11g |
| 10 | 2026-11-17 | Tautan sudah bisa diisi saat sub task dibuat. Sub task punya dua tautan: acuan dari yang menugaskan dan hasil dari yang mengerjakan. Berkas sub task bisa dipratinjau di web | Sprint 11h |
| 11 | 2026-11-17 | PIC sub task boleh siapa saja yang akunnya aktif, bukan hanya PIC task. Yang membuat sub task tetap PIC task pilihan Manajer | Sprint 11i |
| 12 | 2026-11-17 | Manajer tidak bisa ditugaskan di sub task. Dia tetap bisa membuat sub task dan menugaskannya ke orang lain | Sprint 11j |
| 13 | 2026-11-17 | Revisi 12 dibatalkan. Manajer kembali boleh menjadi PIC sub task, dan tetap boleh membuat sub task serta menugaskan orang, sama seperti PIC task | Sprint 11k |
| 14 | 2026-11-17 | Popover profil: klik foto atau nama orang membuka kartu kecil berisi foto, nama, divisi, jabatan, peran, dan tautan ke task-nya di papan | Sprint 11l |
| 15 | 2026-11-18 | Filter PIC, Arsip, dan Task saya ikut menghitung PIC sub task. Mengganti keputusan revisi 11 soal Task saya | Sprint 11m |
| 16 | 2026-11-18 | Sub task punya tenggat yang tidak boleh melewati tenggat task. Form Buat task: Judul lalu Tenggat, dan sub task diisi lengkap dengan form yang sama dengan Tambah sub task | Sprint 11n |
| 17 | 2026-11-18 | PIC sub task boleh memindahkan task ke On Progress dan ke Ready (dengan Ready gate). Form tambah sub task di panel detail dilipat di balik satu tombol | Sprint 11o |
| 18 | 2026-11-18 | PIC sub task menulis catatan hasil (output di mana, dikirim ke mana, versi mana). Detail sub task dibagi Arahan dan Hasil | Sprint 11p |
| 19 | 2026-11-18 | Task berulang harian, mingguan, atau bulanan. Setiap kemunculan task biasa dengan ID sendiri, dibuat otomatis pada harinya | Sprint 11q |
| 20 | 2026-11-18 | Papan punya filter deadline "Hari ini". Kata "Tenggat" diganti "Deadline" di seluruh teks pengguna | Sprint 11r |
| 21 | 2026-11-18 | Aktivitas Harian, pengganti spreadsheet "Daily Activity Monitoring": Staf mengisi aktivitas dan output per jam, Manajer melihat semua orang per tanggal dan bisa mengekspor ke Excel | Sprint 11s |
| 22 | 2026-11-18 | Kartu selalu menampilkan PIC sub task dalam kelompok tersendiri. PIC sub task boleh mengembalikan task Ready ke On Progress dengan membuka lagi sub task miliknya yang dipilih, alasan opsional | Sprint 11t |
| 23 | 2026-11-18 | Halaman Status hanya untuk Admin. Peran lain tidak melihat menunya, dan membuka URL-nya langsung menampilkan 404 | Sprint 11u |
| 24 | 2026-11-18 | Lampiran di Aktivitas Harian: berkas menempel pada satu jam, maksimal 95 MB dan 10 berkas per jam, dengan gambar, video, PDF, dokumen Office, TXT, CSV, dan ZIP. Bisa ditarik, ditempel dari papan ketik, dipratinjau, dan dihapus | Sprint 11v |
| 25 | 2026-11-18 | Salinan rapi semua berkas ke Google Drive akun tim setiap malam: Aktivitas per bulan, hari, dan orang; Task per bulan dengan Brief, Deliverable, dan sub task; dump database terenkripsi age. Satu arah dan tidak menghapus | Sprint 11w |
| 26 | 2026-11-18 | Panel Buat task: batal memilih berkas tidak lagi menutup panel, isi panel tidak melompat saat pilih berkas diklik, dan form Buat task bisa dibuka di tab baru sebagai halaman /task/baru | Sprint 11x |
| 27 | 2026-11-19 | Staf dan Manajer boleh membuat task dengan aturan yang sama persis. Admin tidak. Pembuat task boleh mengubah dan menghapus task buatannya selama masih Requested | Sprint 11y |
| 28 | 2026-11-19 | To Do pribadi di papan: daftar catatan per akun, hanya bisa dibaca dan diubah pemiliknya, dengan centang, ubah di tempat, urutkan, hapus dengan Urungkan, dan bersihkan yang selesai | Sprint 11y |
| 29 | 2026-11-19 | Sub task di kemunculan terbaru task berulang otomatis masuk templat seri: tambah, ubah, hapus, dan urutkan sub task langsung dipakai kemunculan berikutnya; kemunculan lama dan seri yang dihentikan tidak | Sprint 11z |
| 30 | 2026-11-19 | Halaman Timeline: kalender bulan bersama untuk launching, produksi, dan kehadiran, plus Important Notes. Manajer menulis, peran lain membaca; alasan kehadiran pribadi hanya untuk Manajer dan orangnya | Sprint 12a |
| 31 | 2026-11-20 | Menghapus task berulang tidak menghentikan serinya: dialog hapus memilih hapus kemunculan saja atau hapus dan hentikan pengulangan (Manajer), pengaman ticker untuk seri tanpa kemunculan tersisa, status seri di detail | Sprint 12b |
| 32 | 2026-11-20 | PIC task mengubah dan menghapus semua sub task di task-nya, termasuk sub task templat dari pembuat seri; centang tidak berubah | Sprint 12b |
| 33 | 2026-11-20 | Rantai sambung jam di Aktivitas Harian: jam berikutnya yang kosong bisa disambung ke jam sebelumnya dan menampilkan isinya secara readonly, termasuk melintasi jam istirahat; halaman Manajer dan ekspor menampilkan rantai sebagai satu rentang | Sprint 12c |
| 34 | 2026-11-23 | Sub task bisa dicentang dan dibatalkan PIC sub task, PIC task, pembuat task, dan Manajer saat On Progress; PIC task dan Manajer boleh membuka lagi sub task pilihan saat mengembalikan Ready ke On Progress | Sprint 12d |
| 35 | 2026-11-23 | Output Aktivitas dengan beberapa tautan: setiap URL jadi tombol berlabel, dialog detail tanpa gulir horizontal | Sprint 12d |
| 36 | 2026-11-23 | Tempel (Ctrl+V) dan seret-lepas berkas di lampiran Arahan dan Hasil sub task serta lampiran task, dengan antrean berprogres dan coba lagi | Sprint 12d |
| 37 | 2026-11-24 | Staf mengisi Aktivitas Harian sampai 2 hari kerja ke belakang, kartu di papan bisa mundur tanggal, ekspor Drive ikut menulis ulang bulan lalu | Sprint 12e |
| 38 | 2026-11-24 | Akses tambahan per pengguna: Kelola Timeline dan Kelola Important Notes, diatur di halaman Pengguna; alasan Kehadiran terbuka untuk semua pengguna | Sprint 12e |
| 39 | 2026-11-24 | Urutan navbar Timeline, Task, Aktivitas, Arsip; Timeline jadi halaman awal semua peran dan tujuan logo | Sprint 12e |
| 40 | 2026-11-24 | Kolom Completed di papan task paling banyak 10 task terbaru, dengan total dan tautan ke Arsip | Sprint 12e |
| 41 | 2026-11-24 | Whiteboard setara Milanote atau Canva: papan akar bersama dan papan bersarang sebagai kartu, kartu, kolaborasi langsung, Tempat sampah, template, ekspor (tahap W1 sampai W4, `docs/whiteboard.md`) | Sprint 12f |
| 42 | 2026-11-25 | Whiteboard W3 revisi A: gambar bebas tanpa kartu (mode pena di kanvas, objek gambar di atas kartu) | Sprint 12f W3 |
| 43 | 2026-11-25 | Whiteboard W3 revisi B: kartu Tabel punya judul | Sprint 12f W3 |
| 44 | 2026-11-25 | Whiteboard revisi C: bentuk dan diagram (tahap W5), model garis dirancang di W3 | Sprint 12f W3 dan W5 |
| 45 | 2026-11-25 | "Task saya" dan angka kartu profil memuat task yang diminta pengguna, dengan penanda "Kamu peminta" | Sprint 12g |
| 46 | 2026-11-25 | Ubah To Do lewat dobel klik dengan kursor di posisi klik, isian selebar baris, mode Atur tidak menyempitkan teks | Sprint 12g |
| 47 | 2026-11-25 | Papan task realtime lewat WebSocket bersama, Task saya untuk Manajer dan Arsip, kontrol To Do hanya di mode Atur | Sprint 12g |
| 33b | 2026-11-20 | Poles tampilan kartu Aktivitas: tombol rantai pindah ke celah antar isian dan jadi ikon kecil tanpa bingkai, tombol mati tidak dirender, jam tersambung menyatu dengan jam kepalanya, status simpan punya baris tetap, Tambah berkas di sel tabel tanpa bingkai | Sprint 12c |
| 5 | 2026-11-16 | NVM-Internal publik di `https://novoidminds.xyz`, bukan LAN-only. NVM Launcher (`https://dash.novoidminds.xyz`) dan Remotion Studio (`https://studio.novoidminds.xyz`) di PC tim dijaga Caddy forward_auth ke NVM-Internal, jadi hanya akun NVM-Internal yang sudah login yang bisa masuk | Sprint 12 (kode) dan 14 (pemasangan) |

SRS v1.0 belum memuat keempat puluh tujuh revisi (revisi 5 di S14-17). SRS diperbarui sebelum UAT supaya checklist AC sesuai aplikasi.

## Menunggu jawaban Manajer

- Umur mutlak sesi 12 jam sejak login (revisi 5): setelah itu pengguna login ulang walaupun terus aktif. Idle timeout saat itu masih 30 menit. Keduanya diubah di revisi 8: umur sesi jadi 9 jam dan idle timeout dihapus.
- Daftar label awal (nama dan warna) dan daftar pengguna awal (nama, username, peran, divisi).
- Apakah tim masih memakai berkas .ppt lama. Kalau tidak, hapus dari daftar izin.
- Konfirmasi kolom Completed di papan hanya 7 hari terakhir (AC-08 menyebut task Completed hilang dari papan aktif).
- Jabatan diisi sendiri oleh tiap orang atau hanya pengelola.
- Aturan centang: Manajer boleh mencentang sub task siapa pun kecuali yang dipegang Manajer lain. Sejak revisi 6 sub task tanpa pemilik (buatan Manajer) juga boleh dicentang PIC task mana pun. Setuju?
- Revisi 6: apakah PIC boleh menandai task Ready tanpa membuat sub task sama sekali? Sekarang boleh, karena syarat Ready hanya "tidak ada sub task yang belum tercentang".
- Revisi 32, mengurutkan sub task: belum ada pengurutan sub task di detail task, hanya di form Ubah task (Manajer dan pembuat selama Requested). Apakah PIC task perlu tombol naik dan turun di daftar sub task? Sementara tidak ditambah.
- Revisi 33: rantai ditolak bila jam sebelumnya kosong ("Isi jam sebelumnya dulu untuk menyambung."), supaya jam kosong tidak terhitung terisi. Ini tambahan dari kami, tidak ada di permintaan. Setuju? Tombol rantai juga hanya ada di kartu papan task, belum di dialog sel halaman Aktivitas untuk Manajer.

- Revisi 41 Whiteboard, sepuluh pertanyaan di `docs/whiteboard.md` bagian 10. Bawaan dipakai, menunggu konfirmasi:
  1. Komentar: penulis mengubah dan menghapus komentarnya sendiri, Manajer bisa menghapus komentar siapa pun.
  2. Berkas Whiteboard tidak ikut salinan Google Drive malam.
  3. Riwayat aktivitas papan disimpan tanpa batas sampai diputuskan.
  4. Template boleh dibuat dan diubah semua Staf dan Manajer.
  5. Admin hanya baca, termasuk tidak bisa memulihkan dari Tempat sampah.
  6. Tidak ada papan pribadi atau per divisi; semua melihat semua.
  7. Pratinjau tautan boleh mengambil halaman situs luar dari server kantor, dengan batas keamanan di `docs/whiteboard.md` bagian 5.
  8. Tempat sampah 30 hari, tanpa tombol hapus permanen.
  9. Tanpa sebutan (@nama) dan notifikasi di komentar.
  10. Kunci lunak (satu orang mengetik per kartu, yang lain melihat teksnya berubah langsung), bukan mengetik bersamaan seperti Google Docs.
- Whiteboard W3, bawaan dipakai, menunggu konfirmasi:
  1. Pratinjau tautan paling banyak 30 kali per orang per jam; kartu tautan tetap dibuat tanpa pratinjau di atas batas itu.
  2. Jenis berkas Whiteboard: daftar lampiran task ditambah WebP dan GIF, tanpa SVG, ZIP, dan video.
  3. Gambar bebas paling banyak 300 goresan dan 6.000 titik per kartu (rancangan menyebut 500 goresan).
  4. Garis yang terputus karena salah satu kartunya dipindah ke papan lain masuk Tempat sampah, bukan ikut pindah.
- Whiteboard W4, bawaan dipakai, menunggu konfirmasi:
  1. Ekspor PNG dan PDF selalu memakai warna mode terang, dan teks catatan diekspor tanpa format huruf (tebal, miring, tautan).
  2. Salinan template memakai berkas gambar yang sama dengan templatenya, bukan salinan berkas; menghapus template tidak merusak salinannya.
  3. Di bawah zoom 40% kartu hanya menampilkan judul ringkas (level of detail).
- Whiteboard W5, bawaan dipakai, menunggu konfirmasi:
  1. Warna isian bentuk memakai warna kartu yang sudah ada (tanpa palet warna bebas), garisnya memakai warna pena.
  2. Bentuk dan garis memakai jalur siku otomatis yang tidak menembus bentuk asal dan tujuan, tetapi bisa melewati bentuk lain di tengah jalan; titik belok dipakai untuk memutarinya.
  3. Ujung garis bebas (tidak menempel kartu) diatur dari titik yang ditarik; belum ada pegangan untuk menyeretnya setelah dibuat.
- Revisi 47: event papan task (id task dan nama pelaku, tanpa isi) disiarkan ke semua yang login, karena semua peran boleh membaca semua task. Setuju, atau perlu dibatasi per divisi kalau aturan baca berubah?

## Setelah go-live (belum dijadwalkan)

- Fitur ekspor untuk Manajer (BR-11, belum ada di versi ini).
- Uji restore rutin tiap 3 bulan (NFR-REL-04).
- Upgrade npm 12 (ditunda).