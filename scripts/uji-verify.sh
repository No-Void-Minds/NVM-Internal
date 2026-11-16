#!/usr/bin/env bash
# Uji GET /api/auth/verify untuk forward_auth Caddy (revisi 5). Jalankan dari folder backend setelah migrasi
# 000006 dan 000007, restart backend, dan ../scripts/dev-login.sh:
#   API=http://127.0.0.1:8080 bash ../scripts/uji-verify.sh
# Membuat atau memakai ulang akun uji.verify (Staf, password sementara), lalu menonaktifkannya di akhir.
# Semua kasus pencabutan (ganti password, peran diubah, logout semua perangkat, reset password, nonaktif) memakai
# akun uji.verify, jadi password dan sesi admin, staf.uji, dan manajer.uji tidak diubah. Admin hanya dipakai
# untuk mengelola akun uji.verify.
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
    printf 'PASS  %-58s %s\n' "$name" "$got"
    pass=$((pass + 1))
  else
    printf 'FAIL  %-58s %s (harapan %s)\n' "$name" "$got" "$want"
    fail=$((fail + 1))
  fi
}

# verify ARGUMEN_CURL...: header di $TMP/head, body di $TMP/body, mencetak kode HTTP.
verify() {
  curl -s -o "$TMP/body" -D "$TMP/head" -w '%{http_code}' "$@" "$API/api/auth/verify"
}

nostore() { grep -qi '^cache-control: no-store' "$TMP/head" && echo 0 || echo 1; }
nocookie() { grep -qi '^set-cookie:' "$TMP/head" && echo 1 || echo 0; }

