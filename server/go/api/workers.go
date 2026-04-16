package api

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	db "rankedb/db"

	schemafdb "github.com/flocko-motion/schemaf/db"
)

// ─── Worker queue ─────────────────────────────────────────────────────────────

// GetQueueEndpoint returns unprocessed nodes for a given worker type.
type GetQueueEndpoint struct{}

func (e GetQueueEndpoint) Method() string { return "GET" }
func (e GetQueueEndpoint) Path() string   { return "/api/queue" }
func (e GetQueueEndpoint) Auth() bool     { return false }
func (e GetQueueEndpoint) Handle(ctx context.Context, req GetQueueReq) (GetQueueResp, error) {
	conn := schemafdb.DB()
	limit := defaultLimit(req.Limit)

	// Three levels of "already processed" filtering, from most lenient to most strict:
	// 1. by_class:  any derived node of that content class → source is consumed
	// 2. by_worker: any run by a worker with that name → source is consumed
	// 3. by_config: any run by that exact config ID → source is consumed
	// The worker chooses which level to apply.
	var query string
	var args []any

	nodeSelect := `
SELECT n.id, n.level, n.content_class, n.content_type, n.encoding_class, n.encoding_format,
       n.content_sha256, n.content_len, n.content_cached, n.created_at,
       n.artifact_created_at, n.artifact_created_at_blur, n.origin, n.original_name,
       n.valid_from, n.valid_from_blur, n.valid_until, n.valid_until_blur, n.confidence
FROM nodes n
WHERE n.content_class = $1 AND n.content_type = $2`

	switch {
	case req.ByConfig != nil:
		query = nodeSelect + `
  AND NOT EXISTS (
      SELECT 1 FROM edges e
      JOIN runs r ON e.run_id = r.id
      WHERE e.target_node_id = n.id AND e.type = 'provenance/input'
        AND r.worker_config_id = $3
  )
ORDER BY n.created_at ASC LIMIT $4`
		args = []any{req.ContentClass, req.ContentType, *req.ByConfig, limit}

	case req.ByWorker != nil:
		query = nodeSelect + `
  AND NOT EXISTS (
      SELECT 1 FROM edges e
      JOIN runs r ON e.run_id = r.id
      JOIN nodes config ON r.worker_config_id = config.id
      WHERE e.target_node_id = n.id AND e.type = 'provenance/input'
        AND config.content_cached::jsonb->>'name' = $3
  )
ORDER BY n.created_at ASC LIMIT $4`
		args = []any{req.ContentClass, req.ContentType, *req.ByWorker, limit}

	case req.ByClass != nil:
		query = nodeSelect + `
  AND NOT EXISTS (
      SELECT 1 FROM edges e
      JOIN nodes derived ON e.source_node_id = derived.id
      WHERE e.target_node_id = n.id AND e.type = 'provenance/input'
        AND derived.content_class = $3
  )
ORDER BY n.created_at ASC LIMIT $4`
		args = []any{req.ContentClass, req.ContentType, *req.ByClass, limit}

	default:
		query = nodeSelect + `
ORDER BY n.created_at ASC LIMIT $3`
		args = []any{req.ContentClass, req.ContentType, limit}
	}

	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return GetQueueResp{Nodes: []NodeResponse{}}, err
	}
	defer rows.Close()

	var nodes []NodeResponse
	for rows.Next() {
		var n db.Node
		if scanErr := rows.Scan(
			&n.ID, &n.Level, &n.ContentClass, &n.ContentType,
			&n.EncodingClass, &n.EncodingFormat, &n.ContentSha256, &n.ContentLen,
			&n.ContentCached, &n.CreatedAt, &n.ArtifactCreatedAt, &n.ArtifactCreatedAtBlur,
			&n.Origin, &n.OriginalName, &n.ValidFrom, &n.ValidFromBlur,
			&n.ValidUntil, &n.ValidUntilBlur, &n.Confidence,
		); scanErr != nil {
			return GetQueueResp{Nodes: []NodeResponse{}}, scanErr
		}
		nodes = append(nodes, nodeToResponse(n))
	}
	if rows.Err() != nil {
		return GetQueueResp{Nodes: []NodeResponse{}}, rows.Err()
	}

	if nodes == nil {
		nodes = []NodeResponse{}
	}
	return GetQueueResp{Nodes: nodes}, nil
}

type GetQueueReq struct {
	ContentClass string  `query:"content_class"`
	ContentType  string  `query:"content_type"`
	ByClass      *string `query:"by_class"`  // lenient: skip if any derived node of this class exists
	ByWorker     *string `query:"by_worker"` // medium: skip if any run by a worker with this name exists
	ByConfig     *string `query:"by_config"` // strict: skip if any run by this exact config ID exists
	Limit        int     `query:"limit"`
}

type GetQueueResp struct {
	Nodes []NodeResponse `json:"nodes"`
}

// ─── Register run ─────────────────────────────────────────────────────────────

// CreateRunEndpoint registers a new worker run and returns a run_id.
type CreateRunEndpoint struct{}

func (e CreateRunEndpoint) Method() string { return "POST" }
func (e CreateRunEndpoint) Path() string   { return "/api/runs" }
func (e CreateRunEndpoint) Auth() bool     { return false }
func (e CreateRunEndpoint) Handle(ctx context.Context, req CreateRunReq) (CreateRunResp, error) {
	queries := db.New(schemafdb.DB())

	// Validate that the worker_config_id exists and is a worker node
	workerNode, err := queries.GetNode(ctx, req.WorkerConfigID)
	if err != nil {
		if err == sql.ErrNoRows {
			return CreateRunResp{}, fmt.Errorf("worker config node not found: %s", req.WorkerConfigID)
		}
		return CreateRunResp{}, err
	}
	if workerNode.ContentClass != "worker" {
		return CreateRunResp{}, fmt.Errorf("node %s is not a worker config (content_class=%s)", req.WorkerConfigID, workerNode.ContentClass)
	}

	// Generate UUID v7 for run_id
	runID, err := uuid.NewV7()
	if err != nil {
		return CreateRunResp{}, fmt.Errorf("failed to generate run UUID: %w", err)
	}

	err = queries.InsertRun(ctx, db.InsertRunParams{
		ID:             runID.String(),
		WorkerConfigID: req.WorkerConfigID,
	})
	if err != nil {
		return CreateRunResp{}, fmt.Errorf("failed to insert run: %w", err)
	}

	return CreateRunResp{RunID: runID.String()}, nil
}

type CreateRunReq struct {
	WorkerConfigID string `json:"worker_config_id"`
}

type CreateRunResp struct {
	RunID string `json:"run_id"`
}
