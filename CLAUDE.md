# NVM Internal

Sistem internal NVM untuk tim marketing: autentikasi, akun dan divisi, task tracker dengan papan Kanban, sub task, lampiran, arsip, audit log. Dipakai di LAN kantor, dihosting di VM Ubuntu pada PC kantor.

Sumber kebenaran urutan kerja, aturan bisnis, keputusan, dan revisi: `docs/sprint.md`. Baca bagian sprint yang sedang dikerjakan sebelum menulis kode. ID kebutuhan (FR-..., NFR-..., BR-..., AC-...) merujuk ke SRS v1.0 yang tidak ada di repo; kutipan yang perlu sudah ada di `docs/sprint.md`.

## Stack

- Backend: Go 1.27, Gin, pgx/v5, sqlc 1.31.1 (membaca skema dari `backend/migrations`), golang-migrate.
- Database: PostgreSQL 18 di Docker pada VM. Dari laptop diakses lewat tunnel SSH di `127.0.0.1:15432`. Database dev `nvm_internal`, database test `nvm_internal_test`.
- Frontend: Vue 3.5, TypeScript, Vite, Tailwind CSS v4, Pinia. ESLint dan Prettier (tanpa titik koma, kutip tunggal, printWidth 110).
- Laptop pengembang Windows. Terminal utama MSYS2 UCRT64 bash. Python dipanggil `py -3` atau `python`, bukan `python3`.

## Peta repo

- `backend/migrations`: migrasi berurutan (`000001` sampai `000004` sudah ada). Migrasi baru memakai nomor berikutnya, selalu dengan `up` dan `down`.
- `backend/internal/db/queries/*.sql`: query sqlc. `backend/internal/db/sqlc`: hasil generate, jangan diedit tangan.
- `backend/internal/task`: aturan bisnis task (service, sub task, centang, papan, arsip).
- `backend/internal/httpapi`: handler dan `router.go`. Grup rute: `api` (sameOrigin), `authed`, `app` (password sudah diganti), `manage` (Admin dan Manajer).
- `backend/internal/audit`: pencatatan dan kategori audit log.
- `frontend/src`: `api/`, `components/`, `pages/`, `utils/`, `router/`, `stores/`.
- `scripts/check.sh`, `scripts/dev-login.sh`.

## Perintah

- Pemeriksaan wajib sebelum melapor selesai: `./scripts/check.sh` dari root (gofmt, `go build`, `npm run check`). Jangan jalankan `--full` kecuali diminta: test lama memang rusak sampai Sprint 13.
- Setelah mengubah query: `cd backend && sqlc generate`.
- Merapikan: `cd backend && gofmt -w .` dan `cd frontend && npx prettier --write src`.
- Migrasi dev, dari folder `backend`: `migrate -path migrations -database "$DATABASE_URL" up`. Rifky yang memuat `DATABASE_URL`. Kalau variabel itu kosong, minta Rifky menjalankannya, jangan membaca `.env`.
- Uji API manual, dari folder `backend`: `API=http://127.0.0.1:8080`, lalu `../scripts/dev-login.sh` (menyimpan cookie `/tmp/admin.txt`, `/tmp/staf.txt`, `/tmp/manajer.txt`). POST dan PUT wajib memakai `-H "Origin: $API" -H "Content-Type: application/json"`.
- Backend dijalankan Rifky di terminalnya sendiri. Setelah kode Go berubah, minta dia me-restart backend sebelum uji curl: `go build` tidak mengganti proses yang sedang jalan.

## Aturan kerja (wajib)

