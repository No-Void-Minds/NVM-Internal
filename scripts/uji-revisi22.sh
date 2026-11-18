#!/usr/bin/env bash
# Uji API revisi 22: PIC sub task mengembalikan task Ready ke On Progress dengan membuka lagi sub task miliknya.
# Jalankan dari folder backend setelah restart backend dan ../scripts/dev-login.sh:
#   API=http://127.0.0.1:8080 bash ../scripts/uji-revisi22.sh
# Membuat satu task uji di database dev: Manajer PIC task, Staf memegang dua sub task (Naskah, Desain), Manajer
# memegang satu (Foto). Semua dicentang, task dipindah ke Ready, lalu Staf mengembalikannya dengan memilih Naskah.
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

# js EKSPRESI: membaca $TMP/body sebagai d dan mencetak hasil ekspresi Python.
js() { py -3 -c "import json,sys; d=json.load(open(sys.argv[1], encoding='utf-8')); print($1)" "$TMP/body"; }

call "$MGR" GET /auth/me >/dev/null && MGR_ID=$(js "d['user']['id']")
call "$STAF" GET /auth/me >/dev/null && STAF_ID=$(js "d['user']['id']")

code=$(call "$MGR" POST /tasks "{\"title\":\"Uji revisi 22\",\"urgency\":\"medium\",\"pic_ids\":[$MGR_ID],\"label_ids\":[],
  \"subtasks\":[{\"title\":\"Naskah\",\"pic_id\":$STAF_ID},{\"title\":\"Desain\",\"pic_id\":$STAF_ID},{\"title\":\"Foto\",\"pic_id\":$MGR_ID}]}")
hasil "Manajer membuat task dengan tiga sub task" "$code" 201
ID=$(js "d['id']")
echo "      task uji $ID"

ver() { call "$1" GET "/tasks/$ID" >/dev/null && js "d['version']"; }
code=$(call "$MGR" POST "/tasks/$ID/move" "{\"version\":$(ver "$MGR"),\"to\":\"on_progress\",\"note\":\"\"}")
hasil "Manajer memulai task" "$code" 200

call "$MGR" GET "/tasks/$ID" >/dev/null
read -r S1 S2 S3 < <(js "' '.join(str(s['id']) for s in d['subtasks'])")
for s in "$S1" "$S2"; do
  code=$(call "$STAF" PUT "/tasks/$ID/subtasks/$s/done" '{"done":true}')
  hasil "Staf mencentang sub task $s" "$code" 204
done
code=$(call "$MGR" PUT "/tasks/$ID/subtasks/$S3/done" '{"done":true}')
hasil "Manajer mencentang sub task $S3" "$code" 204
code=$(call "$MGR" POST "/tasks/$ID/move" "{\"version\":$(ver "$MGR"),\"to\":\"ready\",\"note\":\"\"}")
hasil "Manajer menandai Ready" "$code" 200

call "$STAF" GET "/tasks/$ID" >/dev/null
hasil "move_to Staf di Ready" "$(js "d['can']['move_to']")" "['on_progress']"
hasil "revision_subtasks Staf" "$(js "[s['title'] for s in d['revision_subtasks']]")" "['Naskah', 'Desain']"
V=$(js "d['version']")

code=$(call "$STAF" POST "/tasks/$ID/move" "{\"version\":$V,\"to\":\"on_progress\",\"note\":\"\",\"subtask_ids\":[]}")
hasil "Daftar sub task kosong ditolak" "$code" 400
code=$(call "$STAF" POST "/tasks/$ID/move" "{\"version\":$V,\"to\":\"on_progress\",\"note\":\"\",\"subtask_ids\":[$S1,$S3]}")
hasil "Memilih sub task milik orang lain ditolak" "$code" 400
echo "      pesan: $(js "d.get('error',{}).get('message', d)")"
code=$(call "$STAF" POST "/tasks/$ID/move" "{\"version\":$((V - 1)),\"to\":\"on_progress\",\"note\":\"\",\"subtask_ids\":[$S1]}")
hasil "Version basi ditolak" "$code" 409
code=$(call "$MGR" POST "/tasks/$ID/move" "{\"version\":$V,\"to\":\"on_progress\",\"note\":\"Revisi\",\"subtask_ids\":[$S1]}")
hasil "PIC task mengirim pilihan sub task ditolak" "$code" 400

code=$(call "$STAF" POST "/tasks/$ID/move" "{\"version\":$V,\"to\":\"on_progress\",\"note\":\"Judul terlalu panjang\",\"subtask_ids\":[$S1]}")
hasil "Staf mengembalikan dengan Naskah saja" "$code" 200

call "$STAF" GET "/tasks/$ID" >/dev/null
hasil "Status sesudahnya" "$(js "d['status']")" "on_progress"
hasil "Centang Naskah, Desain, Foto" "$(js "[s['done'] for s in d['subtasks']]")" "[False, True, True]"
hasil "Riwayat revisi sub task" \
  "$(js "[(h['new_value']['reopened_subtasks'][0]['title'], h['new_value'].get('reason')) for h in d['history'] if (h.get('new_value') or {}).get('reopened_subtasks')]")" \
  "[('Naskah', 'Judul terlalu panjang')]"
hasil "move_to Staf sesudahnya" "$(js "d['can']['move_to']")" "['ready']"

call "$MGR" GET "/board" >/dev/null
hasil "subtask_pics di papan" \
  "$(js "[[p['id'] for p in t['subtask_pics']] for c in d['columns'] for t in c['tasks'] if t['id']=='$ID']")" \
  "[[$STAF_ID, $MGR_ID]]"

echo "lolos $pass, gagal $fail"
[[ $fail -eq 0 ]]
