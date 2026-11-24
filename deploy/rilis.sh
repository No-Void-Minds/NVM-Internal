#!/usr/bin/env bash
# Rilis versi baru di VM produksi (/srv/nvm-internal, user nvmadmin). Menggantikan blok "Update" manual di
# docs/runbook-deploy.md bagian 10, yang kini hanya untuk keadaan darurat.
#
#   ./deploy/rilis.sh            rilis origin/main bila CI-nya lolos
#   ./deploy/rilis.sh --uji      cek 1 sampai 4 saja, langkah 5 sampai 11 hanya ditampilkan
#   ./deploy/rilis.sh --kembali  kembalikan image :sebelum
#
# Tidak pernah menjalankan migrate down atau force, dan tidak punya opsi untuk melewati cek CI.
# Jangan tambahkan set -x: skrip ini memegang MIGRATOR_URL berisi password.
set -euo pipefail

REPO_API=https://api.github.com/repos/No-Void-Minds/NVM-Internal
# Harus sama dengan nama job di .github/workflows/ci.yml.
CI_JOB=pra-deploy
MIGRATE_IMAGE=migrate/migrate:v4.18.3
DB_NAME=nvm_internal_prod
DB_OWNER=nvm_internal_migrator
IMAGES=(nvm-internal-backend nvm-internal-frontend)

deploy_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_dir="$(dirname "$deploy_dir")"
log_file="$deploy_dir/rilis.log"
lock_file="$deploy_dir/.rilis.lock"

# Diisi selama rilis, dibaca catat_akhir saat skrip keluar.
rilis_mulai=0
langkah=0
hasil=""
old=""
target=""
mig_log="-"

mask() { sed -E 's#postgres(ql)?://[^[:space:]"]*#postgres://***#g'; }
die() {
  echo "GAGAL: $*" >&2
  exit 1
}
compose() { docker compose --env-file deploy/.env -f deploy/docker-compose.yml "$@"; }

konfirmasi() {
  local jawab=""
  read -r -p "Ketik ya untuk lanjut: " jawab || true
  [ "$jawab" = ya ]
}

muat_env() {
  [ -f deploy/.env ] || die "deploy/.env tidak ada."
  # Isinya tidak dicetak; hanya variabelnya yang dipakai.
  set -a
  # shellcheck disable=SC1091
  . deploy/.env
  set +a
  local v
  for v in MIGRATOR_URL PG_NETWORK PG_CONTAINER; do
    [ -n "${!v:-}" ] || die "$v kosong di deploy/.env."
  done
}

migrate_cmd() {
  docker run --rm --network "$PG_NETWORK" -v "$PWD/backend/migrations:/migrations:ro" \
    "$MIGRATE_IMAGE" -path=/migrations -database "$MIGRATOR_URL" "$@"
}

# Versi migrasi produksi sebagai angka. Dirty atau tidak terbaca menghentikan skrip: keadaan itu harus diperiksa
# manusia, bukan di-force.
versi_db() {
  local out v
  out=$(migrate_cmd version 2>&1) || {
    printf '%s\n' "$out" | mask >&2
    die "Versi migrasi produksi tidak terbaca."
  }
  v=$(printf '%s\n' "$out" | mask | tail -n 1)
  [[ $v =~ ^[0-9]+$ ]] || die "Database produksi tidak bersih: versi migrasi \"$v\". Periksa manual, jangan force."
  printf '%s' "$v"
}

sehat() {
  local port=${NVM_PORT:-8094} batas=$((SECONDS + 60)) body
  while [ "$SECONDS" -lt "$batas" ]; do
    # Ditampung dulu, bukan dipipa ke grep -q: dengan pipefail, curl yang kena SIGPIPE membuat hasil sehat jadi gagal.
    body=$(curl -fsS --max-time 5 "http://127.0.0.1:$port/api/health" 2>/dev/null) || body=""
    if [[ $body == *'"status":"ok"'* ]]; then
      return 0
    fi
    sleep 3
  done
  return 1
}

cek_image_sebelum() {
  local img
  for img in "${IMAGES[@]}"; do
    docker image inspect "$img:sebelum" >/dev/null 2>&1 || die "Image $img:sebelum tidak ada, tidak ada yang bisa dikembalikan."
  done
}