- Jangan `git commit`, `git push`, atau mengubah riwayat git. Di akhir tiap langkah berikan satu perintah yang siap ditempel: `./scripts/check.sh && git add <path> && git commit -m "<pesan>" && git push`.
- Kerjakan per langkah sprint (mis. S9-06) sesuai `docs/sprint.md`. Setelah tiap langkah tampilkan tabel semua langkah sprint itu beserta statusnya.
- Test otomatis tidak ditulis sebelum Sprint 13. Jangan memperbaiki test lama yang memakai `PICIDs`.
- `nvm_internal_test` sengaja belum dimigrasi ke `000003` dan `000004`. Jangan dimigrasi sebelum Sprint 13.
- Jangan menjalankan `migrate ... down`, `DROP`, `TRUNCATE`, atau `DELETE` massal tanpa izin eksplisit untuk perintah itu.
- Jangan membaca atau mencetak `backend/.env`, `DATABASE_URL`, password, token sesi, atau isi cookie.
- Jangan menebak isi file. Baca dulu, lalu ubah seperlunya.
- Aturan bisnis baru ditulis di bagian Keputusan sprint-nya di `docs/sprint.md`, supaya Sprint 13 tahu apa yang diuji. Hal yang butuh keputusan Manajer masuk "Menunggu jawaban Manajer", lalu pakai pilihan paling aman dan sebutkan.
- Revisi dari Manajer: catat di Daftar revisi, sisipkan ke sprint yang sesuai, lalu kerjakan.
- Akhir baris LF. File dengan CRLF ditandai `gofmt -l`.
- Kalau aturan plugin (antislop, ponytail, ui-ux-pro-max) bertabrakan dengan file ini, file ini yang dipakai.

## Konvensi kode

- Komentar berbahasa Indonesia, sentence case, menjelaskan kenapa, bukan apa. Simpan alasan bisnis, keamanan, trade-off, workaround, edge case. Buang komentar dekoratif, label kosong, dan narasi langkah.
- Pesan galat untuk pengguna berbahasa Indonesia dan spesifik.
- Hak akses selalu dicek di backend. Frontend hanya mengikuti nilai dari server (`can`, `move_to`, `can_check`), tidak menghitung aturannya sendiri.
- Galat validasi lewat `apperr.Invalid` (400). Perubahan task memakai optimistic locking `version` (409 bila basi).
- Parameter array sqlc dikirim sebagai slice kosong, bukan nil, karena `cardinality(NULL)` bukan 0.
- Batas hari memakai zona waktu kantor (`officeTZ`, Asia/Jakarta).
- Ponytail: tulis kode seperlunya, tetapi jangan memotong validasi, penanganan galat, keamanan, atau aksesibilitas.

## Standar UI dan teks (AntiSlop, wajib)

- Tanpa em dash di teks UI, dokumen, pesan commit, dan komentar. Pakai koma, titik, titik dua, atau kurung.
- Tanpa emoji. Ikon memakai `<svg>` inline yang cocok dengan isinya.
- Kontras WCAG AA dihitung, bukan ditebak: 4.5:1 teks biasa, 3:1 teks besar dan elemen non-teks (border, ikon status, focus ring).
- Tap target minimal 44x44 px (`min-h-11`, `size-11`). Tidak ada gulir horizontal di 360, 768, dan 1280 px. Grid memakai `minmax` atau `auto-fit` dan `min-w-0`.
- Keyboard penuh: Tab, Enter atau Space, Escape. Focus terlihat lewat `focus-visible`. Jangan `outline: none` tanpa pengganti.
- Setiap UI berbasis data punya state memuat, kosong, dan galat.
- Tidak ada data karangan (angka, nama, testimoni, klaim). Placeholder ditandai jelas.
- Pakai token warna yang sudah ada di `frontend/src/style.css` (line, canvas, surface, ink, ink-muted, accent, danger, urgent, dan token status). Warna baru butuh alasan satu kalimat.
- Animasi hanya bila punya tujuan UX, tanpa loop, dan punya versi diam untuk `prefers-reduced-motion`.
- Teks: spesifik, tanpa buzzword, tanpa klaim besar, CTA menyebut aksinya.
- Untuk pekerjaan UI dan dokumen, laporan akhir memuat Delivery Gate: satu baris PASS atau FAIL per poin di atas beserta buktinya. Satu FAIL berarti belum selesai.
- Sprint 11 (poles UI) baru dimulai setelah Rifky memberikan `docs/DESIGN.md`. Dial ENERGY, RHYTHM, dan MOTION ditetapkan di awal sprint itu dengan alasan, lalu dipegang sampai akhir.

## Data dev

- Akun: `admin`, `manajer.uji` (id 7, Marketing), `staf.uji` (id 4, Marketing), `staf.a` (id 5, Creative), dan `manajer.dua` untuk uji aturan antar Manajer.
- Task uji mulai `T-0009`. Data dev boleh berubah saat uji, data produksi belum ada.

## Cara melapor

Bahasa Indonesia santai, ringkas. Urutannya: apa yang berubah dan kenapa, perintah uji yang harus dijalankan Rifky (curl atau langkah browser) dengan hasil yang diharapkan, hasil `check.sh`, tabel status sprint, perintah commit.