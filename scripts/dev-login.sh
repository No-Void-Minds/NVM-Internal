#!/usr/bin/env bash
# Login ulang akun uji dev dan simpan cookie di /tmp. Dibutuhkan karena sesi berakhir
# setelah 30 menit tanpa aktivitas, jadi cookie lama tidak lagi dikirim curl.
set -euo pipefail
cd "$(dirname "$0")/.."
set -a; source backend/.env; set +a

API=http://127.0.0.1:8080/api

login() {
  local user=$1 pass=$2 jar=$3
  local code
  code=$(curl -s -o /dev/null -w '%{http_code}' -c "$jar" \
    -H 'Origin: http://127.0.0.1:8080' -H 'Content-Type: application/json' \
    --data "$(printf '{"username":"%s","password":"%s"}' "$user" "$pass")" \
    "$API/auth/login")
  echo "$user: $code"
}

login admin "$DEV_ADMIN_PASSWORD" /tmp/admin.txt
login "$DEV_STAFF_USERNAME" "$DEV_STAFF_PASSWORD" /tmp/staf.txt