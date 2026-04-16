package api

import "context"

// ─── Worker queue ─────────────────────────────────────────────────────────────

// GetQueueEndpoint returns unprocessed nodes for a given worker type.
type GetQueueEndpoint struct{}

func (e GetQueueEndpoint) Method() string { return "GET" }
func (e GetQueueEndpoint) Path() string   { return "/api/queue" }
func (e GetQueueEndpoint) Auth() bool     { return false }
func (e GetQueueEndpoint) Handle(ctx context.Context, req GetQueueReq) (GetQueueResp, error) {
	// TODO: find nodes not yet consumed by the given tool content_type
	return GetQueueResp{Nodes: []NodeResponse{}}, errNotImplemented
}

type GetQueueReq struct {
	ContentType   string  `query:"content_type"`
	Encoding      *string `query:"encoding"`
	NotConsumedBy *string `query:"not_consumed_by"`
	Limit         int     `query:"limit"`
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
	// TODO: validate tool_node_id exists, generate run_id
	return CreateRunResp{}, errNotImplemented
}

type CreateRunReq struct {
	ToolNodeID string `json:"tool_node_id"`
}

type CreateRunResp struct {
	RunID string `json:"run_id"`
}
