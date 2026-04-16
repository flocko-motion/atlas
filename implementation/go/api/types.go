package api

import (
	"errors"
	"time"
)

// errNotImplemented is returned by stub endpoints during scaffolding.
var errNotImplemented = errors.New("not implemented")

// NodeResponse is the API representation of a graph node.
type NodeResponse struct {
	ID                    string   `json:"id"`
	Level                 int      `json:"level"`
	ContentType           string   `json:"content_type"`
	Encoding              string   `json:"encoding"`
	ContentSha256         string   `json:"content_sha256"`
	ContentLen            int64    `json:"content_len"`
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

// ProvenanceEdgeResponse is the API representation of a provenance edge.
type ProvenanceEdgeResponse struct {
	ID        string `json:"id"`
	SourceID  string `json:"source_id"`
	TargetID  string `json:"target_id"`
	RunID     string `json:"run_id"`
	CreatedAt string `json:"created_at"`
}

// SemanticEdgeResponse is the API representation of a semantic edge.
type SemanticEdgeResponse struct {
	ID         string `json:"id"`
	RelationID string `json:"relation_id"`
	HeadID     string `json:"head_id"`
	TailID     string `json:"tail_id"`
	CreatedAt  string `json:"created_at"`
}

// EdgeResponse wraps both edge types for the unified edges API.
type EdgeResponse struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	SourceID   string  `json:"source_id,omitempty"`
	TargetID   string  `json:"target_id,omitempty"`
	RelationID string  `json:"relation_id,omitempty"`
	HeadID     string  `json:"head_id,omitempty"`
	TailID     string  `json:"tail_id,omitempty"`
	RunID      string  `json:"run_id,omitempty"`
	Confidence *float64 `json:"confidence,omitempty"`
	CreatedAt  string  `json:"created_at"`
}

// RelationResponse is an entity's relation with full context.
type RelationResponse struct {
	Node       NodeResponse `json:"node"`
	HeadID     string       `json:"head_id"`
	TailID     string       `json:"tail_id"`
}

// ProvenanceSubgraph is a set of nodes and edges forming a provenance chain.
type ProvenanceSubgraph struct {
	Nodes []NodeResponse           `json:"nodes"`
	Edges []ProvenanceEdgeResponse `json:"edges"`
}

// Pagination helpers
type PaginationParams struct {
	Limit  int `query:"limit"`
	Offset int `query:"offset"`
}

// stub returns the current time as an RFC3339 string.
func now() string {
	return time.Now().Format(time.RFC3339)
}
