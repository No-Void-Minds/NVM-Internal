-- Uji aturan skema 000001. Semua data uji dibatalkan di akhir, database tetap bersih.
DO $$
DECLARE
  d bigint;
  u bigint;
  t varchar;
BEGIN
  INSERT INTO divisions (name) VALUES ('Uji skema') RETURNING id INTO d;
  INSERT INTO users (name, username, password_hash, role, division_id)
    VALUES ('Uji', 'Uji.A', 'x', 'manager', d) RETURNING id INTO u;

  BEGIN
    INSERT INTO users (name, username, password_hash, role, division_id)
      VALUES ('Uji 2', 'uji.a', 'x', 'staff', d);
    RAISE NOTICE 'FAIL: username kembar beda huruf diterima';
  EXCEPTION WHEN unique_violation THEN
    RAISE NOTICE 'PASS: username kembar beda huruf ditolak';
  END;

  INSERT INTO tasks (title, division_id, requested_by, urgency)
    VALUES ('Uji', d, u, 'low') RETURNING id INTO t;
  IF t ~ '^T-[0-9]{4}$' THEN
    RAISE NOTICE 'PASS: ID otomatis %', t;
  ELSE
    RAISE NOTICE 'FAIL: format ID otomatis %', t;
  END IF;

  BEGIN
    INSERT INTO tasks (title, division_id, requested_by, urgency, start_date, due_date)
      VALUES ('Uji', d, u, 'low', '2026-11-10', '2026-11-01');
    RAISE NOTICE 'FAIL: tenggat sebelum tanggal mulai diterima';
  EXCEPTION WHEN check_violation THEN
    RAISE NOTICE 'PASS: tenggat sebelum tanggal mulai ditolak';
  END;

  BEGIN
    UPDATE tasks SET status = 'completed' WHERE id = t;
    RAISE NOTICE 'FAIL: completed tanpa solved_date diterima';
  EXCEPTION WHEN check_violation THEN
    RAISE NOTICE 'PASS: completed tanpa solved_date ditolak';
  END;

  BEGIN
    UPDATE tasks SET solved_date = now() WHERE id = t;
    RAISE NOTICE 'FAIL: solved_date pada task yang belum completed diterima';
  EXCEPTION WHEN check_violation THEN
    RAISE NOTICE 'PASS: solved_date pada task yang belum completed ditolak';
  END;

  BEGIN
    UPDATE tasks SET deliverable_url = 'javascript:alert(1)' WHERE id = t;
    RAISE NOTICE 'FAIL: tautan selain http/https diterima';
  EXCEPTION WHEN check_violation THEN
    RAISE NOTICE 'PASS: tautan selain http/https ditolak';
  END;

  RAISE EXCEPTION 'Uji selesai, semua data uji dibatalkan.';
END $$;

-- Nomor sequence tidak ikut dibatalkan, jadi dikembalikan supaya task pertama tetap T-0001.
SELECT setval('task_number_seq', 1, false);