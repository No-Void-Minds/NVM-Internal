#!/usr/bin/env bash
# Uji API Sprint 10 (profil, foto, jumlah task). Jalankan dari folder backend setelah ../scripts/dev-login.sh:
#   API=http://127.0.0.1:8080 ../scripts/uji-sprint10.sh
# Mengubah nama, jabatan, dan foto akun Staf uji, lalu mengembalikan nama dan jabatannya di akhir.
# Fotonya berakhir terhapus. Gambar uji dibuat di folder sementara, tidak disimpan di repo.
set -uo pipefail

API=${API:-http://127.0.0.1:8080}
STAF=/tmp/staf.txt
MANAJER=/tmp/manajer.txt
ADMIN=/tmp/admin.txt
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

pass=0
fail=0

# hasil NAMA KODE_DAPAT KODE_HARAPAN [syarat tambahan: 0 berarti lolos]
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

# req JAR METHOD PATH [argumen curl lain...]: isi respons di $TMP/body, header di $TMP/head, mencetak kode HTTP.
req() {
  local jar=$1 method=$2 path=$3
  shift 3
  curl -s -o "$TMP/body" -D "$TMP/head" -w '%{http_code}' -b "$jar" -c "$jar" -X "$method" \
    -H "Origin: $API" "$@" "$API/api$path"
}

json() {
  req "$1" "$2" "$3" -H 'Content-Type: application/json' --data "$4"
}

# Nilai string dari JSON datar, cukup untuk respons API ini tanpa jq.
field() {
  grep -o "\"$1\":\"[^\"]*\"" "$TMP/body" | head -1 | cut -d'"' -f4
}

num() {
  grep -o "\"$1\":[0-9]*" "$TMP/body" | head -1 | cut -d: -f2
}

for jar in "$STAF" "$MANAJER" "$ADMIN"; do
  if [[ ! -f "$jar" ]]; then
    echo "Cookie $jar tidak ada. Jalankan ../scripts/dev-login.sh dulu."
    exit 1
  fi
done

code=$(req "$STAF" GET /auth/me)
if [[ "$code" != 200 ]]; then
  echo "GET /auth/me Staf: $code. Sesi habis? Jalankan ../scripts/dev-login.sh lagi."
  exit 1
fi
STAF_ID=$(num id)
ORIG_NAME=$(field name)
ORIG_JOB=$(field job_title)
req "$MANAJER" GET /auth/me >/dev/null
MANAJER_ID=$(num id)
MANAJER_NAME=$(field name)
echo "Staf id $STAF_ID ($ORIG_NAME), Manajer id $MANAJER_ID ($MANAJER_NAME)"
echo

# Generator gambar uji. Modul sendiri supaya tidak tercampur modul backend.
mkdir -p "$TMP/gen"
cat >"$TMP/gen/go.mod" <<'EOF'
module gen

go 1.22
EOF
cat >"$TMP/gen/main.go" <<'EOF'
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
)

