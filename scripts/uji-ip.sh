#!/usr/bin/env bash
# Uji A5 (Sprint 12): header IP palsu yang dikirim langsung ke backend dev tidak dipercaya. Di dev TRUSTED_PROXIES
# kosong, jadi audit log dan limiter login harus memakai alamat koneksi, bukan isi header. Jalankan dari folder
# backend setelah ../scripts/dev-login.sh:
#   API=http://127.0.0.1:8080 bash ../scripts/uji-ip.sh
# Memakai satu percobaan login gagal dengan username yang tidak ada (ikut terhitung limiter login per IP).
set -uo pipefail

API=${API:-http://127.0.0.1:8080}
ADMIN=/tmp/admin.txt
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
PALSU=203.0.113.7
U="uji-ip-$(date +%s)"

if [[ ! -f "$ADMIN" ]]; then
  echo "Cookie $ADMIN tidak ada. Jalankan ../scripts/dev-login.sh dulu."
  exit 1
fi

code=$(curl -s -o /dev/null -w '%{http_code}' -H "Origin: $API" -H 'Content-Type: application/json' \
  -H "CF-Connecting-IP: $PALSU" -H "X-Forwarded-For: $PALSU" -H "X-Real-IP: $PALSU" \
  --data "{\"username\":\"$U\",\"password\":\"salah-sekali-uji\"}" "$API/api/auth/login")
echo "Login gagal dengan header IP palsu: $code (harapan 401)"

curl -s -b "$ADMIN" -c "$ADMIN" "$API/api/audit-logs?category=auth" >"$TMP/audit"
ip=$(grep -o "\"username\":\"$U\"},\"ip\":\"[^\"]*\"" "$TMP/audit" | head -1 | sed 's/.*"ip":"\([^"]*\)"/\1/')
echo "IP tercatat di audit log: ${ip:-tidak ditemukan}"

if [[ "$code" == 401 && -n "$ip" && "$ip" != "$PALSU" ]]; then
  echo "PASS  header CF-Connecting-IP, X-Forwarded-For, dan X-Real-IP palsu tidak dipercaya"
  exit 0
fi
echo "FAIL  harapan: 401 dan IP bukan $PALSU"
exit 1
