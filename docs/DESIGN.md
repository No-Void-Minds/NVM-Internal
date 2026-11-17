# DESIGN: NVM Internal

Identitas visual task tracker NVM sejak Sprint 11b (2026-11-17). Menggantikan versi Stitch. Token di `frontend/src/style.css` adalah sumber kebenarannya; berkas ini menjelaskan alasannya.

## Arah

- Monokrom NVM dari tema Shopify, dengan satu aksen krem hangtag. Tenang, rapi, untuk aplikasi kerja yang dibaca berjam-jam.
- Dial: ENERGY 2, RHYTHM 2, MOTION 2 (alasan di Keputusan Sprint 11 dan 11b di `docs/sprint.md`).
- Larangan: glassmorphism, blur, glow, gradien, grid titik, ikon sparkle atau robot, emoji, data karangan.

## Mode

- Mode terang bawaan, karena kantor terang. Mode gelap lewat tombol di top bar (desktop) dan menu Lainnya (ponsel).
- Bila belum memilih, mengikuti `prefers-color-scheme`. Pilihan disimpan per browser (`localStorage` kunci `nvm-tema`, dengan try/catch).
- `public/tema.js` memasang `data-theme` di `<html>` sebelum halaman digambar, jadi tidak ada kilatan terang. `color-scheme` ikut diatur supaya kalender tanggal, select, dan scrollbar sesuai mode.

## Warna

| Token | Terang | Gelap | Dipakai untuk |
|---|---|---|---|
| canvas | `#f3f2ee` | `#08090b` | latar halaman (terang = latar ikon NVM) |
| surface | `#ffffff` | `#0d0f12` | kartu, panel, tabel, isian |
| ink | `#1a1c20` | `#e8ebef` | teks utama, isi logo |
| ink-muted | `#5a5f68` | `#8a909a` | teks pendukung |
| line | `#7d828b` | `#6b717b` | garis isian, kartu, pemisah |
| accent / on-accent | `#1a1c20` / `#e8e3d5` | `#e8e3d5` / `#1a1c20` | tombol utama dan pilihan aktif: satu kelas, dua mode |
| accent-hover | `#2e3138` | `#f4f0e6` | hover tombol utama |
| mark | `#1a1c20` | `#e8e3d5` | cincin fokus, garis terpilih, teks aktif, isi bar kemajuan |
| tag / tag-ink | `#e8e3d5` / `#1a1c20` | sama | label ID hangtag dan chip label pastel |
| scrim | hitam 55% | sama | latar di belakang dialog |
| urgent / high | `#9b3a2a` / `#7a5a1c` | `#ef9a86` / `#dcb978` | urgency, selalu dengan kata |
| danger / danger-bg | `#8c2f22` / `#f6e6e2` | `#f0a497` / `#2a1411` | galat dan aksi hapus |

Status diredam supaya serasi dengan monokrom, dan selalu berupa teks: Requested, On Progress, Ready, Completed (nilai di `style.css`). Kontras semua pasangan dihitung lolos WCAG AA di kedua mode; tabelnya di Keputusan Sprint 11b.

## Tipografi

- D-DIN 400 dan 700 untuk judul dan teks, dari `/fonts` dengan `font-display: swap` (lisensi SIL OFL, `/fonts/OFL.txt`). Tidak ada bobot 500 dan 600: `font-medium` memakai 400, `font-semibold` memakai 700.
- JetBrains Mono 400 dan 500 (`@fontsource`) khusus ID task, kode, dan angka hitungan.
- Skala: body 15px, kecil 14px, terkecil 13px, judul bagian 18px, judul halaman 24px. Tanpa huruf kapital semua.

## Bentuk dan kedalaman

- Radius 4px untuk kartu, isian, tombol, dan dialog; 2px untuk chip. Bentuk bulat hanya untuk avatar.
- Bayangan hanya untuk menu, dialog, dan kartu yang sedang di-drag. Selebihnya garis 1px.

## Tata letak

- Satu kerangka untuk semua halaman yang login (Sprint 11c). Top bar selalu selebar layar dengan padding kiri dan kanan dari satu sumber (`gutter` di `AppShell.vue`: 16px, 32px mulai 768px), jadi logo, menu, dan aksi kanan tidak berpindah antar halaman.
- Lebar konten: papan (`/task`) memakai lebar penuh karena butuh empat kolom. Halaman lain ditengahkan dengan `mx-auto max-w-6xl` (1152px), supaya tabel dan form tidak terentang sampai 1920px.
- Halaman berisi form atau teks (Profil, Status, Pengaturan, 404) masih membatasi lebar isinya dengan `max-w` di dalam wadah yang sudah ditengahkan.
- Setiap halaman dibuka dengan `PageHeader`: judul 24px tebal, deskripsi opsional, aksi di kanan.
- Cari dan filter selalu lewat `FilterToolbar`: cari di kiri, filter sebaris (kontrol `filterControl`, tinggi 44px), "Hapus filter" bila ada yang aktif. Label tersembunyi bila opsi pertama sudah menjelaskan; label sebaris untuk tanggal dan urutan. Di bawah 768px filter dilipat ke tombol "Filter (n)".
- Kartu profil hanya di papan, di kiri mulai kontainer 90rem; di bawahnya bar ringkas di atas papan.
- Login dan Buat password baru memakai kartu di tengah, tanpa top bar.

## Logo

- Logo resmi `frontend/public/brand/nvm-logo.png` (putih di atas transparan, sama dengan compro) dipakai sebagai mask CSS yang diisi warna `ink`, jadi benar di kedua mode tanpa salinan yang diwarnai ulang (`BrandLogo.vue`). Tidak digambar ulang.
- Favicon: `favicon.ico`, `favicon-32.png`, `apple-touch-icon.png`.

## Motif identitas

ID task tampil sebagai label hangtag: bidang krem, teks tinta, JetBrains Mono, garis tipis (`specimen-tag`). Sama di kartu, panel detail, halaman task penuh, Arsip, dan Audit Log.

## Interaksi

- Animasi 150 sampai 220 ms hanya untuk umpan balik dan orientasi (kartu pindah kolom, panel dan dialog, menu, notifikasi). Tanpa loop. Semua mati di `prefers-reduced-motion`.
- Tap target minimal 44x44 px. Fokus terlihat lewat cincin `mark` 2px.
