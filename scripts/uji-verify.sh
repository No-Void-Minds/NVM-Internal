#!/usr/bin/env bash
# Uji GET /api/auth/verify untuk forward_auth Caddy (revisi 5). Jalankan dari folder backend setelah backend
# di-restart dan ../scripts/dev-login.sh:
#   API=http://127.0.0.1:8080 bash ../scripts/uji-verify.sh
# Kasus logout mengakhiri sesi Admin di semua perangkat; skrip menjalankan dev-login.sh lagi di akhir.
set -uo pipefail

API=${API:-http://127.0.0.1:8080}
ADMIN=/tmp/admin.txt
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
pass=0
fail=0

hasil() {
  local name=$1 got=$2 want=$3 extra=${4:-0}
  if [[ "$got" == "$want" && "$extra" == 0 ]]; then
    printf 'PASS  %-52s %s\n' "$name" "$got"
    pass=$((pass + 1))
  else
    printf 'FAIL  %-52s %s (harapan %s)\n' "$name" "$got" "$want"
    fail=$((fail + 1))
  fi
}

# verify ARGUMEN_CURL...: header di $TMP/head, body di $TMP/body, mencetak kode HTTP.
verify() {
  curl -s -o "$TMP/body" -D "$TMP/head" -w '%{http_code}' "$@" "$API/api/auth/verify"
}

nostore() {
  grep -qi '^cache-control: no-store' "$TMP/head" && echo 0 || echo 1
}

if [[ ! -f "$ADMIN" ]]; then
  echo "Cookie $ADMIN tidak ada. Jalankan ../scripts/dev-login.sh dulu."
  exit 1
fi

code=$(verify -b "$ADMIN" -c "$ADMIN")
ok=1
grep -qi '^x-nvm-user: admin' "$TMP/head" && [[ ! -s "$TMP/body" ]] && [[ "$(nostore)" == 0 ]] && ok=0
hasil 'Cookie sah: 204, X-NVM-User, tanpa body, no-store' "$code" 204 "$ok"

code=$(verify)
hasil 'Tanpa cookie: 401 dengan no-store' "$code" 401 "$(nostore)"

code=$(verify -H 'Cookie: nvm_session=bukan.token.sah')
hasil 'Cookie rusak: 401 dengan no-store' "$code" 401 "$(nostore)"
ok=1
grep -qi '^x-nvm-user' "$TMP/head" || ok=0
hasil 'Cookie rusak: tidak ada X-NVM-User' "$code" 401 "$ok"

# Salinan cookie sebelum logout, untuk membuktikan token lama tidak lagi diterima.
cp "$ADMIN" "$TMP/lama.txt"
code=$(curl -s -o /dev/null -w '%{http_code}' -b "$ADMIN" -c "$ADMIN" -X POST -H "Origin: $API" "$API/api/auth/logout")
hasil 'Logout' "$code" 204
code=$(verify -b "$TMP/lama.txt")
hasil 'Token sebelum logout: 401 dengan no-store' "$code" 401 "$(nostore)"

code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$API/api/auth/verify")
hasil 'POST ke verify tidak dilayani' "$code" 404

echo
echo "Login ulang akun uji:"
"$(dirname "$0")/dev-login.sh"
echo "Lolos $pass, gagal $fail."
[[ "$fail" == 0 ]]
