# Audit fungsi bentuk Whiteboard (revisi 57)

Ditulis 2026-11-27 dari kode di `frontend/src/utils/wbShapeCatalog.json`, `frontend/src/utils/wbShapes.ts` (tabel `GEO`), `frontend/src/components/whiteboard/WbShape.vue`, dan `backend/internal/whiteboard/content_w5.go`. Pembanding: fungsi yang umum di dbdiagram, Lucidchart, draw.io, dan Miro (hanya fungsinya, bukan tampilan atau ikonnya).

Yang sudah dimiliki semua bentuk dan tidak diulang per baris: teks berformat, isian dan warna garis, tebal dan gaya garis, putar, balik, ubah ukuran, port garis (empat sisi dan empat sudut), kunci posisi, grup, undo dan redo, realtime, kunci lunak, salin tempel, template, ekspor PNG dan PDF, dan Gaya bentuk sebagai jalur keyboard.

Prioritas: **Tinggi** berarti bentuk itu tidak bisa dipakai untuk tujuannya tanpa fungsi ini. **Sedang** berarti bisa dipakai tetapi pengguna harus mengakali. **Rendah** berarti pelengkap.

## Kelompok yang sudah ditetapkan (57a sampai 57c)

| Bentuk | Fungsi yang diharapkan | Sekarang | Kekurangan | Usulan | Prioritas |
|---|---|---|---|---|---|
| Tabel ERD (baru) | Nama tabel, kolom dengan tipe, PK, FK, UK, nullable; garis ke kolom tertentu | Belum ada; ERD hanya notasi Chen | Tidak bisa membuat skema basis data | 57a, sudah dikerjakan | Tinggi |
| Bingkai ponsel | Isi tangkapan layar atau foto; bentuk wireframe di dalamnya ikut terseret | Bingkai dan teks "Layar" | Tidak bisa diisi gambar, bukan wadah | 57b | Tinggi |
| Jendela browser | Sama dengan bingkai ponsel, plus alamat halaman | Bingkai, tiga titik, kolom alamat kosong, teks "Halaman" | Tidak bisa diisi gambar, bukan wadah, alamat tidak bisa ditulis di kolomnya | 57b; alamat di kolom alamat masuk 57d | Tinggi |
| Placeholder gambar | Diganti gambar sungguhan saat desain jadi | Kotak bersilang | Tidak bisa diisi gambar | 57b | Tinggi |
| Topik pusat, Cabang (mind map) | Tab menambah anak, Enter menambah saudara, tata letak otomatis, lipat cabang | Bentuk biasa; garis otomatis melengkung | Semua cabang ditaruh dan disambung manual | 57c | Tinggi |
| Garis waktu | Titik peristiwa yang bisa ditambah, digeser, diberi label dan tanggal | Garis dengan empat titik tetap, satu teks judul | Titik tidak bisa diubah dan tidak berlabel | 57c | Tinggi |
| Corong | Jumlah tahap dan label per tahap | Tiga tahap tetap, satu teks di atas | Label per tahap tidak ada | 57c | Sedang |
| Chevron | Deret tahap proses dengan label per tahap | Satu chevron satu teks | Deret harus disusun manual dari banyak bentuk | 57c (chevron bertahap dengan label per tahap) | Sedang |

## Bentuk lain