# admin METHOD PATH [JSON]: permintaan sebagai Admin, body di $TMP/abody, mencetak kode HTTP.
admin() {
  local args=(-s -o "$TMP/abody" -w '%{http_code}' -b "$ADMIN" -c "$ADMIN" -X "$1" -H "Origin: $API")
  [[ $# -ge 3 ]] && args+=(-H 'Content-Type: application/json' --data "$3")
  curl "${args[@]}" "$API/api$2"
}

if [[ ! -f "$ADMIN" ]]; then
  echo "Cookie $ADMIN tidak ada. Jalankan ../scripts/dev-login.sh dulu."
  exit 1
fi

echo '== Sesi sah'
code=$(verify -b "$ADMIN")
ok=1
grep -qi '^x-nvm-user: admin' "$TMP/head" && [[ ! -s "$TMP/body" ]] && [[ "$(nostore)" == 0 ]] && ok=0
hasil 'Cookie sah: 204, X-NVM-User, tanpa body, no-store' "$code" 204 "$ok"
hasil 'Cookie sah: tidak ada Set-Cookie' "$code" 204 "$(nocookie)"

# Cookie jar sengaja hanya dibaca (-b), jadi semua panggilan memakai token yang persis sama.
before=$(md5sum <"$ADMIN")
codes=''
cookies=0
for _ in 1 2 3 4 5; do
  codes+="$(verify -b "$ADMIN") "
  [[ "$(nocookie)" == 0 ]] || cookies=$((cookies + 1))
done
ok=1
[[ "$codes" == '204 204 204 204 204 ' && "$cookies" == 0 && "$(md5sum <"$ADMIN")" == "$before" ]] && ok=0
hasil "Verify 5 kali dengan cookie yang sama (${codes% })" 204 204 "$ok"

echo
echo '== Sesi tidak sah'
code=$(verify)
hasil 'Tanpa cookie: 401 dengan no-store' "$code" 401 "$(nostore)"
code=$(verify -H 'Cookie: nvm_session=bukan.token.sah')
hasil 'Cookie rusak: 401 dengan no-store' "$code" 401 "$(nostore)"
ok=1
grep -qi '^x-nvm-user' "$TMP/head" || ok=0
hasil 'Cookie rusak: tidak ada X-NVM-User' "$code" 401 "$ok"

echo
echo '== Pencabutan sesi (akun uji.verify)'
UJI=uji.verify
P1="Sementara-$(date +%s)-$RANDOM"
P2="Baru-$(date +%s)-$RANDOM"
P3="Reset-$(date +%s)-$RANDOM"

# login PASSWORD JAR: mencetak kode HTTP login uji.verify.
login() {
  curl -s -o /dev/null -w '%{http_code}' -c "$2" -H "Origin: $API" -H 'Content-Type: application/json' \
    --data "{\"username\":\"$UJI\",\"password\":\"$1\"}" "$API/api/auth/login"
}

admin GET /divisions >/dev/null
DIV=$(grep -o '"id":[0-9]*,"name":"[^"]*","is_active":true' "$TMP/abody" | head -1 | grep -o '^"id":[0-9]*' | cut -d: -f2)
code=$(admin POST /users "{\"name\":\"Uji Verify\",\"username\":\"$UJI\",\"role\":\"staff\",\"division_id\":${DIV:-0},\"password\":\"$P1\"}")
if [[ "$code" == 201 ]]; then
  UID_UJI=$(grep -o '"id":[0-9]*' "$TMP/abody" | cut -d: -f2)
  echo "      akun $UJI dibuat (id $UID_UJI)"
else
  # Sudah ada dari run sebelumnya: aktifkan lagi, kembalikan ke Staf, dan pasang password sementara baru.
  admin GET /users >/dev/null
  UID_UJI=$(grep -o "\"id\":[0-9]*,\"name\":\"[^\"]*\",\"username\":\"$UJI\"" "$TMP/abody" | grep -o '^"id":[0-9]*' | cut -d: -f2)
  admin POST "/users/$UID_UJI/activate" >/dev/null
  admin PUT "/users/$UID_UJI" "{\"name\":\"Uji Verify\",\"role\":\"staff\",\"division_id\":${DIV:-0}}" >/dev/null
  admin POST "/users/$UID_UJI/reset-password" "{\"password\":\"$P1\"}" >/dev/null
  echo "      akun $UJI sudah ada (id $UID_UJI), diaktifkan dan diberi password sementara baru"
fi

code=$(login "$P1" "$TMP/a.txt")
hasil "Login $UJI dengan password sementara" "$code" 200
code=$(verify -b "$TMP/a.txt")
hasil 'Password belum diganti: 403 dengan no-store' "$code" 403 "$(nostore)"

# Ganti password: sesi lama dicabut (token_version naik), perangkat ini mendapat sesi baru.
cp "$TMP/a.txt" "$TMP/a-lama.txt"
code=$(curl -s -o /dev/null -w '%{http_code}' -b "$TMP/a.txt" -c "$TMP/a.txt" -X POST -H "Origin: $API" \
  -H 'Content-Type: application/json' --data "{\"current_password\":\"$P1\",\"new_password\":\"$P2\"}" \
  "$API/api/auth/change-password")
hasil 'Ganti password' "$code" 204
code=$(verify -b "$TMP/a-lama.txt")
hasil 'Setelah ganti password: token lama 401' "$code" 401
code=$(verify -b "$TMP/a.txt")
hasil 'Setelah ganti password: sesi baru perangkat ini 204' "$code" 204

# Peran diubah: sejak awal tidak memaksa login ulang, karena hak akses dibaca ulang dari database setiap request.
code=$(admin PUT "/users/$UID_UJI" "{\"name\":\"Uji Verify\",\"role\":\"manager\",\"division_id\":${DIV:-0}}")
hasil 'Admin mengubah peran ke Manajer' "$code" 204
code=$(verify -b "$TMP/a.txt")
hasil 'Setelah peran diubah: sesi tetap 204 (tidak memaksa login ulang)' "$code" 204
admin PUT "/users/$UID_UJI" "{\"name\":\"Uji Verify\",\"role\":\"staff\",\"division_id\":${DIV:-0}}" >/dev/null

# Logout di satu perangkat mengakhiri sesi di semua perangkat (keluar dari semua perangkat).
code=$(login "$P2" "$TMP/b.txt")
hasil 'Login perangkat kedua' "$code" 200
cp "$TMP/b.txt" "$TMP/b-lama.txt"
code=$(curl -s -o /dev/null -w '%{http_code}' -b "$TMP/b.txt" -c "$TMP/b.txt" -X POST -H "Origin: $API" "$API/api/auth/logout")
hasil 'Logout dari perangkat kedua' "$code" 204
code=$(verify -b "$TMP/b-lama.txt")
hasil 'Setelah logout: perangkat yang logout 401' "$code" 401
code=$(verify -b "$TMP/a.txt")
hasil 'Setelah logout: perangkat lain ikut 401' "$code" 401

# Reset password oleh pengelola.
code=$(login "$P2" "$TMP/c.txt")
hasil 'Login lagi' "$code" 200
code=$(verify -b "$TMP/c.txt")
hasil 'Sebelum reset: 204' "$code" 204
code=$(admin POST "/users/$UID_UJI/reset-password" "{\"password\":\"$P3\"}")
hasil 'Admin mereset password' "$code" 204
code=$(verify -b "$TMP/c.txt")
hasil 'Setelah reset password: 401' "$code" 401

# Akun dinonaktifkan.
code=$(login "$P3" "$TMP/d.txt")
hasil 'Login dengan password sementara hasil reset' "$code" 200
code=$(verify -b "$TMP/d.txt")
hasil 'Password sementara hasil reset: 403' "$code" 403
code=$(admin POST "/users/$UID_UJI/deactivate")
hasil "Nonaktifkan $UJI" "$code" 204
code=$(verify -b "$TMP/d.txt")
hasil 'Setelah akun dinonaktifkan: 401' "$code" 401

code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$API/api/auth/verify")
hasil 'POST ke verify tidak dilayani' "$code" 404

echo
echo "Lolos $pass, gagal $fail. Akun $UJI dibiarkan nonaktif; sesi admin, staf.uji, dan manajer.uji tidak disentuh."
[[ "$fail" == 0 ]]
