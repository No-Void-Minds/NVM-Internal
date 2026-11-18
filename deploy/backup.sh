#!/usr/bin/env bash
# Backup harian nvm_internal_prod dan /srv/nvm-internal/storage (S12-06, S12-11). Dipanggil cron di VM oleh user
# yang ada di grup docker. Setiap run menulis satu baris ke $BACKUP_DIR/backup.log (OK atau GAGAL) dan keluar
# dengan kode bukan nol bila ada langkah yang gagal, supaya kegagalan tidak diam-diam (pelajaran compro).
# Simpan 14 cadangan harian dan 8 mingguan (salinan hari Minggu).
#
# Storage disalin sebagai snapshot inkremental berbasis hardlink di $BACKUP_DIR/storage-snapshots/<stamp>
# (revisi 24). Sebelumnya setiap run membuat tar.gz seluruh storage, jadi satu lampiran 95 MB tersalin ulang
# 14 kali. Sekarang berkas yang tidak berubah hanya menambah satu hardlink, yang tidak memakan blok disk baru.
# Arsip tar.gz lama dibiarkan dan hilang sendiri lewat retensi yang sama.
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
# Folder snapshot storage. Isinya dimiliki STORAGE_UID supaya container yang membaca storage bisa menulis
# ke sini; mode 0755 supaya user VM tetap bisa menelusuri dan menghitung ukurannya dengan du.
SNAP_DIR="${SNAP_DIR:-$BACKUP_DIR/storage-snapshots}"

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
err_file="$(mktemp)"
db_size="-"
st_size="-"
snap_total="-"

# storage= adalah blok baru snapshot hari ini (du menghitung hardlink sekali), total= seluruh folder snapshot.
log_line() { printf '%s %s db=%s storage=%s total=%s%s\n' "$(date -Iseconds)" "$1" "$db_size" "$st_size" "$snap_total" "${2:+ pesan=$2}" >>"$log"; }
last_error() { tr -s '\n\t' '  ' <"$err_file" | sed 's/ *$//'; }

