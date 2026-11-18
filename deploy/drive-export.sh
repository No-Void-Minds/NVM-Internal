#!/usr/bin/env bash
# Salinan berkas aplikasi ke Google Drive akun tim (revisi 25). Dipanggil cron di VM setelah backup malam,
# oleh user yang ada di grup docker. Setiap run menulis satu baris ke $BACKUP_DIR/drive.log (OK atau GAGAL)
# dan keluar dengan kode bukan nol bila ada langkah yang gagal, pola yang sama dengan backup.sh.
#
# Tiga langkah, karena unggahannya dikerjakan container rclone di antaranya:
#   1. backend `drive-export stage`  menyusun pohon hardlink bernama rapi di folder staging, menulis ekspor
#      Excel bulan berjalan, dan mengenkripsi dump yang dikirim lewat pipe sebagai tar (age, kunci publik).
#   2. rclone copy staging ke Drive.
#   3. backend `drive-export record` mencatat drive_exports, menulis baris status, lalu membuang staging.
# Kalau langkah mana pun gagal, `drive-export fail` menulis baris status gagal supaya halaman Status (Admin)
# menunjukkannya, dan tidak ada baris drive_exports yang tercatat, jadi malam berikutnya dicoba lagi.
#
# Argumen opsional --dry-run: hanya langkah 1 dengan database tidak disentuh, lalu mencetak daftar path.
set -Eeuo pipefail
umask 077

deploy_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
set -a
# shellcheck disable=SC1091
source "$deploy_dir/.env"
set +a

compose=(docker compose --env-file "$deploy_dir/.env" -f "$deploy_dir/docker-compose.yml")
BACKUP_DIR="${BACKUP_DIR:-/srv/nvm-internal/backup}"
STAGING_DIR_HOST="${STAGING_DIR_HOST:-/srv/nvm-internal/drive-staging}"
SECRETS_DIR="${SECRETS_DIR:-/srv/nvm-internal/secrets}"
STORAGE_UID="${STORAGE_UID:-65532}"
# Versi dipatok, bukan latest: image yang berubah sendiri di tengah malam adalah pelajaran compro.
RCLONE_IMAGE="${RCLONE_IMAGE:-rclone/rclone:1.71.0}"
# Image kecil untuk satu langkah chown, sama dengan yang dipakai backup.sh.
ALPINE_IMAGE="${TAR_IMAGE:-alpine:3.22}"
RCLONE_REMOTE="${RCLONE_REMOTE:-nvmdrive}"
# Folder induk di Drive. rclone membuatnya sendiri bila belum ada.
RCLONE_DEST="${RCLONE_DEST:-NVM Internal}"
# Unggah 3 dump harian terakhir dan 1 per minggu selama 8 minggu.
DRIVE_DUMP_DAILY="${DRIVE_DUMP_DAILY:-3}"
DRIVE_DUMP_WEEKLY="${DRIVE_DUMP_WEEKLY:-8}"

dry_run=0
[[ "${1:-}" == "--dry-run" ]] && dry_run=1

mkdir -p "$BACKUP_DIR"
log="$BACKUP_DIR/drive.log"
err_file="$(mktemp)"
files="-"
size="-"

log_line() { printf '%s %s berkas=%s ukuran=%s%s\n' "$(date -Iseconds)" "$1" "$files" "$size" "${2:+ pesan=$2}" >>"$log"; }
last_error() { tr -s '\n\t' '  ' <"$err_file" | sed 's/ *$//'; }

fail() {
  trap - ERR
  local pesan="$1"
  # Baris status ditulis lewat backend supaya halaman Status menunjukkan kegagalan tanpa membuka log ini.
  "${compose[@]}" run --rm -T backend drive-export fail "$pesan" >/dev/null 2>&1 || true
  rm -f "$err_file" "$err_file.list"
  log_line GAGAL "$pesan"
  echo "salinan Drive GAGAL: $pesan" >&2
  exit 1
}
trap 'fail "perintah gagal di baris $LINENO: $(last_error)"' ERR

