package rankedb

import (
	"encoding/json"
	"fmt"
)

// WorkerConfig holds the identity of a worker for registration with RankeDB.
type WorkerConfig struct {
	Name    string            `json:"name"`
	Version string            `json:"version"`
	Params  map[string]string `json:"params,omitempty"`
}

// EnsureWorkerConfig creates a worker/config node (L1) if it doesn't already exist,
// and returns its node ID. The node is idempotent by content — same config JSON
// produces the same sha256, and L0-style dedup catches it.
//
// Note: worker/config nodes are special — they're L1 nodes that serve as roots
// (no provenance edges). The server allows this for content_class=worker.
func (c *Client) EnsureWorkerConfig(cfg WorkerConfig) (string, error) {
	content, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("marshal worker config: %w", err)
	}
	contentStr := string(content)

	resp, err := c.CreateNode(CreateNodeRequest{
		Level:          0, // treated as root (idempotent by hash)
		ContentClass:   "worker",
		ContentType:    "config",
		EncodingClass:  "text",
		EncodingFormat: "json",
		Content:        &contentStr,
	})
	if err != nil {
		return "", fmt.Errorf("create worker config node: %w", err)
	}
	return resp.ID, nil
}

// StartRun registers a worker config and starts a new run, returning both IDs.
func (c *Client) StartRun(cfg WorkerConfig) (configID string, runID string, err error) {
	configID, err = c.EnsureWorkerConfig(cfg)
	if err != nil {
		return "", "", err
	}
	runID, err = c.CreateRun(configID)
	if err != nil {
		return "", "", err
	}
	return configID, runID, nil
}
