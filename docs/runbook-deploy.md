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

Hasil query terakhir di `10-grants.sql` harus tepat delapan tabel: `activity_attachments`, `audit_logs`, `subtask_assignees`, `task_assignees`, `task_labels`, `task_subtasks`, `user_sessions`, `user_todos`.

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
6. Console browser bersih dari galat Content-Security-Policy di halaman Papan, Arsip, Profil (unggah foto), Pengguna, dan detail task (pratinjau lampiran gambar dan PDF, yang memakai `blob:`), serta dialog sel di Aktivitas yang Output-nya tautan file Drive (thumbnail dari `drive.google.com` dan `*.googleusercontent.com`).
7. `https://novoidminds.xyz/api/board` tanpa login: 401 dengan `Cache-Control: no-store`.

## 8. Backup, restore, dan job harian

Cron milik user VM (`crontab -e`). VM memakai UTC; jadwal dibuat setelah compro dan doomsday (19.00 dan 19.30 UTC):

```cron
# 20.00 UTC = 03.00 WIB: backup database dan storage (14 harian, 8 mingguan)
0 20 * * * /srv/nvm-internal/deploy/backup.sh >>/srv/nvm-internal/backup/cron.log 2>&1
# 20.30 UTC = 03.30 WIB: sesi kedaluwarsa, audit log login lebih dari 1 tahun, berkas yatim
30 20 * * * cd /srv/nvm-internal && docker compose --env-file deploy/.env -f deploy/docker-compose.yml run --rm -T backend maintenance >>/srv/nvm-internal/backup/maintenance.log 2>&1
# 20.45 UTC = 03.45 WIB: salinan berkas ke Google Drive (revisi 25). Baris ini baru dipasang setelah langkah
# persiapan di bagian "Salinan ke Google Drive" di bawah selesai.
45 20 * * * /srv/nvm-internal/deploy/drive-export.sh >>/srv/nvm-internal/backup/drive-cron.log 2>&1
```

Sebelum memasang cron, jalankan sekali dengan tangan:

```bash
/srv/nvm-internal/deploy/backup.sh && tail -1 /srv/nvm-internal/backup/backup.log      # OK db=... storage=...
docker compose --env-file deploy/.env -f deploy/docker-compose.yml run --rm -T backend maintenance --dry-run
```

`backup.log` berisi satu baris per run (OK atau GAGAL dan alasannya). Skrip keluar dengan kode bukan nol bila gagal. Berkas storage dibaca lewat container `alpine` sebagai UID 65532, jadi tidak perlu mengubah izin folder.

#### Snapshot storage inkremental (revisi 24)

Dump database tetap seperti sebelumnya. Storage tidak lagi di-tar setiap hari, tetapi disalin jadi snapshot di `backup/storage-snapshots/<stamp>/`:

- snapshot baru dikloning dari snapshot sebelumnya dengan hardlink, lalu hanya berkas baru yang benar-benar disalin dan berkas yang sudah hilang dari storage dibuang,
- berkas yang tidak berubah tidak memakan blok disk lagi, jadi lampiran 95 MB tidak tersalin 14 kali,
- retensi sama dengan sebelumnya: 14 snapshot harian, dan snapshot hari Minggu (bertanda berkas `.weekly` di dalamnya) disimpan 8 pekan,
- semuanya dikerjakan satu container `alpine` sebagai UID 65532 dengan `--network none` dan storage di-mount baca-saja. `rsync --link-dest` tidak dipakai karena rsync tidak ada di image itu dan memasangnya butuh jaringan.

Arsip `storage-*.tar.gz` lama **tidak dihapus**. Snapshot berjalan di sampingnya, dan arsip lama hilang sendiri karena aturan retensi harian yang sudah ada.

Baris log menjadi `db=... storage=... total=...`: `storage` adalah blok baru snapshot hari ini, `total` seluruh folder snapshot. Keduanya dihitung `du`, yang menghitung hardlink sekali.

