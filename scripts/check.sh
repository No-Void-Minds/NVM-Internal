#!/usr/bin/env bash
# Jalankan sebelum commit atau merge; semua pemeriksaan backend dan frontend harus lolos.
set -euo pipefail
cd "$(dirname "$0")/.."

echo "backend: gofmt"
unformatted=$(gofmt -l backend)
if [ -n "$unformatted" ]; then
  echo "File Go belum diformat (jalankan gofmt -w backend):"
  echo "$unformatted"
  exit 1
fi

echo "backend: go vet dan go test"
(cd backend && go vet ./... && go test ./...)

echo "frontend: tipe, lint, format"
(cd frontend && npm run check)

echo "Semua pemeriksaan lolos."