ALTER TABLE daily_activities
  DROP CONSTRAINT daily_activities_chain_empty,
  DROP COLUMN continues_previous;
