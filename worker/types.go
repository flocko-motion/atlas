package rankedb

import "github.com/flocko-motion/rankedb/worker/apiclient"

// NodeResponse is the API response for a node.
// Re-exported so workers don't need to import the generated apiclient package.
type NodeResponse = apiclient.NodeResponse

// CreateNodeRequest is the JSON body for POST /api/nodes.
// This is defined here because the generated client doesn't include
// a request body type for the RawRoute endpoint.
type CreateNodeRequest struct {
	Level                 int        `json:"level"`
	ContentClass          string     `json:"content_class"`
	ContentType           string     `json:"content_type"`
	EncodingClass         string     `json:"encoding_class"`
	EncodingFormat        string     `json:"encoding_format"`
	Content               *string    `json:"content,omitempty"`
	ArtifactCreatedAt     *string    `json:"artifact_created_at,omitempty"`
	ArtifactCreatedAtBlur *string    `json:"artifact_created_at_blur,omitempty"`
	Origin                *string    `json:"origin,omitempty"`
	OriginalName          *string    `json:"original_name,omitempty"`
	ValidFrom             *string    `json:"valid_from,omitempty"`
	ValidFromBlur         *string    `json:"valid_from_blur,omitempty"`
	ValidUntil            *string    `json:"valid_until,omitempty"`
	ValidUntilBlur        *string    `json:"valid_until_blur,omitempty"`
	Confidence            *float64   `json:"confidence,omitempty"`
	Edges                 []EdgeSpec `json:"edges,omitempty"`
}

// EdgeSpec defines an edge to create alongside a node.
type EdgeSpec struct {
	SourceNodeID string   `json:"source_node_id,omitempty"`
	TargetNodeID string   `json:"target_node_id,omitempty"`
	Type         string   `json:"type"`
	Confidence   *float64 `json:"confidence,omitempty"`
	RunID        *string  `json:"run_id,omitempty"`
}

// Ptr returns a pointer to the given string.
func Ptr(s string) *string { return &s }

// PtrF returns a pointer to the given float64.
func PtrF(f float64) *float64 { return &f }
