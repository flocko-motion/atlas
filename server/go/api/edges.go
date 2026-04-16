package api

import (
	"context"
	"database/sql"
	"fmt"

	db "rankedb/db"

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

// ─── List edges ──────────────────────────────────────────────────────────────

// ListEdgesEndpoint returns edges with optional type filter.
type ListEdgesEndpoint struct{}

func (e ListEdgesEndpoint) Method() string { return "GET" }
func (e ListEdgesEndpoint) Path() string   { return "/api/edges" }
func (e ListEdgesEndpoint) Auth() bool     { return false }
func (e ListEdgesEndpoint) Handle(ctx context.Context, req ListEdgesReq) (ListEdgesResp, error) {
	queries := db.New(schemafdb.DB())
	limit := defaultLimit(req.Limit)
	offset := int32(req.Offset)

	var dbEdges []db.Edge
	var err error

	if req.Type != nil && *req.Type != "" {
		dbEdges, err = queries.ListEdgesByType(ctx, db.ListEdgesByTypeParams{
			Type:   *req.Type,
			Limit:  limit,
			Offset: offset,
		})
	} else {
		dbEdges, err = queries.ListEdges(ctx, db.ListEdgesParams{
			Limit:  limit,
			Offset: offset,
		})
	}
	if err != nil {
		return ListEdgesResp{Edges: []EdgeResponse{}}, err
	}

	edges := make([]EdgeResponse, 0, len(dbEdges))
	for _, edge := range dbEdges {
		edges = append(edges, edgeToResponse(edge))
	}
	return ListEdgesResp{Edges: edges}, nil
}

type ListEdgesReq struct {
	Type   *string `query:"type"`
	Limit  int     `query:"limit"`
	Offset int     `query:"offset"`
}

type ListEdgesResp struct {
	Edges []EdgeResponse `json:"edges"`
}
