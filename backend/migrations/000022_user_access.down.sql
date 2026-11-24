ALTER TABLE users
  DROP CONSTRAINT users_access_staff_only,
  DROP COLUMN can_manage_notes,
  DROP COLUMN can_manage_timeline;