Uji di VM, jalankan dua kali berturut-turut:

```bash
/srv/nvm-internal/deploy/backup.sh && /srv/nvm-internal/deploy/backup.sh
tail -2 /srv/nvm-internal/backup/backup.log
cd /srv/nvm-internal/backup/storage-snapshots

# Jumlah berkas harus sama di kedua snapshot (dan sama dengan storage).
for d in */; do echo "$d $(find "$d" -type f ! -name .weekly | wc -l)"; done
sudo find /srv/nvm-internal/storage -type f | wc -l

# Ukuran: tiap snapshot "terlihat" penuh, tetapi totalnya hampir sama dengan satu snapshot saja,
# karena du menghitung hardlink sekali.
du -sh */
du -sh .

# Bukti hardlink: jumlah tautan sebuah berkas harus 2 atau lebih (satu per snapshot).
sudo stat -c '%h %n' "$(sudo find . -type f ! -name .weekly | head -1)"
```

Kalau `du -sh .` kira-kira sebesar satu snapshot, bukan dua, snapshot inkrementalnya bekerja.

### Uji restore pertama (S14-04)

Catat jam mulai dan selesai; targetnya di bawah 4 jam.

```bash
cd /srv/nvm-internal
set -a; source deploy/.env; set +a
DUMP="$(ls -t backup/db-*.dump | head -1)"; SNAP="$(ls -1d backup/storage-snapshots/*/ | tail -1)"
date

# 1. Database kosong untuk uji, dimiliki migrator.
docker exec "$PG_CONTAINER" psql -U nvm_internal -d postgres -c 'CREATE DATABASE nvm_internal_restore OWNER nvm_internal_migrator'
docker exec -i "$PG_CONTAINER" pg_restore -U nvm_internal_migrator -d nvm_internal_restore --no-owner --exit-on-error <"$DUMP"

# 2. Bandingkan jumlah baris dengan produksi.
for db in nvm_internal_prod nvm_internal_restore; do
  docker exec "$PG_CONTAINER" psql -U nvm_internal_migrator -d "$db" -tAc \
    "SELECT '$db', (SELECT count(*) FROM tasks), (SELECT count(*) FROM users), (SELECT count(*) FROM task_attachments)"
done

# 3. Storage dari snapshot ke folder uji, lalu bandingkan jumlah berkas.
mkdir -p /tmp/restore-storage
sudo cp -a "$SNAP." /tmp/restore-storage/
sudo find /tmp/restore-storage -type f ! -name .weekly | wc -l
sudo find /srv/nvm-internal/storage -type f | wc -l
date
```

Snapshot adalah folder biasa, jadi satu berkas bisa diambil sendiri tanpa mengekstrak apa pun:

```bash
sudo cp -a backup/storage-snapshots/<stamp>/attachments/ab/<uuid> /srv/nvm-internal/storage/attachments/ab/
sudo chown 65532:65532 /srv/nvm-internal/storage/attachments/ab/<uuid>
```

Setelah dicatat, bersihkan database uji (`DROP DATABASE nvm_internal_restore` sebagai `nvm_internal`) dan `/tmp/restore-storage`.

Pemulihan sungguhan memakai langkah yang sama, dengan target `nvm_internal_prod` yang dikosongkan dulu dan storage disalin dari satu snapshot:

```bash
SNAP=backup/storage-snapshots/<stamp>
sudo rm -f "$SNAP/.weekly"                      # penanda retensi, bukan berkas aplikasi
sudo cp -a "$SNAP/." /srv/nvm-internal/storage/
sudo chown -R 65532:65532 /srv/nvm-internal/storage
```

`cp -a` dari snapshot tidak memakan ruang dua kali hanya bila targetnya di filesystem yang sama; ke `/srv` yang terpisah, berkasnya benar-benar disalin. Arsip `storage-*.tar.gz` yang masih ada tetap bisa dipakai dengan `tar -xzf` seperti sebelumnya.

