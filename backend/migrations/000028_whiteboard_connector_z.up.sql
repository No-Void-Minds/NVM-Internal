-- Whiteboard revisi 53: urutan tumpuk garis. Kolom di wb_connectors, tanpa tabel baru, jadi deploy/sql/10-grants.sql
-- tidak berubah.
-- z NULL berarti otomatis: garis digambar tepat di atas kartu ujung yang z-nya paling tinggi, dihitung klien saat
-- itu juga, jadi tetap benar sesudah ujungnya maju atau mundur tanpa menulis ulang baris garis. Nilai berarti urutan
-- diatur manual lewat menu (skala sama dengan wb_cards.z) atau garis yang kedua ujungnya bebas.
ALTER TABLE wb_connectors
  ADD COLUMN z double precision CHECK (z BETWEEN -1000000000 AND 1000000000);
