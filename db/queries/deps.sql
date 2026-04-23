-- name: InsertDep :exec
INSERT INTO dep (task_id, after_id) VALUES (?, ?);

-- name: DeleteDep :exec
DELETE FROM dep WHERE task_id = ? AND after_id = ?;

-- name: ListDepsForTask :many
-- Get all predecessors of a given task
SELECT * FROM dep WHERE task_id = ?;

-- name: ListDependentsOfTask :many
-- Get all tasks that depend on a given task (successors)
SELECT * FROM dep WHERE after_id = ?;

-- name: DeleteDepsByTask :exec
-- Cleanup for unsplit: remove all deps where task is involved
DELETE FROM dep WHERE task_id = ? OR after_id = ?;
