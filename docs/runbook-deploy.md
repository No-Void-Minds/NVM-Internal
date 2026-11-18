# Runbook deploy NVM-Internal

Dijalankan Rifky di VM `nvm-server` (192.168.68.69, Ubuntu 26.04) sebagai user yang ada di grup `docker`. Hasil akhirnya: `https://novoidminds.xyz` lewat Cloudflare Tunnel `nvm-server`, berdampingan dengan compro (`127.0.0.1:8092`) dan doomsday (`127.0.0.1:8090`), memakai Postgres bersama di container `nvm-internal-postgres`.

| Hal | Nilai |
|---|---|
| Folder repo | `/srv/nvm-internal` |
| Berkas unggahan | `/srv/nvm-internal/storage`, milik UID dan GID **65532** (user `nonroot` di image distroless) |
| Cadangan | `/srv/nvm-internal/backup`, milik user VM, izin 700 |
| Port di host | `127.0.0.1:8094` (Nginx); backend tanpa port di host |
| Jaringan Docker | `172.30.94.0/24`, gateway `172.30.94.1` (compro `.92`, doomsday `.93`) |
| Database | `nvm_internal_prod`, role `nvm_internal_migrator` (pemilik, migrasi) dan `nvm_internal_app` (aplikasi) |

Superuser `nvm_internal` hanya dipakai sekali di langkah 2, tidak pernah oleh aplikasi. Database dev `nvm_internal` dan `nvm_internal_test` tidak disentuh.

## 1. Folder dan repo

`git clone` gagal kalau `/srv/nvm-internal` sudah ada dan berisi. Periksa isinya dulu; folder kosong dihapus dengan `rmdir` (yang menolak menghapus folder berisi, jadi aman). Folder berisi jangan dihapus sebelum jelas isinya apa.

```bash
ls -la /srv/nvm-internal 2>/dev/null
sudo rmdir /srv/nvm-internal                    # hanya bila kosong
sudo git clone https://github.com/No-Void-Minds/NVM-Internal /srv/nvm-internal
sudo chown -R "$USER": /srv/nvm-internal
cd /srv/nvm-internal

# Berkas unggahan ditulis backend sebagai UID 65532 (0600 untuk berkas, 0750 untuk folder).
sudo mkdir -p /srv/nvm-internal/storage
sudo chown -R 65532:65532 /srv/nvm-internal/storage
sudo chmod 750 /srv/nvm-internal/storage

mkdir -p -m 700 /srv/nvm-internal/backup
```

Kalau ada hard disk data terpisah (S14-01), mount ke `/srv/nvm-internal/storage` sebelum `chown`, lalu cek dengan `df -h /srv/nvm-internal/storage`.

## 2. Database, role, dan secret

Buat dua password dengan `openssl rand -hex 32` (hex tidak perlu di-URL-encode). Simpan di pengelola password; jangan ditulis di chat atau tiket.

```bash
openssl rand -hex 32   # password nvm_internal_migrator
openssl rand -hex 32   # password nvm_internal_app
openssl rand -hex 32   # JWT_SECRET
```

Buat role dan database. Password diminta interaktif, jadi tidak tertinggal di riwayat shell:

```bash
docker exec nvm-internal-postgres printenv POSTGRES_USER          # harus nvm_internal
docker cp deploy/sql/00-bootstrap.sql nvm-internal-postgres:/tmp/00-bootstrap.sql
docker exec -it nvm-internal-postgres psql -U nvm_internal -d postgres -f /tmp/00-bootstrap.sql
docker exec nvm-internal-postgres rm /tmp/00-bootstrap.sql
```

Baris terakhir keluaran skrip harus `public | nvm_internal_migrator`. Kalau pemiliknya lain, migrate nanti gagal "permission denied for schema public": template database di cluster ini hasil restore, jadi skema `public` di database baru tidak otomatis milik pemilik database. Perbaiki di dalam `nvm_internal_prod` (bukan di database `postgres`):

```bash
docker exec -it nvm-internal-postgres psql -U nvm_internal -d nvm_internal_prod \
  -c 'ALTER SCHEMA public OWNER TO nvm_internal_migrator' -c 'REVOKE CREATE ON SCHEMA public FROM PUBLIC'
```

Isi `deploy/.env`:

```bash
cp deploy/.env.example deploy/.env
chmod 600 deploy/.env
nano deploy/.env    # ganti setiap GANTI: DATABASE_URL (app), MIGRATOR_URL (migrator), JWT_SECRET
```

Cek nama jaringan Postgres sama dengan `PG_NETWORK` (`nvm-internal_default`):

```bash
docker inspect nvm-internal-postgres --format '{{range $k, $v := .NetworkSettings.Networks}}{{$k}} {{end}}'
```

