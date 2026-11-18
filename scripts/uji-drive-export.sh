#!/usr/bin/env bash
# Uji perilaku deploy/drive-export.sh di laptop, tanpa Docker dan tanpa VM (revisi 25). `docker` diganti stub
# yang meniru dua hal yang pernah menipu skrip ini: baris progres Compose di stderr, dan log perintah Go yang
# juga di stderr (dulu di stdout, sehingga daftar path tercemar dan pesan galatnya hilang).
#
# Lima kasus: folder backup tanpa berkas weekly-*, tanpa db-* sama sekali, keduanya ada, kegagalan di langkah
# stage, dan kegagalan tanpa pesan apa pun. Yang diperiksa:
#   - skrip tidak gagal hanya karena pola glob tidak cocok,
#   - daftar path di stdout tidak pernah memuat baris log JSON,
#   - pesan galat asli dari perintah Go muncul utuh di stderr dan di drive.log, tanpa baris Compose.
#
#   bash scripts/uji-drive-export.sh
set -uo pipefail

repo="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
base="$(mktemp -d)"
bin="$base/bin"
mkdir -p "$bin"
gagal_total=0

cat >"$bin/docker" <<'STUB'
#!/usr/bin/env bash
# Stub docker. STUB_FAIL=stage membuat langkah stage gagal seperti perintah Go yang galat, dan
# STUB_FAIL=sunyi membuatnya gagal tanpa pesan apa pun.
if [[ "${1:-}" == "run" ]]; then exit 0; fi   # docker run ... chown atau rclone
if [[ "${1:-}" == "compose" ]]; then
  shift
  args=("$@")
  for a in "${args[@]}"; do
    [[ "$a" == version ]] && exit 0
  done
  # Compose menulis progres ke stderr. Ini yang dulu tersimpan sebagai "pesan galat".
  echo " Container nvm-internal-app-backend-run-abc123  Creating" >&2
  echo " Container nvm-internal-app-backend-run-abc123  Created" >&2
  if [[ "${args[*]}" == *"drive-export fail"* ]]; then exit 0; fi
  if [[ "${args[*]}" == *"drive-export stage"* ]]; then
    # Isi tar yang dikirim skrip, untuk memeriksa pemilihan dump. Stdin kosong (</dev/null) tidak apa-apa.
    if [[ ! -t 0 ]]; then tar -tf - 2>/dev/null | sed 's/^/TAR /' >&2 || true; fi
    case "${STUB_FAIL:-}" in
      stage)
        # Log perintah Go: JSON, dan sekarang di stderr, bukan stdout.
        echo '{"time":"2026-11-18T10:12:59Z","level":"ERROR","msg":"salinan Drive gagal","err":"folder staging lama tidak bisa dibuang: unlinkat /data/drive-staging: read-only file system"}' >&2
        exit 1
        ;;
      sunyi) exit 1 ;;
    esac
    echo '{"time":"2026-11-18T10:13:10Z","level":"INFO","msg":"staging siap","berkas_baru":2}' >&2
    echo 'Task/2026-11/T-0012 Foto katalog/Brief/brief.pdf'
    echo 'Aktivitas/2026-11 November/Aktivitas Harian November 2026.xlsx'
    exit 0
  fi
  exit 0
fi
exit 0
STUB
chmod +x "$bin/docker"

