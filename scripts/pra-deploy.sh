#!/usr/bin/env bash
# Gerbang sebelum commit fitur backend dan deploy. Lahir dari bug 500 revisi 31 yang lolos ke produksi padahal test
# DB-nya sudah ada: test itu SKIP karena TEST_DATABASE_URL tidak termuat, dan test DB tertinggal migrasinya.
# Karena itu SKIP dihitung gagal, tanpa daftar pengecualian.
# Jangan tambahkan set -x: skrip ini memegang URL database berisi password.
set -euo pipefail
cd "$(dirname "$0")/.."

start=$(date +%s)
mig_version="-"
pass=0
fail=0
skip=0
log_dir=$(mktemp -d)
trap 'rm -rf "$log_dir"' EXIT
# Output -json test disimpan di luar folder sementara supaya test yang lambat bisa dilacak sesudahnya (run 1046
# detik di revisi 40 tidak bisa diperiksa karena berkasnya ikut terhapus). Ditimpa setiap run; CI mengunggahnya.
json_out=${PRA_DEPLOY_JSON:-backend/tmp/pra-deploy-test.json}
# Berkas run sebelumnya dibuang dulu, supaya run yang gagal sebelum test tidak menunjuk hasil lama.
rm -f "$json_out"

# Menyamarkan URL database di keluaran perintah lain, berjaga-jaga bila galatnya menyebut DSN.
mask() { sed -E 's#postgres(ql)?://[^[:space:]"]*#postgres://***#g'; }

summary() {
  local status=$1
  echo
  echo "===== Ringkasan pra-deploy ====="
  echo "Versi migrasi test DB : $mig_version"
  echo "Test backend          : PASS $pass, FAIL $fail, SKIP $skip"
  echo "Durasi                : $(($(date +%s) - start)) detik"
  if [ -s "$json_out" ]; then echo "Output -json test     : $json_out"; fi
  if [ "$status" = 0 ]; then echo "PRA-DEPLOY LOLOS"; else echo "PRA-DEPLOY GAGAL"; fi
  exit "$status"
}

die() {
  echo "GAGAL: $*" >&2
  summary 1
}

echo "1. Memuat variabel database"
if [ -z "${TEST_DATABASE_URL:-}" ] || [ -z "${DATABASE_URL:-}" ]; then
  if [ -f backend/.env ]; then
    # Isinya tidak dicetak; hanya variabelnya yang dipakai.
    set -a
    # shellcheck disable=SC1091
    . backend/.env
    set +a
  fi
fi
[ -n "${TEST_DATABASE_URL:-}" ] || die "TEST_DATABASE_URL kosong, baik di env maupun di backend/.env."

