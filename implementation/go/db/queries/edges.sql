-- name: InsertEdge :exec
INSERT INTO edges (id, source_node_id, target_node_id, type, confidence, run_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetEdge :one
SELECT * FROM edges WHERE id = $1;

-- name: GetEdgesBySource :many
SELECT * FROM edges WHERE source_node_id = $1 ORDER BY created_at;

-- name: GetEdgesByTarget :many
SELECT * FROM edges WHERE target_node_id = $1 ORDER BY created_at;

-- name: GetEdgesBySourceAndType :many
SELECT * FROM edges WHERE source_node_id = $1 AND type = $2 ORDER BY created_at;

-- name: GetEdgesByTargetAndType :many
SELECT * FROM edges WHERE target_node_id = $1 AND type = $2 ORDER BY created_at;

-- name: GetEdgesByRun :many
SELECT * FROM edges WHERE run_id = $1 ORDER BY created_at;
