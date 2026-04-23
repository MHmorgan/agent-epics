-- name: InsertTask :exec
INSERT INTO task (id, parent_id, title, body, context, status, position)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetTask :one
SELECT * FROM task WHERE id = ?;

-- name: ListTasks :many
-- All tasks that are branches (status IS NULL) or non-terminal leaves
SELECT * FROM task
WHERE status IS NULL OR status NOT IN ('done', 'abandoned')
ORDER BY position, created_at;

-- name: ListAllTasks :many
SELECT * FROM task ORDER BY position, created_at;

-- name: ListTasksByParent :many
-- Immediate children excluding terminal leaves
SELECT * FROM task
WHERE parent_id = ?
  AND (status IS NULL OR status NOT IN ('done', 'abandoned'))
ORDER BY position, created_at;

-- name: ListAllTasksByParent :many
SELECT * FROM task WHERE parent_id = ? ORDER BY position, created_at;

-- name: UpdateTaskBody :exec
UPDATE task SET body = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: UpdateTaskTitle :exec
UPDATE task SET title = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: UpdateTaskContext :exec
UPDATE task SET context = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: UpdateTaskStatus :exec
UPDATE task SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: ClearTaskStatus :exec
UPDATE task SET status = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: DeleteTask :exec
DELETE FROM task WHERE id = ?;

-- name: DeleteTasksByParent :exec
DELETE FROM task WHERE parent_id = ?;

-- name: CountChildren :one
SELECT COUNT(*) FROM task WHERE parent_id = ?;

-- name: MaxChildPosition :one
SELECT COALESCE(MAX(position), 0) FROM task WHERE parent_id = ?;

-- name: GetTaskContext :one
SELECT context FROM task WHERE id = ?;