### Salinan ke Google Drive (revisi 25)

Setiap malam VM menyalin berkas aplikasi ke Google Drive akun tim dengan struktur yang bisa dibaca orang. VM tetap tempat utama; Drive adalah arsip. Salinannya satu arah dan tidak pernah menghapus: berkas yang dihapus di aplikasi tidak diunggah lagi, tetapi salinan lamanya di Drive dibiarkan.

Struktur di Drive, di dalam folder induk `NVM Internal` yang dibuat rclone sendiri:

```
NVM Internal/
  Aktivitas/2026-11 November/Aktivitas Harian November 2026.xlsx    (ditimpa tiap malam)
  Aktivitas/2026-11 November/18 Rabu/<Nama Staf>/09.00-10.00 - <nama berkas>
  Task/2026-11/T-0012 <judul task>/Brief/<nama berkas>
  Task/2026-11/T-0012 <judul task>/Deliverable/<nama berkas>
  Task/2026-11/T-0012 <judul task>/Sub task 1 - <judul> (<PIC>)/<nama berkas>
  Backup Database/2026-11-18.dump.age
```

Yang mengerjakannya: `deploy/drive-export.sh`, tiga langkah. Pertama perintah `backend drive-export stage` menyusun pohon symlink bernama rapi di `/srv/nvm-internal/drive-staging` (symlink ke berkas di `/data/storage`, jadi tidak ada berkas yang tersalin dan tidak ada disk yang terpakai), menulis ekspor Excel bulan berjalan, dan mengenkripsi dump dengan age. Container rclone ikut memount storage di path yang sama dan dijalankan dengan `--copy-links`, supaya yang terunggah isi berkasnya, bukan symlink-nya. Kedua container rclone resmi menyalin folder itu ke Drive. Ketiga `backend drive-export record` mencatat berkas yang sudah terunggah di tabel `drive_exports`, menulis baris status untuk halaman Status, lalu membuang staging. Kalau langkah mana pun gagal, tidak ada yang tercatat dan malam berikutnya dicoba lagi.

#### 1. Membuat remote rclone dengan client OAuth sendiri

rclone memakai client OAuth milik tim sendiri, bukan client bersama bawaan rclone. Client bersama itu sedang dipensiunkan Google dan kuotanya dibagi dengan semua pengguna rclone di dunia. VM tidak punya browser, jadi izin Google diambil di laptop lalu hasilnya dipindah. **Jangan pernah mengirim token, client secret, atau isi `rclone.conf` lewat chat, dan jangan menempelkannya di tempat yang tidak perlu.**