# Nama database: bagian path terakhir dari URL, tanpa query string. Bentuk key=value (dbname=...) juga diterima.
db_name() {
  local u=$1
  if [[ $u == *://* ]]; then
    u=${u%%\?*}
    printf '%s' "${u##*/}"
  else
    printf '%s' "$u" | sed -nE 's/(^|.*[[:space:]])dbname=([^[:space:]]+).*/\2/p'
  fi
}

echo "2. Pengaman database test"
test_db=$(db_name "$TEST_DATABASE_URL")
[ -n "$test_db" ] || die "Nama database tidak terbaca dari TEST_DATABASE_URL."
# Test mengosongkan tabel (TRUNCATE), jadi salah arah ke database dev berarti data dev hilang.
[[ $test_db == *_test ]] || die "Database test harus berakhiran _test, ternyata \"$test_db\"."
if [ -n "${DATABASE_URL:-}" ]; then
  dev_db=$(db_name "$DATABASE_URL")
  [ "$test_db" != "$dev_db" ] || die "TEST_DATABASE_URL dan DATABASE_URL menunjuk database yang sama ($test_db)."
  [ "$TEST_DATABASE_URL" != "$DATABASE_URL" ] || die "TEST_DATABASE_URL sama persis dengan DATABASE_URL."
fi
echo "   database test: $test_db"

echo "3. Migrasi database test ke versi terbaru"
migrate() {
  (cd backend && go run -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.3 \
    -path=migrations -database "$TEST_DATABASE_URL" "$@")
}
# Hanya up. Tidak pernah force atau down: keadaan dirty harus diperiksa manusia.
if ! migrate up >"$log_dir/migrate.log" 2>&1; then
  mask <"$log_dir/migrate.log" >&2
  die "Migrasi database test gagal."
fi
mask <"$log_dir/migrate.log"
migrate version >"$log_dir/version.log" 2>&1 || {
  mask <"$log_dir/version.log" >&2
  die "Versi migrasi database test tidak terbaca."
}
mig_version=$(mask <"$log_dir/version.log" | tail -n 1)
if [[ $mig_version == *dirty* ]]; then die "Database test dalam keadaan dirty ($mig_version)."; fi
latest=$(find backend/migrations -name '*.up.sql' | sed -E 's#.*/0*([0-9]+)_.*#\1#' | sort -n | tail -n 1)
[ "$mig_version" = "$latest" ] || die "Versi database test $mig_version, migrasi terbaru $latest."

echo "4. Test backend"
# -p 1: semua paket memakai satu database test, dan testdb.Pool meng-TRUNCATE tabel bersama di awal setiap test.
# Advisory lock di testdb sudah membuat test DB antre, tetapi lock itu baru diambil saat Pool dipanggil dan
# dilepas di akhir test. Menjalankan paket satu per satu menutup celah di luar itu dan membuat urutan bisa diulang.
# -count=1 mematikan cache: hasil lama dari sebelum migrasi atau sebelum env termuat tidak boleh terhitung.
set +e
(cd backend && go test -count=1 -p 1 -json ./...) >"$log_dir/test.json" 2>"$log_dir/test.err"
test_exit=$?
set -e
mkdir -p "$(dirname "$json_out")"
mask <"$log_dir/test.json" >"$json_out"

json=$log_dir/test.json
count() { grep -E "\"Action\":\"$1\"" "$json" | grep -c '"Test":' || true; }
pass=$(count pass)
fail=$(count fail)
skip=$(count skip)

# Isi field Output dari satu baris -json. Go terbaru menambah field OutputType sesudahnya, jadi yang diambil
# hanya string Output itu sendiri, lalu escape JSON yang umum dikembalikan.
output_of() {
  sed -E 's/.*"Output":"(([\\].|[^"\\])*)".*/\1/; s/[\\]n$//; s/[\\]t/    /g; s/[\\]"/"/g' |
    sed -E 's/[\\]u003c/</g; s/[\\]u003e/>/g; s/[\\]u0026/\&/g'
}

# Nama dan pesan setiap test yang FAIL atau SKIP, diambil dari baris output milik test itu. Setiap grep di sini
# boleh tidak menemukan apa pun, jadi diberi || true supaya set -e tidak menghentikan skrip.
report() {
  local action=$1
  { grep -E "\"Action\":\"$action\"" "$json" | grep '"Test":' || true; } |
    sed -E 's/.*"Package":"([^"]*)".*"Test":"([^"]*)".*/\1 \2/' | while read -r pkg name; do
      echo "--- $(echo "$action" | tr a-z A-Z): $pkg $name"
      { grep -F "\"Package\":\"$pkg\"" "$json" | grep -F "\"Test\":\"$name\"" | grep '"Action":"output"' || true; } |
        output_of | { grep -vE '^ *(=== (RUN|PAUSE|CONT|NAME)|--- (FAIL|SKIP|PASS))' || true; } | sed 's/^/    /' | mask
    done
}
report fail
report skip

# Paket yang gagal tanpa test (mis. galat kompilasi) tidak punya baris "Test"; tetap harus menggagalkan gerbang.
pkg_fail=$(grep '"Action":"fail"' "$json" | grep -vc '"Test":' || true)
# Paket dengan test FAIL selalu ikut gagal di tingkat paket, jadi bagian ini hanya dicetak bila penyebabnya lain.
if { [ "$pkg_fail" -gt 0 ] && [ "$fail" -eq 0 ]; } || [ -s "$log_dir/test.err" ]; then
  echo "--- Paket gagal tanpa nama test atau galat kompilasi:"
  grep '"Action":"fail"' "$json" | grep -v '"Test":' | sed -E 's/.*"Package":"([^"]*)".*/    \1/' || true
  { grep -E '"Action":"(build-output|output)"' "$json" | grep -v '"Test":' || true; } | output_of |
    { grep -iE 'FAIL|error|cannot|undefined' || true; } | sed 's/^/    /' | mask
  mask <"$log_dir/test.err" | sed 's/^/    /'
fi

[ "$fail" -eq 0 ] || die "Ada $fail test yang FAIL."
[ "$skip" -eq 0 ] || die "Ada $skip test yang SKIP. SKIP dihitung gagal."
[ "$pkg_fail" -eq 0 ] && [ "$test_exit" -eq 0 ] || die "go test keluar dengan kode $test_exit."
[ "$pass" -gt 0 ] || die "Tidak ada test yang lolos sama sekali."

echo "5. check.sh"
./scripts/check.sh || die "check.sh gagal."

summary 0