pulihkan_image() {
  local img
  for img in "${IMAGES[@]}"; do docker tag "$img:sebelum" "$img:latest"; done
  compose up -d --no-build
}

catat() { printf '%s\t%s\t%s\t%s\t%s\n' "$(date -Iseconds)" "$1" "$2" "$3" "$4" >>"$log_file"; }

catat_akhir() {
  [ "$rilis_mulai" = 1 ] || return 0
  [ -n "$hasil" ] || hasil="GAGAL di langkah $langkah"
  catat "$old" "$target" "$mig_log" "$hasil"
}

repo_bersih() {
  git diff --quiet && git diff --cached --quiet ||
    die "Ada perubahan lokal pada berkas yang dilacak git di VM. Rapikan dulu (git status)."
}

kembali() {
  # Entri rilis terakhir, bukan entri --kembali.
  local entri="" mig="-" baru="" lama="" hasil_lalu=""
  [ -f "$log_file" ] && entri=$(awk -F'\t' '$5 !~ /^KEMBALI/' "$log_file" | tail -n 1)
  if [ -n "$entri" ]; then
    IFS=$'\t' read -r _ lama baru mig hasil_lalu <<<"$entri"
    echo "Rilis terakhir di log: ${lama:0:12} ke ${baru:0:12}, migrasi $mig, hasil $hasil_lalu"
    if [ "$mig" != "-" ]; then
      echo "PERINGATAN: rilis itu menjalankan migrasi $mig. Image lama belum tentu cocok dengan skema baru, dan"
      echo "skema tidak ikut dikembalikan. Kalau ragu, pulihkan dump backup sebelum rilis itu (runbook bagian 8)."
    fi
    if [ "$hasil_lalu" != BERHASIL ]; then
      echo "Catatan: rilis itu tidak berakhir BERHASIL, jadi image :sebelum mungkin sudah yang sedang jalan."
    fi
  else
    echo "Belum ada rilis di $log_file."
  fi
  cek_image_sebelum
  echo "Yang akan dilakukan: tag ${IMAGES[*]} :sebelum menjadi :latest, lalu compose up -d --no-build."
  konfirmasi || die "Dibatalkan, tidak ada yang berubah."

  local head
  head=$(git rev-parse HEAD)
  pulihkan_image
  # Repo ikut mundur supaya sama dengan yang jalan, dan rilis berikutnya melihat origin/main sebagai hal baru.
  if [ "$hasil_lalu" = BERHASIL ] && [ "$baru" = "$head" ] && [ -n "$lama" ]; then
    repo_bersih
    git reset --hard --quiet "$lama"
    echo "Repo dikembalikan ke ${lama:0:12}."
  else
    echo "Repo tetap di ${head:0:12} (tidak cocok dengan entri rilis terakhir di log)."
  fi
  if sehat; then
    catat "$head" "$(git rev-parse HEAD)" "-" "KEMBALI, sehat"
    echo "Versi sebelumnya jalan lagi dan sehat."
  else
    catat "$head" "$(git rev-parse HEAD)" "-" "KEMBALI, tidak sehat"
    die "Image lama sudah dipasang tetapi health check gagal 60 detik. Periksa: docker compose --env-file deploy/.env -f deploy/docker-compose.yml logs --tail 100"
  fi
}

gagal_rilis() {
  if [ "$mig_log" = "-" ]; then
    echo "Health check gagal. Rilis ini tanpa migrasi, jadi image lama dipulihkan." >&2
    pulihkan_image
    git reset --hard --quiet "$old"
    if sehat; then
      hasil="DIBATALKAN, versi lama dipulihkan"
      echo "Rilis dibatalkan, versi lama dipulihkan." >&2
    else
      hasil="DIBATALKAN, versi lama tidak sehat"
      echo "Rilis dibatalkan, image lama dipasang, tetapi health check-nya juga gagal. Periksa log container." >&2
    fi
    exit 1
  fi
  hasil="GAGAL setelah migrasi, perlu tindakan manual"
  cat >&2 <<EOF
Rilis gagal setelah migrasi $mig_log. Tidak ada yang dibalik otomatis. Jangan jalankan migrate down.
Repo di ${target:0:12}, database di versi baru, image :sebelum masih ada.
Langkah manual:
1. Cari penyebab: docker compose --env-file deploy/.env -f deploy/docker-compose.yml logs --tail 100 backend frontend
2. Kalau perbaikannya di kode: push perbaikan ke main, tunggu CI lolos, jalankan ./deploy/rilis.sh lagi.
3. Kalau migrasinya hanya menambah tabel atau kolom, image lama biasanya masih jalan dengan skema baru:
   ./deploy/rilis.sh --kembali untuk sementara.
4. Kalau data atau skema harus dikembalikan: pulihkan dump backup sebelum rilis ini (runbook bagian 8),
   lalu ./deploy/rilis.sh --kembali.
EOF
  exit 1
}

