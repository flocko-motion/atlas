package api

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

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
	args := []any{req.ContentClass, req.ContentType}
	paramN := 3

	nodeSelect := `
SELECT n.id, n.level, n.content_class, n.content_type, n.encoding_class, n.encoding_format,
       n.content_sha256, n.content_len, n.content_cached, n.created_at,
       n.artifact_created_at, n.artifact_created_at_blur, n.origin, n.original_name,
       n.valid_from, n.valid_from_blur, n.valid_until, n.valid_until_blur, n.confidence, n.title
FROM nodes n
WHERE n.content_class = $1 AND n.content_type = $2`

	if req.EncodingClass != nil {
		nodeSelect += fmt.Sprintf(" AND n.encoding_class = $%d", paramN)
		args = append(args, *req.EncodingClass)
		paramN++
	}
	if req.EncodingFormat != nil {
		nodeSelect += fmt.Sprintf(" AND n.encoding_format = $%d", paramN)
		args = append(args, *req.EncodingFormat)
		paramN++
	}

	switch {
	case req.ByConfig != nil:
		query = nodeSelect + fmt.Sprintf(`
  AND NOT EXISTS (
      SELECT 1 FROM edges e
      JOIN runs r ON e.run_id = r.id
      WHERE e.source_node_id = n.id AND e.type = 'provenance/input'
        AND r.worker_config_id = $%d
  )
ORDER BY n.created_at ASC LIMIT $%d`, paramN, paramN+1)
		args = append(args, *req.ByConfig, limit)

	case req.ByWorker != nil:
		query = nodeSelect + fmt.Sprintf(`
  AND NOT EXISTS (
      SELECT 1 FROM edges e
      JOIN runs r ON e.run_id = r.id
      JOIN nodes config ON r.worker_config_id = config.id
      WHERE e.source_node_id = n.id AND e.type = 'provenance/input'
        AND config.content_cached::jsonb->>'name' = $%d
  )
ORDER BY n.created_at ASC LIMIT $%d`, paramN, paramN+1)
		args = append(args, *req.ByWorker, limit)

	case req.ByClass != nil:
		query = nodeSelect + fmt.Sprintf(`
  AND NOT EXISTS (
      SELECT 1 FROM edges e
      JOIN nodes derived ON e.target_node_id = derived.id
      WHERE e.source_node_id = n.id AND e.type = 'provenance/input'
        AND derived.content_class = $%d
  )
ORDER BY n.created_at ASC LIMIT $%d`, paramN, paramN+1)
		args = append(args, *req.ByClass, limit)

	default:
		query = nodeSelect + fmt.Sprintf(`
ORDER BY n.created_at ASC LIMIT $%d`, paramN)
		args = append(args, limit)
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
			&n.ValidUntil, &n.ValidUntilBlur, &n.Confidence, &n.Title,
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
	ContentClass   string  `query:"content_class"`
	ContentType    string  `query:"content_type"`
	EncodingClass  *string `query:"encoding_class"`
	EncodingFormat *string `query:"encoding_format"`
	ByClass        *string `query:"by_class"`  // lenient: skip if any derived node of this class exists
	ByWorker       *string `query:"by_worker"` // medium: skip if any run by a worker with this name exists
	ByConfig       *string `query:"by_config"` // strict: skip if any run by this exact config ID exists
	Limit          int     `query:"limit"`
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

// ─── Get run ─────────────────────────────────────────────────────────────────

// GetRunEndpoint returns run details and all nodes created in that run (with cascading derivatives).
type GetRunEndpoint struct{}

func (e GetRunEndpoint) Method() string { return "GET" }
func (e GetRunEndpoint) Path() string   { return "/api/runs/{id}" }
func (e GetRunEndpoint) Auth() bool     { return false }
func (e GetRunEndpoint) Handle(ctx context.Context, req GetRunReq) (GetRunResp, error) {
	conn := schemafdb.DB()
	queries := db.New(conn)

	run, err := queries.GetRun(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return GetRunResp{}, fmt.Errorf("run not found: %s", req.ID)
		}
		return GetRunResp{}, err
	}

	// Collect all nodes created in this run + cascading derivatives
	const collectQuery = `
WITH RECURSIVE run_nodes AS (
    SELECT DISTINCT e.target_node_id AS node_id
    FROM edges e
    WHERE e.run_id = $1 AND e.type = 'provenance/input'
    UNION
    SELECT DISTINCT e.target_node_id
    FROM edges e
    INNER JOIN run_nodes rn ON e.source_node_id = rn.node_id
    WHERE e.type = 'provenance/input'
)
SELECT node_id FROM run_nodes`

	rows, err := conn.QueryContext(ctx, collectQuery, req.ID)
	if err != nil {
		return GetRunResp{}, err
	}
	defer rows.Close()

	var nodeIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return GetRunResp{}, err
		}
		nodeIDs = append(nodeIDs, id)
	}

	// Fetch full node details
	nodes := make([]NodeResponse, 0, len(nodeIDs))
	for _, id := range nodeIDs {
		n, err := queries.GetNode(ctx, id)
		if err != nil {
			continue
		}
		nodes = append(nodes, nodeToResponse(n))
	}

	return GetRunResp{
		RunID:          run.ID,
		WorkerConfigID: run.WorkerConfigID,
		CreatedAt:      run.CreatedAt.Format("2006-01-02T15:04:05Z"),
		Nodes:          nodes,
	}, nil
}

