-- Dijalankan Rifky sekali sebagai superuser Postgres bersama (nvm_internal), bukan oleh aplikasi:
--   psql -U nvm_internal -d postgres -f 00-bootstrap.sql
-- Pola sama dengan compro (deploy/sql/00-bootstrap.sql). Password diminta interaktif supaya tidak tertinggal di
-- riwayat shell; boleh juga lewat -v migrator_password=... -v app_password=...
-- Database dev nvm_internal dan nvm_internal_test tidak disentuh.

\set ON_ERROR_STOP on

\if :{?migrator_password}
\else
  \prompt 'Password nvm_internal_migrator: ' migrator_password
\endif
\if :{?app_password}
\else
  \prompt 'Password nvm_internal_app: ' app_password
\endif

-- Migrator memiliki skema dan menjalankan migrate. App hanya DML, dan tidak pernah superuser.
CREATE ROLE nvm_internal_migrator LOGIN PASSWORD :'migrator_password';
CREATE ROLE nvm_internal_app LOGIN PASSWORD :'app_password';

CREATE DATABASE nvm_internal_prod OWNER nvm_internal_migrator;

-- Container dipakai bersama compro dan doomsday, jadi role lain tidak boleh masuk.
REVOKE ALL ON DATABASE nvm_internal_prod FROM PUBLIC;
GRANT CONNECT ON DATABASE nvm_internal_prod TO nvm_internal_app;

\connect nvm_internal_prod

-- PG15 ke atas sudah tidak memberi CREATE di public untuk PUBLIC; ditulis tetap supaya jelas.
REVOKE ALL ON SCHEMA public FROM PUBLIC;
ALTER SCHEMA public OWNER TO nvm_internal_migrator;
GRANT USAGE ON SCHEMA public TO nvm_internal_app;

-- Tabel baru dari migrasi otomatis bisa dibaca dan diubah app, tetapi tanpa DELETE. DELETE diberikan per tabel
-- di 10-grants.sql setelah migrasi, karena tabelnya belum ada saat skrip ini jalan.
ALTER DEFAULT PRIVILEGES FOR ROLE nvm_internal_migrator IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE ON TABLES TO nvm_internal_app;

-- task_number_seq (ID task) dan kolom identity butuh hak sequence untuk INSERT.
ALTER DEFAULT PRIVILEGES FOR ROLE nvm_internal_migrator IN SCHEMA public
  GRANT USAGE, SELECT ON SEQUENCES TO nvm_internal_app;
