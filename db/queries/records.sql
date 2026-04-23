-- name: InsertRecord :exec
INSERT INTO record (task, ts, source, text)
VALUES (?, CURRENT_TIMESTAMP, ?, ?);

-- name: ListRecordsByPrefix :many
-- Subtree records: exact match OR children (task LIKE id||':%')
SELECT * FROM record
WHERE task = sqlc.arg(task_id) OR task LIKE sqlc.arg(task_id) || ':%'
ORDER BY ts, id;

-- name: ListRecordsByTask :many
-- Exact match only
SELECT * FROM record WHERE task = ? ORDER BY ts, id;

-- name: CountNonSystemRecordsByTask :one
-- Count non-system records for a specific task (used for unsplit precondition check)
SELECT COUNT(*) FROM record WHERE task = ? AND source != 'system';

-- name: DeleteRecordsByTask :exec
-- For unsplit child cleanup
DELETE FROM record WHERE task = ?;