main() {
  local mode=rilis
  case "${1:-}" in
  "") ;;
  --uji) mode=uji ;;
  --kembali) mode=kembali ;;
  *) die "Opsi tidak dikenal: $1. Pakai tanpa opsi, --uji, atau --kembali." ;;
  esac
  [ $# -le 1 ] || die "Terlalu banyak argumen."

  cd "$repo_dir"
  exec 9>"$lock_file"
  flock -n 9 || die "rilis.sh lain sedang berjalan."
  muat_env
  [ "$(git symbolic-ref --short HEAD 2>/dev/null || true)" = main ] || die "Repo di VM tidak sedang di branch main."

  if [ "$mode" = kembali ]; then
    kembali
    return
  fi

  repo_bersih
  echo "1. Mengambil origin/main"
  git fetch --quiet origin main
  old=$(git rev-parse HEAD)
  target=$(git rev-parse origin/main)
  if [ "$old" = "$target" ]; then
    echo "Tidak ada yang baru."
    return 0
  fi
  git merge-base --is-ancestor "$old" "$target" ||
    die "HEAD ${old:0:12} bukan leluhur origin/main ${target:0:12}, tidak bisa fast-forward. Periksa manual."

  echo "2. Status CI ${target:0:12}"
  local json pesan
  json=$(curl -fsSL --max-time 20 -H 'Accept: application/vnd.github+json' -H 'X-GitHub-Api-Version: 2022-11-28' \
    "$REPO_API/commits/$target/check-runs?check_name=$CI_JOB&filter=all&per_page=100") ||
    die "API GitHub tidak bisa dihubungi atau menolak (tanpa token batasnya 60 permintaan per jam). Coba lagi nanti."
  pesan=$(printf '%s' "$json" | python3 deploy/cek-ci.py "$target" "$CI_JOB") || die "$pesan"
  echo "   $pesan"

  echo "3. Migrasi produksi"
  local cur
  cur=$(versi_db)
  echo "   versi $cur, tidak dirty"

  echo "4. Ringkasan"
  local migs sql_berubah=0 grants=0
  migs=$(git ls-tree --name-only "$target" backend/migrations/ |
    awk -F/ -v c="$cur" '$NF ~ /\.up\.sql$/ { n = $NF; sub(/_.*/, "", n); if (n + 0 > c + 0) print $NF }')
  git diff --quiet "$old" "$target" -- deploy/sql || sql_berubah=1
  [ -z "$migs" ] || mig_log=$(printf '%s\n' "$migs" | sed 's/\.up\.sql$//' | paste -sd, -)
  # Hak DELETE diberikan per tabel, jadi migrasi yang menambah tabel butuh 10-grants.sql walaupun deploy/sql tidak
  # berubah. Berkas itu aman diulang, maka dijalankan untuk setiap rilis yang membawa migrasi.
  if [ "$sql_berubah" = 1 ] || [ -n "$migs" ]; then grants=1; fi
  echo "   Commit yang akan naik:"
  git log --oneline "$old..$target" | sed 's/^/     /'
  if [ -n "$migs" ]; then
    echo "   Migrasi baru:"
    printf '%s\n' "$migs" | sed 's/^/     /'
  else
    echo "   Migrasi baru: tidak ada"
  fi
  if [ "$sql_berubah" = 1 ]; then echo "   deploy/sql berubah: ya"; else echo "   deploy/sql berubah: tidak"; fi
  if ! git diff --quiet "$old" "$target" -- deploy/.env.example; then
    echo "   PERHATIAN: deploy/.env.example berubah. Cek variabel baru di deploy/.env sebelum lanjut."
  fi

  if [ "$mode" = uji ]; then
    echo
    echo "Mode --uji: langkah berikut tidak dijalankan."
    echo "5. ./deploy/backup.sh, baris terakhir backup.log harus OK"
    echo "6. Tag ${IMAGES[*]} :latest menjadi :sebelum"
    echo "7. git merge --ff-only ${target:0:12}"
    if [ -n "$migs" ]; then echo "8. migrate up ($mig_log), lalu cek versi tidak dirty"; else echo "8. Tanpa migrasi"; fi
    if [ "$grants" = 1 ]; then echo "   lalu deploy/sql/10-grants.sql sebagai $DB_OWNER"; fi
    echo "9. docker compose up -d --build"
    echo "10. Health check http://127.0.0.1:${NVM_PORT:-8094}/api/health sampai 60 detik"
    if [ -n "$migs" ]; then
      echo "11. Bila gagal: tidak ada yang dibalik (ada migrasi), langkah manual ditampilkan"
    else
      echo "11. Bila gagal: image :sebelum dipulihkan dan repo kembali ke ${old:0:12}"
    fi
    return 0
  fi

  konfirmasi || die "Dibatalkan, tidak ada yang berubah."
  rilis_mulai=1
  trap catat_akhir EXIT

  langkah=5
  echo "5. Backup"
  ./deploy/backup.sh || die "backup.sh gagal. Rilis dihentikan sebelum ada yang berubah."
  tail -n 1 "${BACKUP_DIR:-$repo_dir/backup}/backup.log" | grep -qE '^[^ ]+ OK ' ||
    die "Baris terakhir backup.log bukan OK. Rilis dihentikan sebelum ada yang berubah."

  langkah=6
  echo "6. Tag image :sebelum"
  local img
  for img in "${IMAGES[@]}"; do
    docker image inspect "$img:latest" >/dev/null 2>&1 || die "Image $img:latest tidak ada."
    docker tag "$img:latest" "$img:sebelum"
  done

  langkah=7
  echo "7. Memperbarui repo"
  # Commit yang CI-nya dicek, bukan git pull: pull bisa membawa commit yang masuk sesudah cek di langkah 2.
  git merge --ff-only --quiet "$target"

  langkah=8
  if [ -n "$migs" ]; then
    echo "8. Migrasi ($mig_log)"
    local out
    out=$(migrate_cmd up 2>&1) || {
      printf '%s\n' "$out" | mask >&2
      hasil="GAGAL migrasi, perlu tindakan manual"
      die "Migrasi gagal. Container lama masih jalan. Jangan migrate down atau force; periksa versi dan isi migrasi."
    }
    printf '%s\n' "$out" | mask | sed 's/^/   /'
    local akhir terbaru
    akhir=$(versi_db)
    terbaru=$(printf '%s\n' "$migs" | tail -n 1 | sed -E 's/^0*([0-9]+)_.*/\1/')
    [ "$akhir" = "$terbaru" ] || die "Versi migrasi produksi $akhir, seharusnya $terbaru."
    echo "   versi $akhir, tidak dirty"
  else
    echo "8. Tanpa migrasi"
  fi
  if [ "$grants" = 1 ]; then
    echo "   10-grants.sql"
    docker cp deploy/sql/10-grants.sql "$PG_CONTAINER":/tmp/10-grants.sql
    docker exec "$PG_CONTAINER" psql -U "$DB_OWNER" -d "$DB_NAME" -v ON_ERROR_STOP=1 -f /tmp/10-grants.sql ||
      die "10-grants.sql gagal. Container lama masih jalan; jalankan ulang grants manual (runbook bagian 3)."
    docker exec "$PG_CONTAINER" rm /tmp/10-grants.sql
  fi

  langkah=9
  echo "9. Build dan nyalakan"
  if ! compose up -d --build; then
    echo "docker compose up gagal." >&2
    gagal_rilis
  fi

  langkah=10
  echo "10. Health check"
  sehat || gagal_rilis

  hasil=BERHASIL
  echo "Rilis ${old:0:12} ke ${target:0:12} berhasil."
}

# Semua kerja ada di dalam main: git merge bisa mengganti berkas ini saat sedang berjalan, dan bash membaca skrip
# sedikit demi sedikit. Fungsi sudah terbaca utuh sebelum dijalankan, jadi isi baru tidak ikut tereksekusi.
main "$@"
exit