func main() {
	if os.Args[1] == "check" {
		f, _ := os.Open(os.Args[2])
		img, err := jpeg.Decode(f)
		if err != nil {
			fmt.Println("bukan-jpeg")
			return
		}
		b := img.Bounds()
		top, _, _, _ := img.At(b.Dx()/2, b.Dy()/10).RGBA()
		bottom, _, _, _ := img.At(b.Dx()/2, b.Dy()*9/10).RGBA()
		arah := "miring"
		if top>>8 > 200 && bottom>>8 < 60 {
			arah = "tegak"
		}
		fmt.Printf("%dx%d %s\n", b.Dx(), b.Dy(), arah)
		return
	}
	// Foto mendatar 400x200, kiri merah dan kanan biru, dengan EXIF orientasi 6 seperti foto ponsel yang diambil
	// tegak. Setelah putaran diterapkan, merah harus berada di atas.
	src := image.NewRGBA(image.Rect(0, 0, 400, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 400; x++ {
			c := color.RGBA{220, 30, 30, 255}
			if x >= 200 {
				c = color.RGBA{30, 30, 220, 255}
			}
			src.Set(x, y, c)
		}
	}
	var jb bytes.Buffer
	_ = jpeg.Encode(&jb, src, &jpeg.Options{Quality: 90})
	payload := append([]byte("Exif\x00\x00"), 'I', 'I', 0x2A, 0, 8, 0, 0, 0, 1, 0, 0x12, 0x01, 3, 0, 1, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0)
	app1 := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(app1[2:], uint16(len(payload)+2))
	out := append([]byte{}, jb.Bytes()[:2]...)
	out = append(out, app1...)
	out = append(out, payload...)
	out = append(out, jb.Bytes()[2:]...)
	_ = os.WriteFile("tegak.jpg", out, 0o644)

	var pb bytes.Buffer
	_ = png.Encode(&pb, image.NewNRGBA(image.Rect(0, 0, 300, 300)))
	_ = os.WriteFile("transparan.png", pb.Bytes(), 0o644)
}
EOF
if ! (cd "$TMP/gen" && go run . gen); then
  echo "Generator gambar gagal dijalankan (butuh Go)."
  exit 1
fi
printf 'ini teks biasa, bukan foto\n' >"$TMP/gen/teks.jpg"
# 3 MB berawalan tanda JPEG, supaya yang menolak adalah batas ukuran, bukan pemeriksa tipe.
{ printf '\xff\xd8\xff\xe0'; head -c $((3 * 1024 * 1024)) /dev/zero; } >"$TMP/gen/besar.jpg"

echo '== Profil'
code=$(json "$STAF" PUT /me/profile '{"name":"Staf Uji Sprint 10","job_title":"  Desainer Grafis  "}')
hasil 'Ubah nama dan jabatan (204)' "$code" 204
req "$STAF" GET /auth/me >/dev/null
ok=1
[[ "$(field name)" == 'Staf Uji Sprint 10' && "$(field job_title)" == 'Desainer Grafis' ]] && ok=0
hasil 'Nama dan jabatan tersimpan, spasi jabatan dirapikan' "$code" 204 "$ok"

code=$(json "$STAF" PUT /me/profile '{"name":"Staf Uji Sprint 10","job_title":"   "}')
req "$STAF" GET /auth/me >/dev/null
ok=1
grep -q '"job_title":null' "$TMP/body" && ok=0
hasil 'Jabatan kosong tersimpan sebagai null' "$code" 204 "$ok"

long=$(printf 'J%.0s' $(seq 1 101))
code=$(json "$STAF" PUT /me/profile "{\"name\":\"Staf Uji Sprint 10\",\"job_title\":\"$long\"}")
ok=1
grep -q '"field":"job_title"' "$TMP/body" && ok=0
hasil 'Jabatan 101 karakter ditolak, galat di kolom job_title' "$code" 400 "$ok"

code=$(json "$STAF" PUT /me/profile '{"name":"   ","job_title":""}')
ok=1
grep -q '"field":"name"' "$TMP/body" && ok=0
hasil 'Nama kosong ditolak, galat di kolom name' "$code" 400 "$ok"

echo
echo '== Foto'
code=$(req "$STAF" PUT /me/avatar -F "file=@$TMP/gen/tegak.jpg;type=image/jpeg")
AVATAR=$(field avatar_id)
hasil 'Unggah JPEG valid (foto ponsel EXIF orientasi 6)' "$code" 200
code=$(req "$STAF" GET "/users/$STAF_ID/avatar?v=$AVATAR")
cp "$TMP/body" "$TMP/gen/hasil.jpg"
cek=$(cd "$TMP/gen" && go run . check hasil.jpg)
ok=1
[[ "$cek" == '256x256 tegak' ]] && ok=0
hasil "Hasil 256x256 dan tegak (dapat: $cek)" "$code" 200 "$ok"
ok=1
grep -qa 'Exif' "$TMP/gen/hasil.jpg" || ok=0
hasil 'EXIF tidak ikut tersimpan' "$code" 200 "$ok"
ok=1
grep -qi '^content-type: image/jpeg' "$TMP/head" && grep -qi '^x-content-type-options: nosniff' "$TMP/head" &&
  grep -qi "^etag: \"$AVATAR\"" "$TMP/head" && grep -qi '^cache-control: private, max-age=31536000, immutable' "$TMP/head" && ok=0
hasil 'Header foto: image/jpeg, nosniff, ETag, cache immutable' "$code" 200 "$ok"
code=$(req "$STAF" GET "/users/$STAF_ID/avatar?v=$AVATAR" -H "If-None-Match: \"$AVATAR\"")
hasil 'GET foto dengan If-None-Match yang cocok' "$code" 304
code=$(req "$MANAJER" GET "/users/$STAF_ID/avatar?v=$AVATAR")
hasil 'Foto Staf bisa dilihat akun lain yang login' "$code" 200
code=$(curl -s -o /dev/null -w '%{http_code}' "$API/api/users/$STAF_ID/avatar?v=$AVATAR")
hasil 'Foto tanpa login ditolak' "$code" 401

code=$(req "$STAF" PUT /me/avatar -F "file=@$TMP/gen/transparan.png;type=image/png")
NEW=$(field avatar_id)
ok=1
[[ -n "$NEW" && "$NEW" != "$AVATAR" ]] && ok=0
hasil 'Unggah PNG valid, avatar_id berganti' "$code" 200 "$ok"
code=$(req "$STAF" GET "/users/$STAF_ID/avatar?v=$NEW" -H "If-None-Match: \"$AVATAR\"")
hasil 'ETag lama tidak lagi cocok setelah foto diganti' "$code" 200

code=$(req "$STAF" PUT /me/avatar -F "file=@$TMP/gen/teks.jpg;type=image/jpeg")
hasil 'Berkas teks berganti nama .jpg ditolak' "$code" 415
code=$(req "$STAF" PUT /me/avatar -F "file=@$TMP/gen/besar.jpg;type=image/jpeg")
hasil 'Berkas di atas 2 MB ditolak' "$code" 413

code=$(req "$STAF" DELETE /me/avatar)
hasil 'Hapus foto' "$code" 204
code=$(req "$STAF" GET "/users/$STAF_ID/avatar")
hasil 'GET foto setelah dihapus' "$code" 404
code=$(req "$STAF" DELETE /me/avatar)
hasil 'Hapus foto saat sudah tidak ada foto tetap 204' "$code" 204

echo
echo '== Hak: Staf tidak bisa mengubah profil orang lain'
code=$(json "$STAF" PUT "/users/$MANAJER_ID" '{"name":"Diubah Staf","role":"manager","division_id":1}')
hasil 'PUT /users/:id milik Manajer oleh Staf' "$code" 403
code=$(json "$STAF" PUT /me/profile "{\"id\":$MANAJER_ID,\"user_id\":$MANAJER_ID,\"name\":\"Staf Uji Sprint 10\",\"job_title\":\"\"}")
req "$MANAJER" GET /auth/me >/dev/null
ok=1
[[ "$(field name)" == "$MANAJER_NAME" ]] && ok=0
hasil 'PUT /me/profile dengan id Manajer hanya mengubah akun sendiri' "$code" 204 "$ok"
code=$(req "$STAF" PUT "/users/$MANAJER_ID/avatar" -F "file=@$TMP/gen/tegak.jpg;type=image/jpeg")
ok=1
[[ "$code" == 404 || "$code" == 405 ]] && ok=0
hasil 'Tidak ada rute unggah foto untuk akun lain' "$code" "$code" "$ok"
code=$(req "$STAF" DELETE "/users/$MANAJER_ID/avatar")
ok=1
[[ "$code" == 404 || "$code" == 405 ]] && ok=0
hasil 'Tidak ada rute hapus foto untuk akun lain' "$code" "$code" "$ok"

echo
echo '== Jumlah task'
for jar in "$STAF" "$MANAJER"; do
  who=$([[ "$jar" == "$STAF" ]] && echo Staf || echo Manajer)
  code=$(req "$jar" GET /me/task-counts)
  cp "$TMP/body" "$TMP/counts"
  board=$(req "$jar" GET '/board?scope=mine')
  ok=0
  for s in requested on_progress ready completed; do
    a=$(grep -o "\"$s\":[0-9]*" "$TMP/counts" | cut -d: -f2)
    b=$(grep -o "\"status\":\"$s\",\"count\":[0-9]*" "$TMP/body" | grep -o '[0-9]*$')
    [[ -n "$a" && "$a" == "$b" ]] || ok=1
    printf '      %s %-12s kartu=%s papan=%s\n' "$who" "$s" "${a:-?}" "${b:-?}"
  done
  hasil "task-counts $who cocok dengan papan scope=mine (papan $board)" "$code" 200 "$ok"
done

echo
echo '== Audit'
code=$(req "$ADMIN" GET '/audit-logs?category=account')
n=$(grep -o '"action":"profile_updated"' "$TMP/body" | wc -l)
ok=1
[[ "$n" -ge 5 ]] && grep -q '"avatar":"diganti"' "$TMP/body" && grep -q '"avatar":"dihapus"' "$TMP/body" && ok=0
hasil "Audit profile_updated tercatat ($n baris di halaman 1)" "$code" 200 "$ok"

echo
json "$STAF" PUT /me/profile "{\"name\":\"$ORIG_NAME\",\"job_title\":\"$ORIG_JOB\"}" >/dev/null
echo "Nama dan jabatan Staf dikembalikan ke \"$ORIG_NAME\" / \"${ORIG_JOB:-kosong}\"."
echo "Lolos $pass, gagal $fail."
[[ "$fail" == 0 ]]
