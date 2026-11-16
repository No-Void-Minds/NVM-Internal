#!/usr/bin/env bash
# Backup harian nvm_internal_prod dan /srv/nvm-internal/storage (S12-06, S12-11). Dipanggil cron di VM oleh user
# yang ada di grup docker. Setiap run menulis satu baris ke $BACKUP_DIR/backup.log (OK atau GAGAL) dan keluar
# dengan kode bukan nol bila ada langkah yang gagal, supaya kegagalan tidak diam-diam (pelajaran compro).
# Simpan 14 cadangan harian dan 8 mingguan (salinan hari Minggu).
set -Eeuo pipefail
# Dump berisi data akun dan isi task, jadi hanya pemiliknya yang boleh membaca.
umask 077

deploy_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
set -a
# shellcheck disable=SC1091
source "$deploy_dir/.env"
set +a

PG_CONTAINER="${PG_CONTAINER:-nvm-internal-postgres}"
DB_NAME="${DB_NAME:-nvm_internal_prod}"
DB_USER="${DB_USER:-nvm_internal_migrator}"
STORAGE_DIR_HOST="${STORAGE_DIR_HOST:-/srv/nvm-internal/storage}"
BACKUP_DIR="${BACKUP_DIR:-/srv/nvm-internal/backup}"
KEEP_DAILY="${KEEP_DAILY:-14}"
KEEP_WEEKLY="${KEEP_WEEKLY:-8}"
# UID user nonroot di image distroless backend. Berkas storage ditulis 0600 dan folder 0750 oleh UID ini, jadi
# user VM tidak bisa membacanya; arsip dibuat di container sementara yang berjalan sebagai UID yang sama.
STORAGE_UID="${STORAGE_UID:-65532}"
TAR_IMAGE="${TAR_IMAGE:-alpine:3.22}"

# Dicek sebelum trap dipasang, karena trap sendiri menulis log ke folder ini (pola doomsday).
mkdir -p "$BACKUP_DIR"
if [[ ! -w "$BACKUP_DIR" ]]; then
  echo "folder backup $BACKUP_DIR tidak bisa ditulis oleh $(id -un)" >&2
  exit 1
fi
chmod 700 "$BACKUP_DIR"

stamp="$(date -u +%Y%m%dT%H%M%SZ)"
log="$BACKUP_DIR/backup.log"
db_file="$BACKUP_DIR/db-$stamp.dump"
st_file="$BACKUP_DIR/storage-$stamp.tar.gz"
err_file="$(mktemp)"
db_size="-"
st_size="-"

log_line() { printf '%s %s db=%s storage=%s%s\n' "$(date -Iseconds)" "$1" "$db_size" "$st_size" "${2:+ pesan=$2}" >>"$log"; }
last_error() { tr -s '\n\t' '  ' <"$err_file" | sed 's/ *$//'; }

fail() {
  trap - ERR
  rm -f "$db_file.tmp" "$st_file.tmp" "$err_file"
  log_line GAGAL "$1"
  echo "backup GAGAL: $1" >&2
  exit 1
}
trap 'fail "perintah gagal di baris $LINENO: $(last_error)"' ERR

# Sisa run yang terputus tanpa sempat menjalankan trap, misalnya VM mati di tengah backup.
find "$BACKUP_DIR" -maxdepth 1 -name '*.tmp' -mtime +0 -delete

# Socket lokal di dalam container Postgres tidak butuh password. Format custom sudah terkompresi.
docker exec "$PG_CONTAINER" pg_dump -U "$DB_USER" -d "$DB_NAME" -Fc >"$db_file.tmp" 2>"$err_file" ||
  fail "pg_dump: $(last_error)"
# Ukuran file bukan bukti: dump yang terpotong ditolak saat daftar isinya dibaca (pola doomsday).
docker exec -i "$PG_CONTAINER" pg_restore --list <"$db_file.tmp" >/dev/null 2>"$err_file" ||
  fail "hasil dump tidak bisa dibaca pg_restore: $(last_error)"
mv "$db_file.tmp" "$db_file"
db_size="$(du -h "$db_file" | cut -f1)"

[[ -d "$STORAGE_DIR_HOST" ]] || fail "folder storage $STORAGE_DIR_HOST tidak ada"
# tar sebagai UID pemilik storage, baca-saja. Berkas yang tidak terbaca membuat tar keluar dengan kode bukan nol,
# jadi masalah izin tidak tersembunyi.
docker run --rm --network none --user "$STORAGE_UID:$STORAGE_UID" -v "$STORAGE_DIR_HOST:/data:ro" "$TAR_IMAGE" \
  tar -czf - -C /data . >"$st_file.tmp" 2>"$err_file" || fail "tar storage: $(last_error)"
# Arsip yang rusak ditolak sebelum menggantikan cadangan lama.
tar -tzf "$st_file.tmp" >/dev/null 2>"$err_file" || fail "arsip storage tidak bisa dibaca: $(last_error)"
mv "$st_file.tmp" "$st_file"
st_size="$(du -h "$st_file" | cut -f1)"

# Salinan mingguan hari Minggu, lewat hard link supaya tidak memakan ruang dua kali.
if [[ "$(date -u +%u)" == 7 ]]; then
  ln "$db_file" "$BACKUP_DIR/weekly-db-$stamp.dump"
  ln "$st_file" "$BACKUP_DIR/weekly-storage-$stamp.tar.gz"
fi

# Baru dihapus setelah backup hari ini terbukti baik. -mtime +13 berarti umur 14 hari atau lebih.
deleted=0
deleted=$((deleted + $(find "$BACKUP_DIR" -maxdepth 1 \( -name 'db-*.dump' -o -name 'storage-*.tar.gz' \) \
  -mtime +"$((KEEP_DAILY - 1))" -print -delete | wc -l)))
deleted=$((deleted + $(find "$BACKUP_DIR" -maxdepth 1 \( -name 'weekly-db-*.dump' -o -name 'weekly-storage-*.tar.gz' \) \
  -mtime +"$((KEEP_WEEKLY * 7 - 1))" -print -delete | wc -l)))

rm -f "$err_file"
log_line OK "dihapus=$deleted"