[[ -f "$SECRETS_DIR/rclone.conf" ]] || fail "$SECRETS_DIR/rclone.conf tidak ada, lihat docs/runbook-deploy.md bagian Salinan ke Google Drive"

# Folder staging harus bisa ditulis container yang berjalan sebagai UID pemilik storage, dan harus satu
# filesystem dengan storage supaya hardlink-nya tidak memakan disk.
mkdir -p "$STAGING_DIR_HOST"
if [[ "$(stat -c %u "$STAGING_DIR_HOST")" != "$STORAGE_UID" ]]; then
  docker run --rm --network none -v "$STAGING_DIR_HOST:/s" "$ALPINE_IMAGE" \
    chown "$STORAGE_UID:$STORAGE_UID" /s 2>"$err_file" || fail "folder staging tidak bisa disiapkan: $(last_error)"
fi

# Dump yang diunggah: 3 terbaru, ditambah satu per minggu (salinan weekly-) sebanyak 8.
mapfile -t dumps < <(
  {
    ls -t "$BACKUP_DIR"/db-*.dump 2>/dev/null | head -n "$DRIVE_DUMP_DAILY"
    ls -t "$BACKUP_DIR"/weekly-db-*.dump 2>/dev/null | head -n "$DRIVE_DUMP_WEEKLY"
  } | sort -u
)
if [[ -z "${DRIVE_BACKUP_AGE_RECIPIENT:-}" ]]; then
  echo "DRIVE_BACKUP_AGE_RECIPIENT kosong: dump tidak diunggah, hanya berkas aplikasi" >&2
  dumps=()
fi

# Langkah 1. Dump dialirkan sebagai tar lewat stdin, jadi tidak ada salinan plaintext tambahan di disk dan
# container ini tidak perlu bisa membaca folder backup.
stage_args=(stage)
[[ "$dry_run" == 1 ]] && stage_args+=(--dry-run)
if ((${#dumps[@]})); then
  tar -cf - -C "$BACKUP_DIR" "${dumps[@]##*/}" |
    "${compose[@]}" run --rm -T backend drive-export "${stage_args[@]}" >"$err_file.list" 2>"$err_file" ||
    fail "susun staging: $(last_error)"
else
  "${compose[@]}" run --rm -T backend drive-export "${stage_args[@]}" </dev/null >"$err_file.list" 2>"$err_file" ||
    fail "susun staging: $(last_error)"
fi
files="$(wc -l <"$err_file.list" | tr -d ' ')"
size="$(du -sh "$STAGING_DIR_HOST" 2>/dev/null | cut -f1 || echo -)"

if [[ "$dry_run" == 1 ]]; then
  echo "uji kering, tidak diunggah. $files path:"
  cat "$err_file.list"
  rm -f "$err_file" "$err_file.list"
  log_line OK "uji kering"
  exit 0
fi

# Langkah 2. rclone sebagai UID pemilik storage, supaya bisa membaca berkas 0600 di staging.
# Folder secrets di-mount sebagai folder (bukan satu berkas) dan baca-tulis, karena rclone menulis
# ulang rclone.conf setiap kali token OAuth diperbarui, dan penulisan itu lewat rename di dalam folder.
# --exclude manifest: berkas itu catatan internal, bukan isi arsip.
docker run --rm --user "$STORAGE_UID:$STORAGE_UID" -e XDG_CACHE_HOME=/tmp \
  -v "$STAGING_DIR_HOST:/staging:ro" -v "$SECRETS_DIR:/config/rclone" \
  "$RCLONE_IMAGE" copy /staging "$RCLONE_REMOTE:$RCLONE_DEST" \
  --config /config/rclone/rclone.conf \
  --transfers 2 --checkers 4 --exclude '.nvm-drive-manifest.json' --stats-one-line --stats 5m \
  2>"$err_file" || fail "rclone copy: $(last_error)"

# Langkah 3. Baru dicatat setelah unggahan berhasil.
"${compose[@]}" run --rm -T backend drive-export record >/dev/null 2>"$err_file" ||
  fail "catat hasil salinan: $(last_error)"

rm -f "$err_file" "$err_file.list"
log_line OK
