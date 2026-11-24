# Sistem Internal NVM

Autentikasi dan Task Tracker untuk tim NVM. Kebutuhan lengkap ada di docs/SRS_Sistem_Internal_NVM.docx, rencana kerja di docs/sprint.md.

## Versi alat

- Go 1.27.0
- Node 24.20.0, npm 11.19.0
- PostgreSQL 18 (dev dan test: lokal di laptop; produksi: Docker postgres:18-alpine di VM `nvm-server`)
- golang-migrate v4.20.1
- sqlc 1.31.1

## Database dev dan test

Database dev dan test memakai PostgreSQL 18 lokal di laptop, bukan Postgres di VM. Server hanya mendengar di localhost (`listen_addresses = 'localhost'` di `postgresql.conf`), port 5432. Aplikasi masuk sebagai role `nvm_dev` yang bukan superuser dan memiliki dua database:

- `nvm_internal`: database dev untuk backend dan uji di browser.
- `nvm_internal_test`: database test untuk `go test` dan `scripts/pra-deploy.sh`. Isinya dikosongkan setiap test, jadi jangan dipakai untuk data yang ingin disimpan.

Membuat role dan database, sekali saja, sebagai superuser `postgres` (ganti `GANTI` dengan password sendiri):

```bash
psql -U postgres -h localhost -c "CREATE ROLE nvm_dev LOGIN PASSWORD 'GANTI'"
psql -U postgres -h localhost -c "CREATE DATABASE nvm_internal OWNER nvm_dev"
psql -U postgres -h localhost -c "CREATE DATABASE nvm_internal_test OWNER nvm_dev"
```

`nvm_dev` cukup menjadi pemilik database. Ekstensi `pg_trgm` (migrasi `000002`) adalah trusted extension, jadi pemilik database boleh memasangnya tanpa superuser, dan pemilik tabel boleh menjalankan `TRUNCATE` yang dipakai test.

Isi `backend/.env` dengan dua URL ini (contoh format, password-nya bukan yang asli; karakter khusus di password harus di-URL-encode):

```bash
DATABASE_URL=postgres://nvm_dev:GANTI@localhost:5432/nvm_internal?sslmode=disable
TEST_DATABASE_URL=postgres://nvm_dev:GANTI@localhost:5432/nvm_internal_test?sslmode=disable
```

Nama database test harus berakhiran `_test` dan berbeda dari database dev. `scripts/pra-deploy.sh` menolak jalan kalau syarat itu tidak terpenuhi, lalu memigrasikan `nvm_internal_test` sendiri ke versi terbaru (hanya `up`) sebelum menjalankan test. Test DB tidak perlu dimigrasikan manual.

## Menjalankan saat dev

Dua terminal:

1. Backend, dari `backend/`: `go run ./cmd/server`
2. Frontend, dari `frontend/`: `npm run dev`, lalu buka http://localhost:5173

Pertama kali di database dev kosong, dari `backend/`:

```bash
set -a; source .env; set +a
migrate -path migrations -database "$DATABASE_URL" up
go run ./cmd/seed
```

Sebelum commit: `./scripts/check.sh`

## Produksi

Deploy, migrasi, backup dan restore, job harian, update versi, dan rollback ditulis langkah per langkah di [docs/runbook-deploy.md](docs/runbook-deploy.md). Berkas deploy ada di `deploy/` (compose, contoh `.env`, SQL role database, skrip backup). Panduan pemakaian untuk tim: [docs/panduan-tim.md](docs/panduan-tim.md).
