package api

import "context"

// ─── Create edge ──────────────────────────────────────────────────────────────

// CreateEdgeEndpoint creates a provenance or semantic edge.
type CreateEdgeEndpoint struct{}

func (e CreateEdgeEndpoint) Method() string { return "POST" }
func (e CreateEdgeEndpoint) Path() string   { return "/api/edges" }
func (e CreateEdgeEndpoint) Auth() bool     { return false }
func (e CreateEdgeEndpoint) Handle(ctx context.Context, req CreateEdgeReq) (EdgeResponse, error) {
	// TODO: validate edge type, insert provenance or semantic edge
	return EdgeResponse{}, errNotImplemented
}

type CreateEdgeReq struct {
	SourceID   string   `json:"source_id"`
	TargetID   string   `json:"target_id"`
	Type       string   `json:"type"`       // provenance, head, tail
	RunID      string   `json:"run_id"`
	Confidence *float64 `json:"confidence"`
}

// ─── Get edge ─────────────────────────────────────────────────────────────────

// GetEdgeEndpoint returns an edge by ID with its metadata.
type GetEdgeEndpoint struct{}

func (e GetEdgeEndpoint) Method() string { return "GET" }
func (e GetEdgeEndpoint) Path() string   { return "/api/edges/{id}" }
func (e GetEdgeEndpoint) Auth() bool     { return false }
func (e GetEdgeEndpoint) Handle(ctx context.Context, req GetEdgeReq) (EdgeResponse, error) {
	// TODO: look up edge by ID in both tables
	return EdgeResponse{}, errNotImplemented
}

type GetEdgeReq struct {
	ID string `path:"id"`
}

// ─── Get edge provenance ──────────────────────────────────────────────────────

// GetEdgeProvenanceEndpoint returns the provenance context of an edge: run, tool node, config.
type GetEdgeProvenanceEndpoint struct{}

func (e GetEdgeProvenanceEndpoint) Method() string { return "GET" }
func (e GetEdgeProvenanceEndpoint) Path() string   { return "/api/edges/{id}/provenance" }
func (e GetEdgeProvenanceEndpoint) Auth() bool     { return false }
func (e GetEdgeProvenanceEndpoint) Handle(ctx context.Context, req GetEdgeProvenanceReq) (EdgeProvenanceResp, error) {
	// TODO: follow run_id to tool node
	return EdgeProvenanceResp{}, errNotImplemented
}

type GetEdgeProvenanceReq struct {
	ID string `path:"id"`
}

type EdgeProvenanceResp struct {
	Edge     EdgeResponse  `json:"edge"`
	RunID    string        `json:"run_id"`
	ToolNode *NodeResponse `json:"tool_node,omitempty"`
}
