package api

import (
	"database/sql"
	"errors"
	"time"

	db "rankedb/db"
)

// errNotImplemented is returned by stub endpoints during scaffolding.
var errNotImplemented = errors.New("not implemented")

// NodeResponse is the API representation of a graph node.
type NodeResponse struct {
	ID                    string   `json:"id"`
	Level                 int      `json:"level"`
	ContentClass          string   `json:"content_class"`
	ContentType           string   `json:"content_type"`
	EncodingClass         string   `json:"encoding_class"`
	EncodingFormat        string   `json:"encoding_format"`
	ContentSha256         string   `json:"content_sha256"`
	ContentLen            int64    `json:"content_len"`
	Title                 *string  `json:"title,omitempty"`
	Content               *string  `json:"content,omitempty"`
	CreatedAt             string   `json:"created_at"`
	ArtifactCreatedAt     *string  `json:"artifact_created_at,omitempty"`
	ArtifactCreatedAtBlur *string  `json:"artifact_created_at_blur,omitempty"`
	Origin                *string  `json:"origin,omitempty"`
	OriginalName          *string  `json:"original_name,omitempty"`
	ValidFrom             *string  `json:"valid_from,omitempty"`
	ValidFromBlur         *string  `json:"valid_from_blur,omitempty"`
	ValidUntil            *string  `json:"valid_until,omitempty"`
	ValidUntilBlur        *string  `json:"valid_until_blur,omitempty"`
	Confidence            *float64 `json:"confidence,omitempty"`
}

// EdgeResponse is the unified API representation of any edge.
type EdgeResponse struct {
	ID           string   `json:"id"`
	SourceNodeID string   `json:"source_node_id"`
	TargetNodeID string   `json:"target_node_id"`
	Type         string   `json:"type"`
	Confidence   *float64 `json:"confidence,omitempty"`
	RunID        *string  `json:"run_id,omitempty"`
	CreatedAt    string   `json:"created_at"`
}

// RelationResponse is a relation node with its head and tail edges.
type RelationResponse struct {
	Node      NodeResponse   `json:"node"`
	HeadEdges []EdgeResponse `json:"head_edges"`
	TailEdges []EdgeResponse `json:"tail_edges"`
}

// ProvenanceSubgraph is a set of nodes and edges forming a provenance chain.
type ProvenanceSubgraph struct {
	Nodes []NodeResponse `json:"nodes"`
	Edges []EdgeResponse `json:"edges"`
}

// Pagination helpers
type PaginationParams struct {
	Limit  int `query:"limit"`
	Offset int `query:"offset"`
}

// now returns the current time as an RFC3339 string.
func now() string {
	return time.Now().Format(time.RFC3339)
}

// defaultLimit returns the given limit, or 50 if it is zero.
func defaultLimit(limit int) int32 {
	if limit <= 0 {
		return 50
	}
	return int32(limit)
}

// nullString converts a sql.NullString to *string.
func nullString(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

// nullTime converts a sql.NullTime to *string (RFC3339).
func nullTime(nt sql.NullTime) *string {
	if nt.Valid {
		s := nt.Time.Format(time.RFC3339)
		return &s
	}
	return nil
}

// nullFloat64 converts a sql.NullFloat64 to *float64.
func nullFloat64(nf sql.NullFloat64) *float64 {
	if nf.Valid {
		return &nf.Float64
	}
	return nil
}

// nodeToResponse converts a db.Node to a NodeResponse.
func nodeToResponse(n db.Node) NodeResponse {
	resp := NodeResponse{
		ID:                    n.ID,
		Level:                 int(n.Level),
		ContentClass:          n.ContentClass,
		ContentType:           n.ContentType,
		EncodingClass:         n.EncodingClass,
		EncodingFormat:        n.EncodingFormat,
		ContentSha256:         n.ContentSha256,
		ContentLen:            n.ContentLen,
		Title:                 nullString(n.Title),
		CreatedAt:             n.CreatedAt.Format(time.RFC3339),
		ArtifactCreatedAt:     nullTime(n.ArtifactCreatedAt),
		ArtifactCreatedAtBlur: nullString(n.ArtifactCreatedAtBlur),
		Origin:                nullString(n.Origin),
		OriginalName:          nullString(n.OriginalName),
		ValidFrom:             nullTime(n.ValidFrom),
		ValidFromBlur:         nullString(n.ValidFromBlur),
		ValidUntil:            nullTime(n.ValidUntil),
		ValidUntilBlur:        nullString(n.ValidUntilBlur),
		Confidence:            nullFloat64(n.Confidence),
		Content:               nullString(n.ContentCached),
	}
	return resp
}

// edgeToResponse converts a db.Edge to an EdgeResponse.
func edgeToResponse(e db.Edge) EdgeResponse {
	return EdgeResponse{
		ID:           e.ID,
		SourceNodeID: e.SourceNodeID,
		TargetNodeID: e.TargetNodeID,
		Type:         e.Type,
		Confidence:   nullFloat64(e.Confidence),
		RunID:        nullString(e.RunID),
		CreatedAt:    e.CreatedAt.Format(time.RFC3339),
	}
}
