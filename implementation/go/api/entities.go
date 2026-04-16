package api

import "context"

// ─── Get entity ───────────────────────────────────────────────────────────────

// GetEntityEndpoint returns an entity with all its relations, sorted by temporal validity.
type GetEntityEndpoint struct{}

func (e GetEntityEndpoint) Method() string { return "GET" }
func (e GetEntityEndpoint) Path() string   { return "/api/entities/{id}" }
func (e GetEntityEndpoint) Auth() bool     { return false }
func (e GetEntityEndpoint) Handle(ctx context.Context, req GetEntityReq) (GetEntityResp, error) {
	// TODO: fetch entity node + all connected relations via semantic edges
	return GetEntityResp{}, errNotImplemented
}

type GetEntityReq struct {
	ID            string   `path:"id"`
	MinConfidence *float64 `query:"min_confidence"`
}

type GetEntityResp struct {
	Entity    NodeResponse       `json:"entity"`
	Relations []RelationResponse `json:"relations"`
}

// ─── Search entities ──────────────────────────────────────────────────────────

// SearchEntitiesEndpoint searches entities by full-text query over names and aliases.
type SearchEntitiesEndpoint struct{}

func (e SearchEntitiesEndpoint) Method() string { return "GET" }
func (e SearchEntitiesEndpoint) Path() string   { return "/api/entities" }
func (e SearchEntitiesEndpoint) Auth() bool     { return false }
func (e SearchEntitiesEndpoint) Handle(ctx context.Context, req SearchEntitiesReq) (SearchEntitiesResp, error) {
	// TODO: full-text search over L2 entity nodes
	return SearchEntitiesResp{Entities: []NodeResponse{}}, errNotImplemented
}

type SearchEntitiesReq struct {
	Q      *string `query:"q"`
	Type   *string `query:"type"` // e.g. entity/person, entity/organization
	Limit  int     `query:"limit"`
	Offset int     `query:"offset"`
}

type SearchEntitiesResp struct {
	Entities []NodeResponse `json:"entities"`
}

// ─── Entity timeline ──────────────────────────────────────────────────────────

// GetEntityTimelineEndpoint returns all relations of an entity sorted chronologically by valid_from.
type GetEntityTimelineEndpoint struct{}

func (e GetEntityTimelineEndpoint) Method() string { return "GET" }
func (e GetEntityTimelineEndpoint) Path() string   { return "/api/entities/{id}/timeline" }
func (e GetEntityTimelineEndpoint) Auth() bool     { return false }
func (e GetEntityTimelineEndpoint) Handle(ctx context.Context, req GetEntityTimelineReq) (GetEntityTimelineResp, error) {
	// TODO: fetch relations sorted by valid_from
	return GetEntityTimelineResp{Relations: []RelationResponse{}}, errNotImplemented
}

type GetEntityTimelineReq struct {
	ID string `path:"id"`
}

type GetEntityTimelineResp struct {
	Relations []RelationResponse `json:"relations"`
}

// ─── List relations ───────────────────────────────────────────────────────────

// ListRelationsEndpoint returns filtered relation nodes.
type ListRelationsEndpoint struct{}

func (e ListRelationsEndpoint) Method() string { return "GET" }
func (e ListRelationsEndpoint) Path() string   { return "/api/relations" }
func (e ListRelationsEndpoint) Auth() bool     { return false }
func (e ListRelationsEndpoint) Handle(ctx context.Context, req ListRelationsReq) (ListRelationsResp, error) {
	// TODO: filter relation nodes by type, unresolved status
	return ListRelationsResp{Relations: []RelationResponse{}}, errNotImplemented
}

type ListRelationsReq struct {
	Unresolved *bool   `query:"unresolved"`
	Type       *string `query:"type"` // e.g. relation/alias, relation/has_role
	Limit      int     `query:"limit"`
	Offset     int     `query:"offset"`
}

type ListRelationsResp struct {
	Relations []RelationResponse `json:"relations"`
}
