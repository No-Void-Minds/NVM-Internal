#!/usr/bin/env bash
# Uji perilaku deploy/drive-export.sh di laptop, tanpa Docker dan tanpa VM (revisi 25). `docker` diganti stub
# yang meniru keluaran Compose, termasuk baris "Creating/Created" ke stderr yang dulu menutupi pesan galat
# sungguhan. Empat kasus: folder backup tanpa berkas weekly-*, tanpa db-* sama sekali, keduanya ada, dan satu
# kegagalan buatan di langkah stage. Yang diperiksa: skrip tidak gagal hanya karena pola glob tidak cocok,
# dan pesan galat asli dari perintah Go muncul utuh di stderr dan di drive.log.
#
#   bash scripts/uji-drive-export.sh "$PWD"
set -uo pipefail

repo="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
base="$(mktemp -d)"
bin="$base/bin"
mkdir -p "$bin"

cat >"$bin/docker" <<'STUB'
#!/usr/bin/env bash
# Stub docker. STUB_FAIL=stage membuat langkah drive-export stage gagal seperti perintah Go yang galat.
if [[ "${1:-}" == "run" ]]; then exit 0; fi   # docker run ... chown
if [[ "${1:-}" == "compose" ]]; then
  shift
  args=("$@")
  for a in "${args[@]}"; do
    case "$a" in
      version) exit 0 ;;
    esac
  done
  # Compose menulis progres ke stderr, persis ini yang tadi mencemari pesan galat.
  echo " Container nvm-internal-app-backend-run-abc123  Creating" >&2
  echo " Container nvm-internal-app-backend-run-abc123  Created" >&2
  if [[ "${args[*]}" == *"drive-export fail"* ]]; then exit 0; fi
  if [[ "${args[*]}" == *"drive-export stage"* ]]; then
    if [[ "${STUB_FAIL:-}" == "stage" ]]; then
      echo 'galat: DRIVE_BACKUP_AGE_RECIPIENT tidak valid: malformed recipient' >&2
      exit 1
    fi
    echo 'Task/2026-11/T-0012 Foto katalog/Brief/brief.pdf'
    echo 'Aktivitas/2026-11 November/Aktivitas Harian November 2026.xlsx'
    exit 0
  fi
  exit 0
fi
exit 0
STUB
chmod +x "$bin/docker"

jalankan() {
  local nama="$1" weekly="$2" daily="$3" gagal="${4:-}"
  local run="$base/$nama"
  mkdir -p "$run/backup" "$run/staging" "$run/secrets" "$run/deploy"
  : >"$run/secrets/rclone.conf"
  cp "$repo/deploy/drive-export.sh" "$run/deploy/"
  cp "$repo/deploy/docker-compose.yml" "$run/deploy/" 2>/dev/null || true
  {
    echo "BACKUP_DIR=$run/backup"
    echo "STAGING_DIR_HOST=$run/staging"
    echo "SECRETS_DIR=$run/secrets"
    echo "DRIVE_BACKUP_AGE_RECIPIENT=age1contoh"
  } >"$run/deploy/.env"
  [[ "$daily" == 1 ]] && { echo isi >"$run/backup/db-20261118T200000Z.dump"; echo isi >"$run/backup/db-20261117T200000Z.dump"; }
  [[ "$weekly" == 1 ]] && echo isi >"$run/backup/weekly-db-20261115T200000Z.dump"

  echo "=== $nama (db-*: $daily, weekly-*: $weekly, gagal: ${gagal:-tidak}) ==="
  PATH="$bin:$PATH" STUB_FAIL="$gagal" bash "$run/deploy/drive-export.sh" --dry-run
  local kode=$?
  echo "exit=$kode"
  echo "log: $(tail -1 "$run/backup/drive.log" 2>/dev/null || echo '(tidak ada)')"
  echo
}

jalankan tanpa-weekly 0 1
jalankan tanpa-dump-apa-pun 0 0
jalankan keduanya-ada 1 1
jalankan stage-gagal 1 1 stage

rm -rf "$base"