**a. Project Google Cloud** (sekali saja, di [console.cloud.google.com](https://console.cloud.google.com), masuk dengan akun Google tim):

1. Buat project baru, misalnya `NVM Internal Drive`.
2. APIs & Services, Library: aktifkan **Google Drive API**.
3. **Google Auth Platform**:
   - Audience: **External**.
   - Data Access: tambah scope `https://www.googleapis.com/auth/drive.file` saja.
   - Branding: nama aplikasi, email dukungan, home page `https://novoidminds.xyz`, dan privacy policy `https://novoidminds.xyz/privasi/`. **Jangan unggah logo**: logo memicu verifikasi merek oleh Google, sedangkan tanpa logo aplikasi bisa dipakai langsung.
   - Audience: **Publish app** sampai statusnya **In production**. Di status Testing, token kedaluwarsa setelah 7 hari dan job malam berhenti diam-diam seminggu kemudian.
4. Clients: buat OAuth client bertipe **Desktop app**, lalu unduh berkas JSON-nya. Berkas ini berisi `client_id` dan `client_secret`; simpan di pengelola password, bukan di repo.

Dua hal di konsol yang terlihat seperti masalah tetapi wajar:

- Status Branding "not shown" (merek tidak ditampilkan) memang hasil dari tidak mengunggah logo dan tidak memverifikasi. Biarkan.
- **Jangan submit verifikasi.** `drive.file` bukan scope sensitif, jadi aplikasi tetap jalan In production tanpa verifikasi.

**b. Laptop** (rclone dari [rclone.org/downloads](https://rclone.org/downloads/)). `client_id` dan `client_secret` diambil dari berkas JSON tadi (kunci `installed.client_id` dan `installed.client_secret`):

```powershell
rclone config create nvmdrive drive scope drive.file client_id <client_id dari JSON> client_secret <client_secret dari JSON>
```

Pakai `rclone config create` dengan nilai tertulis seperti di atas, **bukan** `rclone authorize` dengan argumen JSON: tanda kutip ganda di dalam JSON dimakan PowerShell, dan hasilnya galat yang membingungkan.

Browser terbuka. Masuk dengan akun Google tim, lalu:

- Layar **"Google hasn't verified this app"** wajar untuk aplikasi tanpa verifikasi merek. Pilih Advanced, lalu lanjut ke aplikasinya.
- Izinkan akses. `drive.file` berarti rclone hanya bisa melihat dan mengubah berkas yang dia buat sendiri, bukan seluruh Drive akun tim.
- **Biarkan PowerShell menunggu sampai muncul `Got code`.** Menutup jendela PowerShell lebih awal membuat remote tersimpan tanpa token.

Periksa dari laptop bahwa remote-nya jalan:

```powershell
rclone about nvmdrive:
rclone ls "nvmdrive:NVM Internal"
```

Baris kedua kosong sebelum unggahan pertama. Setelah itu, perintah ini cara tercepat memeriksa isi Drive dari laptop. Lokasi berkas konfigurasi laptop dicetak `rclone config file`; blok `[nvmdrive]` di dalamnya yang dipindah ke VM.

**c. VM**: buat folder secrets lalu tempel blok `[nvmdrive]` dari laptop:

```bash
sudo install -d -o 65532 -g 65532 -m 711 /srv/nvm-internal/secrets
sudo nano /srv/nvm-internal/secrets/rclone.conf
```

Isinya lima baris kunci di bawah satu kepala, dengan nilai asli dari laptop:

```ini
[nvmdrive]
type = drive
client_id = <dari laptop>
client_secret = <dari laptop>
scope = drive.file
token = <dari laptop>
```

Dua catatan yang menghabiskan waktu kalau tidak disebut:

- **`sudo nano`, bukan `sudoedit`.** `sudoedit` menolak path yang induknya bisa ditulis user biasa, dan `/srv/nvm-internal` memang begitu.
- **Mode folder 711, bukan 700.** Isinya milik UID 65532 (container rclone), sedangkan `drive-export.sh` berjalan sebagai user VM dan memeriksa dulu apakah `rclone.conf` ada. Dengan 700, user VM bahkan tidak bisa menelusuri folder itu, jadi pemeriksaannya selalu gagal dan skripnya berhenti sebelum mulai. 711 memberi hak menelusuri (`x`) tanpa hak membaca daftar isi (`r`), dan berkasnya tetap 600 milik 65532, jadi isinya tetap tidak bisa dibaca user VM.

Kunci izinnya. Folder di-mount sebagai folder, bukan satu berkas, karena rclone menulis ulang berkas ini setiap kali token diperbarui:

```bash
sudo chown -R 65532:65532 /srv/nvm-internal/secrets
sudo chmod 711 /srv/nvm-internal/secrets
sudo chmod 600 /srv/nvm-internal/secrets/rclone.conf
```

Periksa isinya tanpa mencetak rahasia. Setiap angka harus **1**:

```bash
for k in '^\[nvmdrive\]' '^token *=' '^client_id *=' '^client_secret *='; do
  printf '%-20s %s\n' "$k" "$(sudo grep -c "$k" /srv/nvm-internal/secrets/rclone.conf)"
done
```

Angka 2 berarti bloknya tertempel dua kali (kejadian di nano saat tempel kedua). Perintah ini menyisakan blok pertama saja, lewat berkas sementara bermode 600 supaya isinya tidak pernah terbaca user lain, lalu ulangi pemeriksaan di atas:

```bash
sudo sh -c 'umask 077; f=/srv/nvm-internal/secrets/rclone.conf; awk "/^\[nvmdrive\]\$/{n++} n<=1" "$f" >"$f.baru" && chown 65532:65532 "$f.baru" && mv "$f.baru" "$f"'
```

Periksa remote-nya bisa dipakai dari VM, tanpa mencetak isi rclone.conf:

```bash
docker run --rm --user 65532:65532 -e XDG_CACHE_HOME=/tmp \
  -v /srv/nvm-internal/secrets:/config/rclone rclone/rclone:1.71.0 \
  --config /config/rclone/rclone.conf about nvmdrive:
```

**d. Kalau client_id diganti** (misalnya client lama dihapus atau dibuat ulang): `drive.file` berlaku per client, jadi client baru tidak bisa melihat folder `NVM Internal` buatan client lama dan membuat folder baru bernama sama. Akibatnya:

1. Catatan `drive_exports` harus dikosongkan, supaya semua lampiran terunggah ulang lewat client baru. Ini **DELETE di produksi**, jadi hanya dijalankan dengan izin Rifky, sesudah backup:
   ```bash
   docker exec -i nvm-internal-postgres psql -U nvm_internal_migrator -d nvm_internal_prod -c "DELETE FROM drive_exports"
   ```
   `drive_folders` dibiarkan: nama folder task dan sub task tetap sama di folder yang baru.
2. Jalankan `drive-export.sh` sekali, lalu periksa folder baru dengan `rclone ls "nvmdrive:NVM Internal"` dari laptop.
3. Hapus folder `NVM Internal` lama secara manual lewat Drive di browser. Folder lama itu yang tidak punya isi terbaru.

**e. Halaman privasi.** Tautan privacy policy di layar izin Google menunjuk `https://novoidminds.xyz/privasi/`. Halamannya berkas statis `frontend/public/privasi/index.html` (gayanya di `gaya.css` sebelahnya, karena CSP melarang `<style>` inline), disajikan Nginx lewat location `/privasi/` sendiri tanpa login. Halaman itu harus tetap tayang: jangan pindahkan ke dalam aplikasi Vue dan jangan beri syarat login. Pemeriksaan setelah deploy:

```bash
curl -s https://novoidminds.xyz/privasi/ | grep -o '<title>[^<]*'   # harus: <title>Kebijakan Privasi NVM Internal
```

#### 2. Kunci age untuk dump database

Dump berisi seluruh data akun dan isi task, jadi yang diunggah ke Drive hanya versi terenkripsi. Kunci privatnya tidak pernah ada di VM, jadi siapa pun yang bisa membaca Drive tetap tidak bisa membuka dumpnya.

Di laptop (age dari [github.com/FiloSottile/age/releases](https://github.com/FiloSottile/age/releases)):

```powershell
age-keygen -o nvm-backup-key.txt
```

Berkas itu memuat dua hal: baris `# public key: age1...` dan kunci privat `AGE-SECRET-KEY-1...`.

- Kunci privat disimpan di **dua tempat di luar VM**, misalnya hard disk eksternal yang disimpan di kantor dan satu lagi di pengelola password (sebagai catatan aman), atau di volume BitLocker atau VeraCrypt. Tanpa kunci ini dump di Drive tidak bisa dipulihkan, jadi kehilangan kunci sama dengan kehilangan cadangan itu.
- Kunci privat tidak pernah ditaruh di VM, di repo, atau dikirim lewat chat.

Kunci publiknya dimasukkan ke `deploy/.env` di VM (`sudo nano`, bukan `sudoedit`, dengan alasan yang sama seperti di atas):

```bash
sudo nano /srv/nvm-internal/deploy/.env
# DRIVE_BACKUP_AGE_RECIPIENT=age1contohkuncipublikyangpanjang...
```

Kalau variabel itu kosong, berkas aplikasi tetap diunggah dan dumpnya tidak, dengan alasannya tertulis di `drive.log`.

#### 3. Uji kering dan unggah pertama

Uji kering hanya menyusun staging dan mencetak daftar path, tanpa menyentuh database dan tanpa mengunggah apa pun:

```bash
/srv/nvm-internal/deploy/drive-export.sh --dry-run
```

Daftar path keluar di stdout, dan ringkasan per jenis di stderr, bentuknya seperti ini:

```
ringkasan staging:
  brief            4
  deliverable      2
  sub task         0
  aktivitas        0
  ekspor bulanan   1
  dump             2
dilewati:
  lampiran sudah dihapus           1
```

Periksa strukturnya: nama folder task, nama Staf, jam di depan nama berkas, dan ekspor bulanan. Kalau ada jenis yang nol padahal seharusnya ada isinya, bagian `dilewati` menyebut sebabnya. Tiga perintah baca untuk mendalaminya. Semuanya hanya membaca: query dijalankan dengan `default_transaction_read_only=on`, jadi salah ketik pun tidak bisa mengubah data:

```bash
docker exec -i -e PGOPTIONS="-c default_transaction_read_only=on" nvm-internal-postgres psql -U nvm_internal_migrator -d nvm_internal_prod -c "SELECT a.id::text AS lampiran, a.task_id, t.is_deleted AS task_dihapus, (a.subtask_id IS NOT NULL) AS di_sub_task, EXISTS (SELECT 1 FROM drive_exports d WHERE d.source_kind = 'task_attachment' AND d.source_id = a.id::text) AS sudah_tercatat, 'attachments/' || substr(a.id::text, 1, 2) || '/' || a.id::text AS path_di_storage FROM task_attachments a JOIN tasks t ON t.id = a.task_id WHERE NOT a.is_deleted ORDER BY a.uploaded_at"
```

```bash
docker run --rm --network none --user 65532:65532 -v /srv/nvm-internal/storage:/data/storage:ro alpine:3.22 sh -c 'find /data/storage/attachments -type f | head -20; echo "jumlah: $(find /data/storage/attachments -type f | wc -l)"'
```

```bash
docker exec -i -e PGOPTIONS="-c default_transaction_read_only=on" nvm-internal-postgres psql -U nvm_internal_migrator -d nvm_internal_prod -c "SELECT source_kind, source_id, drive_path, exported_at FROM drive_exports ORDER BY exported_at" -c "SELECT id, finished_at, ok, files, message FROM drive_export_runs ORDER BY id"; tail -5 /srv/nvm-internal/backup/drive.log
```

Yang pertama memperlihatkan apakah task-nya sudah dihapus, apakah berkasnya sudah tercatat, dan di path mana berkasnya seharusnya berada. Yang kedua memperlihatkan berkas yang benar-benar ada di disk, dibaca sebagai UID 65532 dengan storage `:ro`, jadi aman dijalankan saat aplikasi sedang dipakai. Yang ketiga memperlihatkan berkas yang sudah tercatat terunggah dan riwayat run: baris `drive_exports` tanpa run OK yang cocok berarti `record` pernah dijalankan tanpa unggahan. Setelah cocok, jalankan sekali sungguhan:

```bash
/srv/nvm-internal/deploy/drive-export.sh && tail -1 /srv/nvm-internal/backup/drive.log
```

Barisnya berbentuk `OK berkas=12 ukuran=48M`, atau `GAGAL ... pesan=...`. Lalu cek di Drive akun tim lewat browser, dan di aplikasi buka halaman **Status** (Admin): di bawah pemeriksaan koneksi ada "Salinan Drive terakhir" dengan waktu, OK atau galat, dan jumlah berkas.

Staging setelah run yang berhasil kosong isinya, karena dikosongkan perintah `record`. Foldernya sendiri tetap ada, karena itu titik mount container:

```bash
ls -la /srv/nvm-internal/drive-staging
```

Kalau uji kering mencetak daftar path tetapi `rclone copy` gagal dengan "source file not found", berarti symlink di staging menggantung: path storage di dalam container rclone berbeda dari yang dilihat backend. Keduanya harus `/data/storage`:

```bash
docker compose --env-file deploy/.env -f deploy/docker-compose.yml run --rm -T backend drive-export stage --dry-run </dev/null | head -5
ls -l /srv/nvm-internal/drive-staging/Task/*/*/Brief 2>/dev/null | head -3   # symlink harus menunjuk /data/storage/attachments/...
```

#### 4. Memulihkan dump dari Drive

Unduh dumpnya (di laptop, dari Drive lewat browser atau rclone), lalu dekripsi dengan kunci privat age:

```powershell
age --decrypt -i nvm-backup-key.txt -o 2026-11-18.dump 2026-11-18.dump.age
```

Setelah itu pindahkan `.dump` ke VM dan ikuti langkah di bagian **Uji restore pertama** di atas, dengan `$DUMP` menunjuk berkas itu. Berkas lampiran di Drive tidak terenkripsi, jadi bisa disalin langsung ke `/srv/nvm-internal/storage` bila perlu, tetapi namanya di Drive adalah nama yang dibaca orang, bukan UUID yang dipakai aplikasi: untuk pemulihan penuh pakai snapshot storage di VM, bukan Drive.

#### 5. Cron

Cron memanggil skripnya lewat path lengkap, jadi izin eksekusinya harus ada. Izin itu ikut di git (mode 100755),
dan `scripts/check.sh` memeriksanya untuk semua `deploy/*.sh`, karena di Windows git tidak merekam bit itu
sendiri dan `drive-export.sh` pernah ter-commit tanpa izin eksekusi. Setelah `git pull` di VM, pastikan:

```bash
ls -l /srv/nvm-internal/deploy/drive-export.sh   # harus -rwxr-xr-x
```

```cron
# 20.45 UTC = 03.45 WIB: salinan berkas ke Google Drive, setelah backup (20.00) dan job harian (20.30)
45 20 * * * /srv/nvm-internal/deploy/drive-export.sh >>/srv/nvm-internal/backup/drive-cron.log 2>&1
```

Bandwidth tidak dibatasi karena jalannya malam; rclone memakai `--transfers 2` dan percobaan ulang bawaannya.

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
# sesudahnya, lalu cek daftar DELETE di keluarannya tetap delapan tabel (task_series tidak boleh ada di sana).
# Migrasi 000016 (revisi 21, Aktivitas Harian) menambah tabel daily_activities: 10-grants.sql WAJIB dijalankan
# ulang sesudahnya, lalu cek daftar DELETE di keluarannya tetap delapan tabel (daily_activities tidak boleh ada
# di sana, karena sel yang dikosongkan disimpan NULL lewat UPDATE).
# Migrasi 000017 (revisi 24, lampiran Aktivitas Harian) menambah tabel activity_attachments: 10-grants.sql WAJIB
# dijalankan ulang sesudahnya, dan tabel itu MEMANG harus ada di daftar DELETE, jadi keluarannya delapan tabel.
# Migrasi 000019 (revisi 28, To Do pribadi) menambah tabel user_todos, yang juga perlu DELETE: 10-grants.sql
# WAJIB dijalankan ulang sesudahnya, dan keluarannya jadi delapan tabel.
# Migrasi 000020 (revisi 30, Timeline) menambah tabel timeline_events dan timeline_notes: 10-grants.sql WAJIB
# dijalankan ulang sesudahnya, lalu cek daftar DELETE tetap delapan tabel (keduanya tidak boleh ada di sana,
# karena hapus memakai is_deleted).
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