fail() {
  trap - ERR
  rm -f "$db_file.tmp" "$err_file"
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
mkdir -p "$SNAP_DIR"
# Folder snapshot harus bisa ditulis container yang berjalan sebagai STORAGE_UID. Dibetulkan hanya bila perlu,
# jadi run berikutnya tidak lagi memanggil container root.
if [[ "$(stat -c %u "$SNAP_DIR")" != "$STORAGE_UID" ]]; then
  docker run --rm --network none -v "$SNAP_DIR:/snap" "$TAR_IMAGE" \
    sh -c "chown $STORAGE_UID:$STORAGE_UID /snap && chmod 0755 /snap" 2>"$err_file" ||
    fail "folder snapshot tidak bisa disiapkan: $(last_error)"
fi

# Satu container, sebagai UID pemilik storage, storage baca-saja:
#   1. snapshot sebelumnya dikloning dengan hardlink (cp -al), jadi belum ada blok baru yang dipakai,
#   2. berkas yang belum ada disalin dari storage, berkas yang sudah tidak ada di storage dibuang,
#   3. retensi dijalankan di sini juga, karena folder snapshot dimiliki UID ini, bukan user VM.
# rsync --link-dest akan lebih singkat, tetapi tidak ada di image alpine dan memasangnya butuh jaringan,
# sedangkan job ini sengaja berjalan dengan --network none. Berkas storage tidak pernah diubah di tempat
# (nama berkas adalah UUID barisnya), jadi menyalin yang baru dan membuang yang hilang sudah setara.
docker run --rm --network none --user "$STORAGE_UID:$STORAGE_UID" \
  -v "$STORAGE_DIR_HOST:/data:ro" -v "$SNAP_DIR:/snap" "$TAR_IMAGE" \
  sh -eu -c '
    stamp="$1"; keep_daily="$2"; keep_weekly="$3"; weekly="$4"
    baru="/snap/$stamp"
    rm -rf "$baru.tmp"
    prev="$(ls -1d /snap/*/ 2>/dev/null | grep -v "\.tmp/$" | tail -1 || true)"
    if [ -n "$prev" ]; then cp -al "${prev%/}" "$baru.tmp"; else mkdir -p "$baru.tmp"; fi
    cd /data
    find . -type f | while read -r f; do
      if [ -e "$baru.tmp/$f" ]; then continue; fi
      mkdir -p "$baru.tmp/$(dirname "$f")"
      cp -p "$f" "$baru.tmp/$f"
    done
    cd "$baru.tmp"
    find . -type f | while read -r f; do [ -e "/data/$f" ] || rm -f "$f"; done
    find . -type d -empty -delete 2>/dev/null || true
    mkdir -p "$baru.tmp"
    # if, bukan "[ ... ] && ...": dengan set -e, tes yang gagal di awal daftar && mengakhiri skrip.
    if [ "$weekly" = 7 ]; then : >"$baru.tmp/.weekly"; fi
    # Jumlah berkas harus sama dengan storage, kalau tidak snapshot ini tidak dipakai.
    ada="$(find /data -type f | wc -l)"
    salin="$(find "$baru.tmp" -type f ! -name .weekly | wc -l)"
    [ "$ada" = "$salin" ] || { echo "jumlah berkas beda: storage $ada, snapshot $salin" >&2; exit 1; }
    mv "$baru.tmp" "$baru"
    # Folder kerja tadi sudah tidak ada setelah mv; tanpa pindah dulu, find di bawah gagal membaca cwd.
    cd /
    # Retensi: snapshot harian dibuang setelah keep_daily hari, yang bertanda .weekly setelah keep_weekly pekan.
    for d in /snap/*/; do
      d="${d%/}"
      [ -d "$d" ] || continue
      if [ "$d" = "$baru" ]; then continue; fi
      if [ -e "$d/.weekly" ]; then batas="$keep_weekly"; else batas="$keep_daily"; fi
      # Hanya hapus bila find benar-benar mengembalikan folder itu. Hasil kosong karena galat tidak boleh
      # terbaca sebagai "sudah tua".
      if [ "$(find "$d" -maxdepth 0 -mtime +"$((batas - 1))" | wc -l)" = 1 ]; then rm -rf "$d"; fi
    done
  ' sh "$stamp" "$KEEP_DAILY" "$((KEEP_WEEKLY * 7))" "$(date -u +%u)" 2>"$err_file" ||
  fail "snapshot storage: $(last_error)"

# du menghitung hardlink sekali, jadi angka ini adalah blok baru yang benar-benar dipakai snapshot hari ini.
st_size="$(du -sh --exclude='.weekly' "$SNAP_DIR/$stamp" | cut -f1)"

# Salinan mingguan hari Minggu, lewat hard link supaya tidak memakan ruang dua kali. Snapshot storage tidak
# perlu disalin: penandanya ada di dalam folder snapshot itu sendiri (.weekly).
if [[ "$(date -u +%u)" == 7 ]]; then
  ln "$db_file" "$BACKUP_DIR/weekly-db-$stamp.dump"
fi

# Baru dihapus setelah backup hari ini terbukti baik. -mtime +13 berarti umur 14 hari atau lebih.
deleted=0
deleted=$((deleted + $(find "$BACKUP_DIR" -maxdepth 1 \( -name 'db-*.dump' -o -name 'storage-*.tar.gz' \) \
  -mtime +"$((KEEP_DAILY - 1))" -print -delete | wc -l)))
deleted=$((deleted + $(find "$BACKUP_DIR" -maxdepth 1 \( -name 'weekly-db-*.dump' -o -name 'weekly-storage-*.tar.gz' \) \
  -mtime +"$((KEEP_WEEKLY * 7 - 1))" -print -delete | wc -l)))

snap_total="$(du -sh "$SNAP_DIR" 2>/dev/null | cut -f1 || echo -)"
snapshots="$(find "$SNAP_DIR" -mindepth 1 -maxdepth 1 -type d | wc -l)"

rm -f "$err_file"
log_line OK "dihapus=$deleted snapshot=$snapshots"