Cek isi `.env` sebelum langkah 3, tanpa mencetak secret. Panjang `JWT_SECRET` harus 64, dan password di URL disamarkan:

```bash
set -a; source deploy/.env; set +a
echo "panjang JWT_SECRET: ${#JWT_SECRET}"                          # harus 64
echo "$MIGRATOR_URL" | sed -E 's#(://[^:]+:)[^@]*@#\1****@#'      # user harus nvm_internal_migrator
echo "$DATABASE_URL" | sed -E 's#(://[^:]+:)[^@]*@#\1****@#'      # user harus nvm_internal_app
```

**Setiap kali `deploy/.env` diubah:** source ulang (`set -a; source deploy/.env; set +a`) atau buka sesi SSH baru, lalu buat ulang container dengan `--force-recreate`. Variabel yang sudah di-export di shell mengalahkan isi `--env-file`, dan compose tidak membuat ulang container hanya karena isi env berubah.

```bash
docker compose --env-file deploy/.env -f deploy/docker-compose.yml up -d --force-recreate
```

## 3. Migrasi (A7)

Sama dengan compro: image `migrate/migrate` di jaringan Postgres, sebagai `nvm_internal_migrator`. Setelah setiap migrasi yang menambah tabel, jalankan `10-grants.sql`, karena hak DELETE diberikan per tabel.

```bash
cd /srv/nvm-internal
set -a; source deploy/.env; set +a

docker run --rm --network "$PG_NETWORK" -v "$PWD/backend/migrations:/migrations:ro" \
  migrate/migrate:v4.18.3 -path=/migrations -database "$MIGRATOR_URL" up
docker run --rm --network "$PG_NETWORK" -v "$PWD/backend/migrations:/migrations:ro" \
  migrate/migrate:v4.18.3 -path=/migrations -database "$MIGRATOR_URL" version   # harus 7

docker cp deploy/sql/10-grants.sql "$PG_CONTAINER":/tmp/10-grants.sql
docker exec "$PG_CONTAINER" psql -U nvm_internal_migrator -d nvm_internal_prod -f /tmp/10-grants.sql
docker exec "$PG_CONTAINER" rm /tmp/10-grants.sql
```

Hasil query terakhir di `10-grants.sql` harus tepat enam tabel: `audit_logs`, `subtask_assignees`, `task_assignees`, `task_labels`, `task_subtasks`, `user_sessions`.

Cek bahwa role aplikasi tidak bisa mengubah skema (harus `permission denied`):

```bash
docker exec "$PG_CONTAINER" psql -U nvm_internal_app -d nvm_internal_prod -c 'CREATE TABLE coba (x int)'
```

## 4. Build dan nyalakan

Build pertama terjadi di VM (PC kantor tidak punya Docker). Dockerfile backend menggagalkan build bila binary tidak statis.

```bash
cd /srv/nvm-internal
docker compose --env-file deploy/.env -f deploy/docker-compose.yml up -d --build
docker compose --env-file deploy/.env -f deploy/docker-compose.yml ps        # frontend: healthy
curl -s http://127.0.0.1:8094/api/health                                     # {"database":"ok","status":"ok"}
curl -sI http://127.0.0.1:8094/ | grep -iE '^(content-security-policy|x-frame-options|cache-control):'
```

Karena `COOKIE_SECURE=true`, login lewat `http://127.0.0.1:8094` tidak bisa (browser tidak mengirim cookie Secure lewat HTTP). Login diuji lewat domain di langkah 7.

## 5. Akun Admin pertama

`seed` membuat divisi Marketing dan Creative serta akun Admin dengan password sementara (wajib diganti saat login pertama, minimal 6 karakter). Aman dijalankan berulang: akun yang sudah ada tidak diubah.

Di produksi akun Admin pertama bernama `adminit`.

```bash
P="$(openssl rand -base64 18)"; echo "Password sementara adminit: $P"
docker compose --env-file deploy/.env -f deploy/docker-compose.yml run --rm \
  --entrypoint /usr/local/bin/seed -e SEED_ADMIN_USERNAME=adminit -e SEED_ADMIN_PASSWORD="$P" backend
unset P
```

## 6. Cloudflare

Keadaan setelah deploy 2026-11-17 (Zero Trust, Networks, Tunnels):

| Tunnel | Mesin | Route |
|---|---|---|
| `nvm-server` | VM 192.168.68.69 | `novoidminds.xyz` ke `http://127.0.0.1:8094` |
| `nvm-tim-pc` | PC tim 192.168.68.68 | `dash.novoidminds.xyz` dan `studio.novoidminds.xyz` ke `http://127.0.0.1:8088` (Caddy, forward_auth ke `https://novoidminds.xyz/api/auth/verify`) |

