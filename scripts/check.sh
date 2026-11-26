#!/usr/bin/env bash
# Mode bawaan cepat untuk fase build; --full menjalankan vet dan test (dipakai di Sprint 13 dan sebelum rilis).
set -euo pipefail
cd "$(dirname "$0")/.."

full=0
[ "${1:-}" = "--full" ] && full=1

echo "backend: gofmt"
unformatted=$(gofmt -l backend)
if [ -n "$unformatted" ]; then
  echo "File Go belum diformat (jalankan gofmt -w backend):"
  echo "$unformatted"
  exit 1
fi

# Skrip dipanggil lewat path (cron di VM, CI, ./scripts/...), jadi izin eksekusinya harus ikut di git. Di Windows
# git tidak merekam bit itu sendiri (core.filemode false), dan drive-export.sh pernah ter-commit 100644
# sehingga cron tidak bisa menjalankannya (revisi 25).
echo "skrip: izin eksekusi"
nonexec=$(git ls-files -s -- '*.sh' | awk '$1 != "100755" { print $4 }')
if [ -n "$nonexec" ]; then
  echo "Skrip berikut belum 100755 di indeks git (perbaiki dengan git add --chmod=+x <berkas>):"
  echo "$nonexec"
  exit 1
fi

if [ "$full" = 1 ]; then
  echo "backend: go vet dan go test"
  (cd backend && go vet ./... && go test ./...)
else
  # go build tidak mengompilasi file test, jadi test lama yang rusak oleh rombakan sub task tidak menghalangi commit.
  echo "backend: go build"
  (cd backend && go build ./...)
fi

echo "frontend: tipe, lint, format"
(cd frontend && npm run check)

# Tailwind v4 hanya mengeluarkan variabel @theme yang dirujuknya; token yang namanya dirakit dinamis pernah hilang
# dari build produksi (label kursor Whiteboard transparan di tema terang). Membangun CSS ke folder sementara.
echo "frontend: token CSS ada di build"
(cd frontend && node scripts/cek-token.mjs)

echo "Semua pemeriksaan lolos."