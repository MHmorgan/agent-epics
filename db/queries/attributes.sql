-- name: SetAttribute :exec
INSERT OR REPLACE INTO attribute (attribute, value) VALUES (?, ?);

-- name: GetAttribute :one
SELECT value FROM attribute WHERE attribute = ?;
