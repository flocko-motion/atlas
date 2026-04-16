package api

import (
	"context"
	"database/sql"
	"fmt"

	db "github.com/flocko-motion/rankedb/go/db"

	schemafdb "github.com/flocko-motion/schemaf/db"
)

// ─── Get edge ─────────────────────────────────────────────────────────────────

// GetEdgeEndpoint returns an edge by ID with its metadata.
type GetEdgeEndpoint struct{}

func (e GetEdgeEndpoint) Method() string { return "GET" }
func (e GetEdgeEndpoint) Path() string   { return "/api/edges/{id}" }
func (e GetEdgeEndpoint) Auth() bool     { return false }
func (e GetEdgeEndpoint) Handle(ctx context.Context, req GetEdgeReq) (EdgeResponse, error) {
	queries := db.New(schemafdb.DB())
	edge, err := queries.GetEdge(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return EdgeResponse{}, fmt.Errorf("edge not found: %s", req.ID)
		}
		return EdgeResponse{}, err
	}
	return edgeToResponse(edge), nil
}

type GetEdgeReq struct {
	ID string `path:"id"`
}

// ─── Get edge provenance ──────────────────────────────────────────────────────

// GetEdgeProvenanceEndpoint returns the provenance context of an edge: run, worker config node.
type GetEdgeProvenanceEndpoint struct{}

func (e GetEdgeProvenanceEndpoint) Method() string { return "GET" }
func (e GetEdgeProvenanceEndpoint) Path() string   { return "/api/edges/{id}/provenance" }
func (e GetEdgeProvenanceEndpoint) Auth() bool     { return false }
func (e GetEdgeProvenanceEndpoint) Handle(ctx context.Context, req GetEdgeProvenanceReq) (EdgeProvenanceResp, error) {
	queries := db.New(schemafdb.DB())

	edge, err := queries.GetEdge(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return EdgeProvenanceResp{}, fmt.Errorf("edge not found: %s", req.ID)
		}
		return EdgeProvenanceResp{}, err
	}

	resp := EdgeProvenanceResp{
		Edge: edgeToResponse(edge),
	}

	// If the edge has a run_id, fetch the run and worker config node
	if edge.RunID.Valid {
		resp.RunID = edge.RunID.String
		run, err := queries.GetRun(ctx, edge.RunID.String)
		if err == nil {
			workerNode, err := queries.GetNode(ctx, run.WorkerConfigID)
			if err == nil {
				nr := nodeToResponse(workerNode)
				resp.WorkerConfigNode = &nr
			}
		}
	}

	return resp, nil
}

type GetEdgeProvenanceReq struct {
	ID string `path:"id"`
}

type EdgeProvenanceResp struct {
	Edge             EdgeResponse  `json:"edge"`
	RunID            string        `json:"run_id,omitempty"`
	WorkerConfigNode *NodeResponse `json:"worker_config_node,omitempty"`
}
