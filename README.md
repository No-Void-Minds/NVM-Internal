# Sistem Internal NVM

Autentikasi dan Task Tracker untuk tim NVM. Kebutuhan lengkap ada di docs/SRS_Sistem_Internal_NVM.docx, rencana kerja di docs/sprint.md.

## Versi alat

- Go 1.27.0
- Node 24.20.0, npm 11.19.0
- PostgreSQL 18.6 (Docker, postgres:18-alpine)
- golang-migrate v4.20.1
- sqlc 1.31.1

## Menjalankan saat dev

Tiga terminal:

1. Tunnel ke database di server: `ssh -N -o ServerAliveInterval=60 -L 15432:127.0.0.1:5432 nvmadmin@192.168.68.144`
2. Backend, dari `backend/`: `go run ./cmd/server`
3. Frontend, dari `frontend/`: `npm run dev`, lalu buka http://localhost:5173

Pertama kali di database kosong, dari `backend/`:

```bash
set -a; source .env; set +a
migrate -path migrations -database "$DATABASE_URL" up
go run ./cmd/seed
```

Sebelum commit: `./scripts/check.sh`