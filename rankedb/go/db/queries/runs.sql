-- name: InsertRun :exec
INSERT INTO runs (id, worker_config_id) VALUES ($1, $2);

-- name: GetRun :one
SELECT * FROM runs WHERE id = $1;