type GetRunReq struct {
	ID string `path:"id"`
}

type GetRunResp struct {
	RunID          string         `json:"run_id"`
	WorkerConfigID string         `json:"worker_config_id"`
	CreatedAt      string         `json:"created_at"`
	Nodes          []NodeResponse `json:"nodes"`
}

// ─── Purge run ───────────────────────────────────────────────────────────────

// PurgeRunEndpoint deletes all edges and nodes created in a run, cascading
// to any nodes derived from them. L0 root nodes are never deleted.
type PurgeRunEndpoint struct{}

func (e PurgeRunEndpoint) Method() string { return "DELETE" }
func (e PurgeRunEndpoint) Path() string   { return "/api/runs/{id}" }
func (e PurgeRunEndpoint) Auth() bool     { return false }
func (e PurgeRunEndpoint) Handle(ctx context.Context, req PurgeRunReq) (PurgeRunResp, error) {
	conn := schemafdb.DB()

	// Find all target nodes created in this run (target = derived node)
	// then recursively find all nodes derived from those
	const collectQuery = `
WITH RECURSIVE run_nodes AS (
    -- Nodes directly created in this run (derived nodes = targets of provenance edges)
    SELECT DISTINCT e.target_node_id AS node_id
    FROM edges e
    WHERE e.run_id = $1 AND e.type = 'provenance/input'
    UNION
    -- Nodes derived from run nodes (cascade downstream)
    SELECT DISTINCT e.target_node_id
    FROM edges e
    INNER JOIN run_nodes rn ON e.source_node_id = rn.node_id
    WHERE e.type = 'provenance/input'
)
SELECT node_id FROM run_nodes`

	rows, err := conn.QueryContext(ctx, collectQuery, req.ID)
	if err != nil {
		return PurgeRunResp{}, fmt.Errorf("collect run nodes: %w", err)
	}
	defer rows.Close()

	var nodeIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return PurgeRunResp{}, err
		}
		nodeIDs = append(nodeIDs, id)
	}
	if rows.Err() != nil {
		return PurgeRunResp{}, rows.Err()
	}

	if len(nodeIDs) == 0 {
		return PurgeRunResp{NodesDeleted: 0, EdgesDeleted: 0}, nil
	}

	// Delete in a transaction: edges first, then nodes, then the run
	tx, err := conn.Begin()
	if err != nil {
		return PurgeRunResp{}, err
	}
	defer tx.Rollback()

	// Delete all edges connected to these nodes (in either direction)
	edgeResult, err := tx.ExecContext(ctx, `
		DELETE FROM edges WHERE source_node_id = ANY($1) OR target_node_id = ANY($1)
	`, pq.Array(nodeIDs))
	if err != nil {
		return PurgeRunResp{}, fmt.Errorf("delete edges: %w", err)
	}
	edgesDeleted, _ := edgeResult.RowsAffected()

	// Delete the nodes
	nodeResult, err := tx.ExecContext(ctx, `
		DELETE FROM nodes WHERE id = ANY($1)
	`, pq.Array(nodeIDs))
	if err != nil {
		return PurgeRunResp{}, fmt.Errorf("delete nodes: %w", err)
	}
	nodesDeleted, _ := nodeResult.RowsAffected()

	// Delete the run record
	tx.ExecContext(ctx, `DELETE FROM runs WHERE id = $1`, req.ID)

	if err := tx.Commit(); err != nil {
		return PurgeRunResp{}, fmt.Errorf("commit purge: %w", err)
	}

	return PurgeRunResp{
		NodesDeleted: int(nodesDeleted),
		EdgesDeleted: int(edgesDeleted),
	}, nil
}

type PurgeRunReq struct {
	ID string `path:"id"`
}

type PurgeRunResp struct {
	NodesDeleted int `json:"nodes_deleted"`
	EdgesDeleted int `json:"edges_deleted"`
}
