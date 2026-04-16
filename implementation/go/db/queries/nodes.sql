-- name: GetNode :one
SELECT * FROM nodes WHERE id = $1;

-- name: ListNodesByLevel :many
SELECT * FROM nodes WHERE level = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: ListNodesByContentClass :many
SELECT * FROM nodes WHERE content_class = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: ListNodesByContentClassAndType :many
SELECT * FROM nodes WHERE content_class = $1 AND content_type = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4;

-- name: NodeExistsBySha256 :one
SELECT id FROM nodes WHERE level = 0 AND content_sha256 = $1 LIMIT 1;

-- name: InsertNode :exec
INSERT INTO nodes (
    id, level, content_class, content_type, encoding_class, encoding_format,
    content_sha256, content_len, content_cached,
    artifact_created_at, artifact_created_at_blur, origin, original_name,
    valid_from, valid_from_blur, valid_until, valid_until_blur, confidence
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9,
    $10, $11, $12, $13,
    $14, $15, $16, $17, $18
);
