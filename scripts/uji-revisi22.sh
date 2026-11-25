#!/usr/bin/env bash
# Uji API revisi 22: PIC sub task mengembalikan task Ready ke On Progress dengan membuka lagi sub task miliknya.
# Jalankan dari folder backend setelah restart backend dan ../scripts/dev-login.sh:
#   API=http://127.0.0.1:8080 bash ../scripts/uji-revisi22.sh
# Setiap kali jalan, skrip membuat SATU TASK UJI BARU "Uji revisi 22" di database dev dan tidak menghapusnya
# (ID-nya dicetak di awal). Manajer PIC task, Staf memegang dua sub task (Naskah, Desain), Manajer memegang satu
# (Foto). Semua dicentang, task dipindah ke Ready, lalu Staf mengembalikannya dengan memilih Naskah.
# Bentuk respons: GET /api/tasks/:id membungkus isi di kunci "task"; galat berbentuk {"error":{"message":...}}.
# Penolakan dianggap lolos hanya bila kode DAN pesannya cocok, supaya penolakan karena JSON rusak ("Isian tidak
# terbaca.") tidak terhitung lolos. Bila satu nilai (id, version) gagal diambil, skrip berhenti.
set -uo pipefail

API=${API:-http://127.0.0.1:8080}
MGR=/tmp/manajer.txt
STAF=/tmp/staf.txt
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
pass=0
fail=0

hasil() {
  local name=$1 got=$2 want=$3
  if [[ "$got" == "$want" ]]; then
    printf 'PASS  %-58s %s\n' "$name" "$got"
    pass=$((pass + 1))
  else
    printf 'FAIL  %-58s %s (harapan %s)\n' "$name" "$got" "$want"
    fail=$((fail + 1))
  fi
}

# call COOKIE METHOD PATH [BODY]: body jawaban di $TMP/body, mencetak kode HTTP.
call() {
  local args=(-s -o "$TMP/body" -w '%{http_code}' -b "$1" -X "$2" "$API/api$3" -H "Origin: $API")
  if [[ $# -ge 4 ]]; then args+=(-H "Content-Type: application/json" --data "$4"); fi
  curl "${args[@]}"
}

# js EKSPRESI: membaca $TMP/body sebagai d dan mencetak hasil ekspresi Python. Body kosong atau bukan JSON
# mencetak string kosong, supaya `need` bisa menangkapnya. CR dibuang: Python di Windows mencetak CRLF, dan
# tanpa itu nilai terakhir yang dibaca `read` membawa CR (mis. id sub task "33" plus CR).
js() {
  py -3 -c "import json,sys
try:
    d=json.load(open(sys.argv[1], encoding='utf-8'))
    print($1)
except Exception:
    print('')" "$TMP/body" | tr -d '\r'
}

# need NAMA NILAI: hentikan skrip bila nilai yang dibutuhkan langkah berikutnya kosong.
need() {
  if [[ -z "$2" || "$2" == "None" ]]; then
    echo "BERHENTI: $1 kosong. Body terakhir: $(head -c 300 "$TMP/body")"
    echo "lolos $pass, gagal $fail"
    exit 1
  fi
}

# ok NAMA KODE HARAPAN: langkah persiapan; bila gagal, skrip berhenti karena langkah berikutnya bergantung padanya.
ok() {
  hasil "$1" "$2" "$3"
  if [[ "$2" != "$3" ]]; then
    echo "BERHENTI: langkah persiapan gagal. Body: $(head -c 300 "$TMP/body")"
    echo "lolos $pass, gagal $fail"
    exit 1
  fi
}

# tolak NAMA KODE HARAPAN POTONGAN_PESAN: kode harus cocok dan pesan galat harus memuat potongan itu.
tolak() {
  local msg
  msg=$(js "d['error']['message']")
  hasil "$1" "$2" "$3"
  if [[ "$msg" == *"$4"* ]]; then
    printf 'PASS  %-58s %s\n' "  pesan" "$msg"
    pass=$((pass + 1))
  else
    printf 'FAIL  %-58s "%s" (harapan memuat "%s")\n' "  pesan" "$msg" "$4"
    fail=$((fail + 1))
  fi
}

# detail COOKIE: GET /tasks/$ID ke $TMP/body; isinya di d['task'].
detail() {
  local code
  code=$(call "$1" GET "/tasks/$ID")
  need "GET /tasks/$ID (HTTP $code)" "$(js "d['task']['id']")"
}

call "$MGR" GET /auth/me >/dev/null
MGR_ID=$(js "d['user']['id']")
need "id Manajer (cookie /tmp/manajer.txt)" "$MGR_ID"
call "$STAF" GET /auth/me >/dev/null
STAF_ID=$(js "d['user']['id']")
need "id Staf (cookie /tmp/staf.txt)" "$STAF_ID"

code=$(call "$MGR" POST /tasks "{\"title\":\"Uji revisi 22\",\"urgency\":\"medium\",\"pic_ids\":[$MGR_ID],\"label_ids\":[],
  \"subtasks\":[{\"title\":\"Naskah\",\"pic_id\":$STAF_ID},{\"title\":\"Desain\",\"pic_id\":$STAF_ID},{\"title\":\"Foto\",\"pic_id\":$MGR_ID}]}")
ok "Manajer membuat task dengan tiga sub task" "$code" 201
ID=$(js "d['id']")
need "id task" "$ID"
echo "      task uji $ID"

detail "$MGR"
V=$(js "d['task']['version']")
need "version" "$V"
code=$(call "$MGR" POST "/tasks/$ID/move" "{\"version\":$V,\"to\":\"on_progress\",\"note\":\"\"}")
ok "Manajer memulai task" "$code" 200

detail "$MGR"
read -r S1 S2 S3 < <(js "' '.join(str(s['id']) for s in d['task']['subtasks'])")
need "id sub task Naskah" "${S1:-}"
need "id sub task Desain" "${S2:-}"
need "id sub task Foto" "${S3:-}"
hasil "Urutan sub task" "$(js "[s['title'] for s in d['task']['subtasks']]")" "['Naskah', 'Desain', 'Foto']"
for s in "$S1" "$S2"; do
  ok "Staf mencentang sub task $s" "$(call "$STAF" PUT "/tasks/$ID/subtasks/$s/done" '{"done":true}')" 204
done
ok "Manajer mencentang sub task $S3" "$(call "$MGR" PUT "/tasks/$ID/subtasks/$S3/done" '{"done":true}')" 204

detail "$MGR"
V=$(js "d['task']['version']")
need "version" "$V"
code=$(call "$MGR" POST "/tasks/$ID/move" "{\"version\":$V,\"to\":\"ready\",\"note\":\"\"}")
ok "Manajer menandai Ready" "$code" 200

detail "$STAF"
hasil "move_to Staf di Ready" "$(js "d['task']['can']['move_to']")" "['on_progress']"
hasil "revision_subtasks Staf" "$(js "[s['title'] for s in d['task']['revision_subtasks']]")" "['Naskah', 'Desain']"
V=$(js "d['task']['version']")
need "version" "$V"

code=$(call "$STAF" POST "/tasks/$ID/move" "{\"version\":$V,\"to\":\"on_progress\",\"note\":\"\",\"subtask_ids\":[]}")
tolak "Daftar sub task kosong ditolak" "$code" 400 "Pilih minimal satu sub task yang direvisi."
code=$(call "$STAF" POST "/tasks/$ID/move" "{\"version\":$V,\"to\":\"on_progress\",\"note\":\"\",\"subtask_ids\":[$S1,$S3]}")
tolak "Memilih sub task milik orang lain ditolak" "$code" 400 "Sub task $S3 tidak bisa dibuka lagi"
code=$(call "$STAF" POST "/tasks/$ID/move" "{\"version\":$((V - 1)),\"to\":\"on_progress\",\"note\":\"\",\"subtask_ids\":[$S1]}")
tolak "Version basi ditolak" "$code" 409 "sudah diubah"
code=$(call "$MGR" POST "/tasks/$ID/move" "{\"version\":$V,\"to\":\"on_progress\",\"note\":\"Revisi\",\"subtask_ids\":[$S1]}")
tolak "PIC task mengirim pilihan sub task ditolak" "$code" 400 "Pilihan sub task hanya dipakai PIC sub task"

# Penolakan di atas tidak boleh mengubah apa pun.
detail "$STAF"
hasil "Masih Ready setelah semua penolakan" "$(js "d['task']['status']")" "ready"
hasil "Version tidak berubah" "$(js "d['task']['version']")" "$V"

code=$(call "$STAF" POST "/tasks/$ID/move" "{\"version\":$V,\"to\":\"on_progress\",\"note\":\"Judul terlalu panjang\",\"subtask_ids\":[$S1]}")
hasil "Staf mengembalikan dengan Naskah saja" "$code" 200

detail "$STAF"
hasil "Status sesudahnya" "$(js "d['task']['status']")" "on_progress"
hasil "Centang Naskah, Desain, Foto" "$(js "[s['done'] for s in d['task']['subtasks']]")" "[False, True, True]"
hasil "Riwayat revisi sub task" \
  "$(js "[(h['new_value']['reopened_subtasks'][0]['title'], h['new_value'].get('reason'), 'note' in h['new_value']) for h in d['task']['history'] if (h.get('new_value') or {}).get('reopened_subtasks')]")" \
  "[('Naskah', 'Judul terlalu panjang', False)]"
hasil "move_to Staf sesudahnya" "$(js "d['task']['can']['move_to']")" "['ready']"
hasil "revision_subtasks Staf sesudahnya" "$(js "d['task']['revision_subtasks']")" "[]"

code=$(call "$MGR" GET "/board")
need "GET /board (HTTP $code)" "$(js "d['columns'][0]['status']")"
hasil "subtask_pics di papan" \
  "$(js "[[p['id'] for p in t['subtask_pics']] for c in d['columns'] for t in c['tasks'] if t['id']=='$ID']")" \
  "[[$STAF_ID, $MGR_ID]]"

echo "lolos $pass, gagal $fail"
[[ $fail -eq 0 ]]
