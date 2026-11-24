#!/usr/bin/env bash
# Menguji deploy/cek-ci.py dengan jawaban check-runs contoh di deploy/testdata/cek-ci, tanpa memanggil API GitHub.
# Bentuk jawabannya diambil dari contoh di dokumentasi GitHub "List check runs for a Git reference".
set -euo pipefail
cd "$(dirname "$0")/.."

py=python3
# Di laptop Windows python3 tidak ada; CLAUDE.md memakai py -3.
command -v python3 >/dev/null 2>&1 && python3 -c '' 2>/dev/null || py="py -3"
sha=ce587453ced02b1526dfb4cb910479d431683101
dir=deploy/testdata/cek-ci
gagal=0

kasus() {
  local berkas=$1 mau=$2 pesan=$3 sha_uji=${4:-$sha} keluar=0 out
  out=$($py deploy/cek-ci.py "$sha_uji" pra-deploy <"$dir/$berkas") || keluar=$?
  if [ "$keluar" != "$mau" ] || [[ $out != *"$pesan"* ]]; then
    echo "GAGAL $berkas: keluar $keluar (mau $mau), pesan \"$out\""
    gagal=$((gagal + 1))
  else
    echo "lolos $berkas: $out"
  fi
}

kasus lolos.json 0 "lolos untuk commit ce587453ced0"
kasus ulang-lolos.json 0 "lolos untuk commit"
kasus gagal.json 1 "tidak lolos (hasil failure)"
kasus berjalan.json 1 "belum selesai (status in_progress)"
kasus docs-asli.json 1 "tidak ditemukan"
kasus app-lain.json 1 "tidak ditemukan"
kasus lolos.json 1 "tidak ditemukan" 0000000000000000000000000000000000000000

keluar=0
out=$(echo 'bukan json' | $py deploy/cek-ci.py "$sha" pra-deploy) || keluar=$?
if [ "$keluar" = 1 ] && [[ $out == *"bukan JSON"* ]]; then echo "lolos bukan-json: $out"; else
  echo "GAGAL bukan-json: keluar $keluar, pesan \"$out\""
  gagal=$((gagal + 1))
fi

[ "$gagal" = 0 ] || { echo "$gagal kasus gagal"; exit 1; }
echo "Semua kasus cek-ci lolos."