periksa() {
  local nama="$1" kondisi="$2" pesan="$3"
  if eval "$kondisi"; then
    echo "  OK   $pesan"
  else
    echo "  GAGAL $pesan"
    gagal_total=$((gagal_total + 1))
  fi
}

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
    echo "STORAGE_DIR_HOST=$run/storage"
    echo "SECRETS_DIR=$run/secrets"
    echo "DRIVE_BACKUP_AGE_RECIPIENT=age1contoh"
  } >"$run/deploy/.env"
  if [[ "$daily" == dobel ]]; then
    dump_bertanggal_sama "$run/backup"
  elif [[ "$daily" == 1 ]]; then
    echo isi >"$run/backup/db-20261118T200000Z.dump"
    echo isi >"$run/backup/db-20261117T200000Z.dump"
  fi
  [[ "$weekly" == 1 ]] && echo isi >"$run/backup/weekly-db-20261115T200000Z.dump"

  echo "=== $nama (db-*: $daily, weekly-*: $weekly, gagal: ${gagal:-tidak}) ==="
  PATH="$bin:$PATH" STUB_FAIL="$gagal" bash "$run/deploy/drive-export.sh" --dry-run \
    >"$run/stdout.txt" 2>"$run/stderr.txt"
  local kode=$?
  local log="$(tail -1 "$run/backup/drive.log" 2>/dev/null || echo '')"

  if [[ -z "$gagal" ]]; then
    periksa "$nama" "[[ $kode == 0 ]]" "keluar dengan kode 0 (dapat $kode)"
    periksa "$nama" "! grep -q '^{' '$run/stdout.txt'" "daftar path di stdout bebas baris log JSON"
    periksa "$nama" "grep -q 'Brief/brief.pdf' '$run/stdout.txt'" "daftar path tercetak"
    periksa "$nama" "[[ '$log' == *OK* ]]" "drive.log mencatat OK"
    if [[ "$daily" == dobel ]]; then
      local terkirim tanggal_unik
      terkirim="$(grep -c '^TAR ' "$run/stderr.txt" || true)"
      # Tanggal diambil dari nama berkas: db-20261118T... dan weekly-db-20261118T... sama tanggalnya.
      tanggal_unik="$(grep '^TAR ' "$run/stderr.txt" | grep -o '[0-9]\{8\}T' | sort -u | wc -l | tr -d ' ')"
      periksa "$nama" "[[ '$terkirim' == '$tanggal_unik' ]]" "dump yang dikirim satu per tanggal ($terkirim berkas, $tanggal_unik tanggal)"
      periksa "$nama" "[[ '$terkirim' == 2 ]]" "dua tanggal dump yang dikirim (dapat $terkirim)"
      periksa "$nama" "grep -q 'TAR db-20261118T210000Z.dump' '$run/stderr.txt'" "dump terbaru di tanggal itu yang dipakai"
      periksa "$nama" "! grep -q 'TAR db-20261118T200000Z.dump' '$run/stderr.txt'" "dump lama di tanggal yang sama tidak ikut"
    fi
  else
    periksa "$nama" "[[ $kode == 1 ]]" "keluar dengan kode 1 (dapat $kode)"
    periksa "$nama" "! grep -q 'Creating' '$run/stderr.txt'" "pesan galat bebas baris progres Compose"
    periksa "$nama" "[[ '$log' == *GAGAL* ]]" "drive.log mencatat GAGAL"
    if [[ "$gagal" == stage ]]; then
      periksa "$nama" "grep -q 'read-only file system' '$run/stderr.txt'" "pesan galat asli perintah Go muncul utuh"
      periksa "$nama" "[[ '$log' == *'read-only file system'* ]]" "pesan galat asli masuk drive.log"
    fi
  fi
  echo
}

# Dump bertanggal sama: tiga kali backup di hari yang sama, plus satu dump mingguan yang bertanggal sama
# dengan salah satu dump harian. Nama di Drive dibuat per tanggal, jadi yang terkirim harus satu per tanggal.
dump_bertanggal_sama() {
  local dir="$1"
  echo isi1 >"$dir/db-20261118T200000Z.dump"
  sleep 0.1
  echo isi2 >"$dir/db-20261118T203000Z.dump"
  sleep 0.1
  echo isi3 >"$dir/db-20261118T210000Z.dump"   # terbaru di tanggal 18
  echo isi4 >"$dir/db-20261117T200000Z.dump"
  echo isi5 >"$dir/weekly-db-20261117T200000Z.dump"
}

jalankan tanpa-weekly 0 1
jalankan tanpa-dump-apa-pun 0 0
jalankan keduanya-ada 1 1
jalankan dump-bertanggal-sama 0 dobel
jalankan stage-gagal 1 1 stage
jalankan stage-gagal-tanpa-pesan 1 1 sunyi

rm -rf "$base"
if [[ "$gagal_total" != 0 ]]; then
  echo "$gagal_total pemeriksaan gagal"
  exit 1
fi
echo "semua pemeriksaan lolos"
