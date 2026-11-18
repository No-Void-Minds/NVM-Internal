-- name: CreateSeries :one
INSERT INTO task_series (freq, skip_weekend, weekdays, month_day, start_date, end_date, template, created_by)
VALUES (@freq, @skip_weekend, @weekdays::smallint[], sqlc.narg(month_day), @start_date, sqlc.narg(end_date),
        @template, @created_by)
RETURNING id;

-- name: GetSeries :one
SELECT id, freq, skip_weekend, weekdays, month_day, start_date, end_date, active, template, created_by
FROM task_series WHERE id = @id;

-- name: GetSeriesForUpdate :one
SELECT id, freq, skip_weekend, weekdays, month_day, start_date, end_date, active, template, created_by
FROM task_series WHERE id = @id
FOR UPDATE;

-- name: UpdateSeries :exec
UPDATE task_series
SET freq = @freq, skip_weekend = @skip_weekend, weekdays = @weekdays::smallint[], month_day = sqlc.narg(month_day),
    end_date = sqlc.narg(end_date), template = @template
WHERE id = @id;

-- name: StopSeries :execrows
UPDATE task_series SET active = false WHERE id = @id AND active;

-- name: EndExpiredSeries :execrows
-- Seri yang end_date-nya sudah lewat otomatis berhenti.
UPDATE task_series SET active = false WHERE active AND end_date < @today::date;

-- name: ListActiveSeries :many
SELECT id, freq, skip_weekend, weekdays, month_day, start_date, end_date, active, template, created_by
FROM task_series
WHERE active AND start_date <= @today::date AND (end_date IS NULL OR end_date >= @today::date)
ORDER BY id;

-- name: OccurrenceExists :one
SELECT EXISTS (SELECT 1 FROM tasks WHERE series_id = @series_id AND occurrence_date = @occurrence_date);

-- name: SetTaskSeries :exec
UPDATE tasks SET series_id = @series_id, occurrence_date = @occurrence_date WHERE id = @id;

-- name: GetTaskSeriesRef :one
SELECT series_id, occurrence_date, start_date FROM tasks WHERE id = @id AND NOT is_deleted;