- Tulis `127.0.0.1`, bukan `localhost`, karena port hanya terikat di IPv4.
- Jangan isi HTTP Host Header (Additional application settings, HTTP Settings). Backend membandingkan Origin dengan Host asli (S12-15).

**HTTPS.** SSL/TLS, Edge Certificates: Always Use HTTPS menyala, HSTS `max-age` 6 bulan (tanpa preload).

**Rate limit login (A10).** Security, WAF, Rate limiting rules, satu aturan (paket Free):

- Ekspresi: `(http.host eq "novoidminds.xyz" and http.request.uri.path eq "/api/auth/login" and http.request.method eq "POST")`
- Per IP, 5 permintaan per 10 detik, aksi Block selama 10 detik (batas paket Free).
- Aturan ini lapis luar. Di backend masih ada limiter per IP asli (10 percobaan, lalu 1 per 30 detik) dan kuncian akun setelah 5 kali gagal.

## 7. Cek setelah hidup

1. Dari HP dengan **data seluler** (bukan Wi-Fi kantor), buka `https://novoidminds.xyz`, login sebagai `adminit`, ganti password sementara.
2. DevTools, Application, Cookies: `nvm_session` punya Domain `.novoidminds.xyz`, Secure, HttpOnly, SameSite Lax, kedaluwarsa sekitar 9 jam.
3. Setelah login, buka `https://novoidminds.xyz/api/auth/verify`: tab Network menunjukkan 204 dan header `X-NVM-User: adminit`, tanpa `Set-Cookie`.
4. Audit Log: baris "Masuk" dari HP mencatat IP seluler, bukan `172.30.94.1` atau `127.0.0.1`.
5. IP dari tunnel, di VM:

   ```bash
   curl -s -o /dev/null -H 'CF-Connecting-IP: 203.0.113.7' http://127.0.0.1:8094/api/health
   docker compose --env-file deploy/.env -f deploy/docker-compose.yml logs --tail=1 frontend   # mencatat 203.0.113.7
   ```

   Dari komputer lain di LAN, `http://192.168.68.69:8094` harus gagal tersambung (port hanya di 127.0.0.1).
6. Console browser bersih dari galat Content-Security-Policy di halaman Papan, Arsip, Profil (unggah foto), Pengguna, dan detail task (pratinjau lampiran gambar dan PDF, yang memakai `blob:`).
7. `https://novoidminds.xyz/api/board` tanpa login: 401 dengan `Cache-Control: no-store`.

## 8. Backup, restore, dan job harian

Cron milik user VM (`crontab -e`). VM memakai UTC; jadwal dibuat setelah compro dan doomsday (19.00 dan 19.30 UTC):

```cron
# 20.00 UTC = 03.00 WIB: backup database dan storage (14 harian, 8 mingguan)
0 20 * * * /srv/nvm-internal/deploy/backup.sh >>/srv/nvm-internal/backup/cron.log 2>&1
# 20.30 UTC = 03.30 WIB: sesi kedaluwarsa, audit log login lebih dari 1 tahun, berkas yatim
30 20 * * * cd /srv/nvm-internal && docker compose --env-file deploy/.env -f deploy/docker-compose.yml run --rm -T backend maintenance >>/srv/nvm-internal/backup/maintenance.log 2>&1
```

Sebelum memasang cron, jalankan sekali dengan tangan:

```bash
/srv/nvm-internal/deploy/backup.sh && tail -1 /srv/nvm-internal/backup/backup.log      # OK db=... storage=...
docker compose --env-file deploy/.env -f deploy/docker-compose.yml run --rm -T backend maintenance --dry-run
```

`backup.log` berisi satu baris per run (OK atau GAGAL dan alasannya). Skrip keluar dengan kode bukan nol bila gagal. Berkas storage dibaca lewat container `alpine` sebagai UID 65532, jadi tidak perlu mengubah izin folder.

### Uji restore pertama (S14-04)

Catat jam mulai dan selesai; targetnya di bawah 4 jam.