| Bentuk | Fungsi yang diharapkan | Sekarang | Kekurangan | Usulan | Prioritas |
|---|---|---|---|---|---|
| Persegi, persegi sudut bulat, elips, segitiga, belah ketupat, segi enam, bintang | Teks dan gaya | Lengkap | Teks panjang terpotong, bentuk tidak ikut membesar | Lihat "Teks melebihi bentuk" di bawah | Rendah |
| Proses, data, dokumen, mulai atau selesai, input manual, persiapan, delay, merge, sort | Teks dan garis alur | Lengkap | Menambah langkah berikutnya selalu lewat palet lalu garis | Lihat "Tambah bentuk tersambung" di bawah | Sedang |
| Keputusan (flowchart) | Cabang Ya dan Tidak | Bentuk dan label garis | Label Ya dan Tidak diketik manual di tiap garis | Tidak perlu fungsi khusus; label garis sudah cukup | Rendah |
| Database (flowchart), Database (arsitektur) | Teks | Lengkap | Tidak ada | Tidak ada | Rendah |
| Proses terdefinisi | Rujukan ke proses lain | Bentuk dan teks | Tidak menaut ke papan atau kartu yang dirujuk | Tautan ke kartu atau papan lain (bersama Konektor halaman lain) | Rendah |
| Konektor halaman lain | Menaut ke bagian lain atau papan lain dan bisa diklik untuk pindah ke sana | Segi lima dengan teks | Tidak menaut ke mana pun | Tautan ke kartu atau papan tujuan; klik atau Enter pindah ke sana | Sedang |
| Kelas UML (empat varian) | Nama, atribut, metode; tinggi mengikuti isi | Tiga bagian berformat, varian, stereotip | Tinggi tetap: atribut atau metode yang panjang terpotong tanpa tanda | Tinggi kelas mengikuti isi bagiannya (seperti tabel ERD) | Sedang |
| Aktor UML | Nama di bawah | Lengkap; port di tangan | Tidak ada | Tidak ada | Rendah |
| Catatan UML | Catatan yang ditautkan ke elemen dengan garis putus-putus | Bentuk; preset Association ada | Tidak ada | Tidak ada | Rendah |
| Paket | Wadah berlabel di tab yang membawa kelas di dalamnya | Bentuk dengan teks; bukan wadah | Kelas di dalam paket tidak ikut terseret | Jadikan wadah seperti Batas sistem | Sedang |
| Batas sistem, Use case | Wadah dan elips | Lengkap (wadah sejak 54b) | Tidak ada | Tidak ada | Rendah |
| Swimlane, Pool dan lane | Lane bisa ditambah, diubah nama, diurutkan, isi ikut lane | Lengkap (54b dan 56) | Tidak ada | Tidak ada | Rendah |
| Node awal, node akhir, flow final, aksi, keputusan atau merge, fork atau join, object node, kirim dan terima sinyal, choice | Bentuk activity | Lengkap; port fork di batangnya | Tidak ada | Tidak ada | Rendah |
| State | Nama dan aktivitas internal (entry, do, exit) | Satu teks | Aktivitas internal harus ditulis di teks yang sama | Bagian kedua untuk aktivitas internal, dipisah garis seperti kelas | Rendah |
| State komposit | Wadah berjudul | Lengkap | Tidak ada | Tidak ada | Rendah |
| Lifeline, lifeline aktor, activation bar, fragmen | Pesan mendatar, activation menempel, operand | Lengkap (54d, 56) | Nomor pesan diketik manual | Penomoran pesan otomatis (opsional, per papan) | Rendah |
| Entitas, entitas lemah, atribut, atribut kunci, relasi (Chen) | Notasi Chen | Lengkap | Atribut multivalue (elips ganda) dan atribut turunan belum ada; turunan bisa memakai garis putus-putus | Varian atribut: multivalue dan turunan | Rendah |
| Event mulai, antara, selesai (BPMN) | Varian event; event batas yang menempel di tepi task dan ikut terseret | Varian lengkap; event bebas | Event batas tidak menempel ke task; saat task digeser, event tertinggal | Event antara yang dilepas di tepi task menempel ke tepi itu dan ikut terseret | Sedang |
| Task (BPMN) | Varian jenis task, penanda loop, multi-instance, kompensasi | Tujuh varian jenis | Penanda loop dan multi-instance tidak ada | Penanda di bawah task (loop, multi-instance paralel dan berurutan, kompensasi), bisa digabung dengan jenis | Sedang |
| Sub-proses (BPMN) | Penanda tertutup [+] atau terbuka, isi proses di dalamnya | Bentuk dengan penanda tetap; bukan wadah | Proses di dalam sub-proses terbuka tidak ikut terseret | Varian tertutup dan terbuka; yang terbuka menjadi wadah | Sedang |
| Gateway, data object, data store, anotasi (BPMN) | Varian dan label | Lengkap | Data object koleksi (tiga garis) belum ada | Varian koleksi pada data object | Rendah |
| Panah blok (lima arah) | Teks dan arah | Lengkap | Tidak ada | Tidak ada | Rendah |
| Server, cloud, pengguna, laptop, ponsel, firewall, komponen, node 3D | Ikon berlabel; zona jaringan atau region sebagai wadah | Ikon dan label | Tidak ada wadah zona (VPC, region, jaringan kantor) | Bentuk "Zona" (wadah bergaris putus-putus berjudul) | Sedang |
| Tombol, isian, dropdown (wireframe) | Teks di dalam | Lengkap | Tidak ada | Tidak ada | Rendah |
| Kotak centang (wireframe) | Klik kotak untuk mencentang | Varian tercentang lewat Gaya bentuk | Mencentang perlu membuka Gaya bentuk | Klik atau Spasi pada kotak centang mengganti varian (satu langkah undo) | Sedang |
| Callout | Ekor menunjuk ke titik yang dimaksud | Ekor tetap di kiri bawah | Ekor tidak bisa diarahkan | Pegangan ujung ekor yang bisa diseret (posisi disimpan di isi bentuk) | Sedang |

## Lintas bentuk

| Hal | Fungsi yang diharapkan | Sekarang | Kekurangan | Usulan | Prioritas |
|---|---|---|---|---|---|
| Tambah bentuk tersambung | Dari bentuk terpilih, satu klik atau pintasan menambah bentuk berikutnya yang langsung tersambung garis (flowchart, BPMN, activity) | Palet lalu tarik garis | Tiga sampai lima langkah per langkah alur | Tombol di sisi bentuk terpilih (dan Ctrl+Enter) membuat bentuk sejenis 60 px di sisi itu dan garis ke sana, satu langkah undo | Sedang |
| Teks melebihi bentuk | Teks tidak hilang diam-diam | Teks terpotong di tepi area teks | Isi terpotong tanpa tanda | Tanda terpotong (gradasi atau ikon titik tiga) dan perintah "Pas ke teks" yang menambah tinggi bentuk | Rendah |

## Usulan 57d (disetujui 2026-11-27, dikerjakan 57d1 sampai 57d5)

Butir prioritas sedang dari tabel di atas, urut dari yang paling sering dipakai tim:

1. Tambah bentuk tersambung dari bentuk terpilih (lintas bentuk). Selesai di 57d1.
2. Kotak centang wireframe bisa dicentang langsung.
3. Tinggi kelas UML mengikuti isi.
4. Paket UML dan Zona arsitektur sebagai wadah.
5. BPMN: event batas menempel di tepi task, penanda task (loop, multi-instance, kompensasi), sub-proses tertutup dan terbuka (terbuka sebagai wadah).
6. Ekor callout bisa diarahkan.
7. Konektor halaman lain menaut ke kartu atau papan tujuan, dan alamat di kolom alamat jendela browser.

Butir rendah tidak dikerjakan di revisi 57 kecuali diminta.
