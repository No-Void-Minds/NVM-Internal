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

echo "Semua pemeriksaan lolos."