```bash
cd /srv/nvm-internal
set -a; source deploy/.env; set +a
DUMP="$(ls -t backup/db-*.dump | head -1)"; ARSIP="$(ls -t backup/storage-*.tar.gz | head -1)"
date

# 1. Database kosong untuk uji, dimiliki migrator.
docker exec "$PG_CONTAINER" psql -U nvm_internal -d postgres -c 'CREATE DATABASE nvm_internal_restore OWNER nvm_internal_migrator'
docker exec -i "$PG_CONTAINER" pg_restore -U nvm_internal_migrator -d nvm_internal_restore --no-owner --exit-on-error <"$DUMP"

# 2. Bandingkan jumlah baris dengan produksi.
for db in nvm_internal_prod nvm_internal_restore; do
  docker exec "$PG_CONTAINER" psql -U nvm_internal_migrator -d "$db" -tAc \
    "SELECT '$db', (SELECT count(*) FROM tasks), (SELECT count(*) FROM users), (SELECT count(*) FROM task_attachments)"
done

# 3. Storage ke folder uji, lalu bandingkan jumlah berkas.
mkdir -p /tmp/restore-storage
tar -xzf "$ARSIP" -C /tmp/restore-storage
find /tmp/restore-storage -type f | wc -l
sudo find /srv/nvm-internal/storage -type f | wc -l
date
```

Setelah dicatat, bersihkan database uji (`DROP DATABASE nvm_internal_restore` sebagai `nvm_internal`) dan `/tmp/restore-storage`.

Pemulihan sungguhan memakai langkah yang sama, dengan target `nvm_internal_prod` yang dikosongkan dulu dan storage diekstrak ke `/srv/nvm-internal/storage`, lalu `sudo chown -R 65532:65532 /srv/nvm-internal/storage`.

### Salinan di luar VM (manual)

- Hard disk eksternal: salin `/srv/nvm-internal/backup` mingguan dengan `rsync -a`.
- PC IT: tarik lewat `scp` atau `rsync` dari VM.
- Laptop: hanya dalam bentuk terenkripsi, misalnya `gpg --symmetric --cipher-algo AES256 db-....dump` atau di volume BitLocker atau VeraCrypt. Dump berisi data akun dan isi task.

## 9. Impor Excel (Sprint 12 Bagian B, setelah pemetaan dari Manajer)

Impor memakai `nvm_internal_migrator`, karena `setval` pada nomor task tidak diizinkan untuk role aplikasi. Jalankan `--dry-run` dulu, cocokkan laporannya dengan Manajer, baru impor sungguhan. Impor menolak jalan bila masih ada baris yang ditolak.

```bash
cd /srv/nvm-internal
set -a; source deploy/.env; set +a
chmod 644 /tmp/impor/berkas.xlsx /tmp/impor/pemetaan-nama.csv   # dibaca UID 65532 di container
docker compose --env-file deploy/.env -f deploy/docker-compose.yml run --rm -T \
  --entrypoint /usr/local/bin/import-excel -e DATABASE_URL="$MIGRATOR_URL" \
  -v /tmp/impor:/impor:ro backend --file /impor/berkas.xlsx --map /impor/pemetaan-nama.csv --dry-run
```

Template pemetaan: `deploy/import/pemetaan-nama.example.csv` (kolom `nama_excel,username`).

## 10. Update versi dan rollback

Update:

```bash
cd /srv/nvm-internal
./deploy/backup.sh && tail -1 backup/backup.log                       # harus OK
docker tag nvm-internal-backend:latest nvm-internal-backend:sebelum
docker tag nvm-internal-frontend:latest nvm-internal-frontend:sebelum
git pull
set -a; source deploy/.env; set +a
docker run --rm --network "$PG_NETWORK" -v "$PWD/backend/migrations:/migrations:ro" \
  migrate/migrate:v4.18.3 -path=/migrations -database "$MIGRATOR_URL" up
# Bila ada tabel baru: jalankan lagi 10-grants.sql (langkah 3). Migrasi 000008 menambah task_assignees, jadi wajib.
# Migrasi 000013 (revisi 16, tenggat sub task) dan 000014 (revisi 18, catatan hasil) hanya menambah kolom,
# jadi tidak perlu grants ulang.
# Migrasi 000015 (revisi 19, task berulang) menambah tabel task_series: 10-grants.sql WAJIB dijalankan ulang
# sesudahnya, lalu cek daftar DELETE di keluarannya tetap enam tabel (task_series tidak boleh ada di sana).
docker compose --env-file deploy/.env -f deploy/docker-compose.yml up -d --build
curl -s http://127.0.0.1:8094/api/health
```

Rollback kode (tanpa migrasi baru): kembalikan image sebelumnya, tanpa build.

```bash
docker tag nvm-internal-backend:sebelum nvm-internal-backend:latest
docker tag nvm-internal-frontend:sebelum nvm-internal-frontend:latest
docker compose --env-file deploy/.env -f deploy/docker-compose.yml up -d --no-build
git checkout <commit sebelumnya>     # supaya repo sama dengan yang jalan
```

Rollback bila migrasi baru ternyata bermasalah: jangan `migrate down` di produksi tanpa membaca isi berkas `.down.sql` (bisa menghapus data). Pilihan yang lebih aman adalah memulihkan dump dari backup sebelum update (bagian 8), lalu kembalikan image.
