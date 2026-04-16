package api

import (
	"context"
	"net/http"
)

// ─── Create node ──────────────────────────────────────────────────────────────

// CreateNodeEndpoint creates a graph node.
// L0 sources use multipart form upload; L1/L2 use JSON.
// L0 root creation is idempotent via content_sha256.
type CreateNodeEndpoint struct{}

func (e CreateNodeEndpoint) Method() string { return "POST" }
func (e CreateNodeEndpoint) Path() string   { return "/api/nodes" }
func (e CreateNodeEndpoint) Auth() bool     { return false }

func (e CreateNodeEndpoint) HandleRaw(w http.ResponseWriter, r *http.Request) error {
	// TODO: detect multipart vs JSON, parse accordingly, insert node + provenance edges
	return errNotImplemented
}

// ─── Get node ─────────────────────────────────────────────────────────────────

// GetNodeEndpoint returns a node by ID with full metadata and inline content.
type GetNodeEndpoint struct{}

func (e GetNodeEndpoint) Method() string { return "GET" }
func (e GetNodeEndpoint) Path() string   { return "/api/nodes/{id}" }
func (e GetNodeEndpoint) Auth() bool     { return false }
func (e GetNodeEndpoint) Handle(ctx context.Context, req GetNodeReq) (NodeResponse, error) {
	// TODO: fetch node from DB, resolve content from cache or S3
	return NodeResponse{}, errNotImplemented
}

type GetNodeReq struct {
	ID string `path:"id"`
}

// ─── Get node content ─────────────────────────────────────────────────────────

// GetNodeContentEndpoint returns the raw content bytes of a node.
// Serves from Postgres cache if available, otherwise proxies from S3.
type GetNodeContentEndpoint struct{}

func (e GetNodeContentEndpoint) Method() string { return "GET" }
func (e GetNodeContentEndpoint) Path() string   { return "/api/nodes/{id}/content" }
func (e GetNodeContentEndpoint) Auth() bool     { return false }

func (e GetNodeContentEndpoint) HandleRaw(w http.ResponseWriter, r *http.Request) error {
	// TODO: resolve content from cache or S3, stream to response
	return errNotImplemented
}

// ─── Get node provenance ──────────────────────────────────────────────────────

// GetNodeProvenanceEndpoint returns the upstream provenance chain from a node back to L0 roots.
type GetNodeProvenanceEndpoint struct{}

func (e GetNodeProvenanceEndpoint) Method() string { return "GET" }
func (e GetNodeProvenanceEndpoint) Path() string   { return "/api/nodes/{id}/provenance" }
func (e GetNodeProvenanceEndpoint) Auth() bool     { return false }
func (e GetNodeProvenanceEndpoint) Handle(ctx context.Context, req GetNodeProvenanceReq) (ProvenanceSubgraph, error) {
	// TODO: recursive traversal of provenance edges to L0 roots
	return ProvenanceSubgraph{}, errNotImplemented
}

type GetNodeProvenanceReq struct {
	ID string `path:"id"`
}

// ─── List nodes ───────────────────────────────────────────────────────────────

// ListNodesEndpoint returns a filtered, paginated list of nodes.
type ListNodesEndpoint struct{}

func (e ListNodesEndpoint) Method() string { return "GET" }
func (e ListNodesEndpoint) Path() string   { return "/api/nodes" }
func (e ListNodesEndpoint) Auth() bool     { return false }
func (e ListNodesEndpoint) Handle(ctx context.Context, req ListNodesReq) (ListNodesResp, error) {
	// TODO: build filtered query from parameters
	return ListNodesResp{Nodes: []NodeResponse{}}, errNotImplemented
}

type ListNodesReq struct {
	Level         *int    `query:"level"`
	ContentType   *string `query:"content_type"`
	Encoding      *string `query:"encoding"`
	CreatedAfter  *string `query:"created_after"`
	CreatedBefore *string `query:"created_before"`
	RunID         *string `query:"run_id"`
	Limit         int     `query:"limit"`
	Offset        int     `query:"offset"`
}

type ListNodesResp struct {
	Nodes []NodeResponse `json:"nodes"`
}

// ─── Get node edges ───────────────────────────────────────────────────────────

// GetNodeEdgesEndpoint returns edges connected to a node, filterable by direction and type.
type GetNodeEdgesEndpoint struct{}

func (e GetNodeEdgesEndpoint) Method() string { return "GET" }
func (e GetNodeEdgesEndpoint) Path() string   { return "/api/nodes/{id}/edges" }
func (e GetNodeEdgesEndpoint) Auth() bool     { return false }
func (e GetNodeEdgesEndpoint) Handle(ctx context.Context, req GetNodeEdgesReq) (GetNodeEdgesResp, error) {
	// TODO: query provenance + semantic edges by node ID with filters
	return GetNodeEdgesResp{Edges: []EdgeResponse{}}, errNotImplemented
}

type GetNodeEdgesReq struct {
	ID        string  `path:"id"`
	Direction *string `query:"direction"` // incoming, outgoing, both
	Type      *string `query:"type"`      // provenance, head, tail, all
	Limit     int     `query:"limit"`
	Offset    int     `query:"offset"`
}

type GetNodeEdgesResp struct {
	Edges []EdgeResponse `json:"edges"`
}
