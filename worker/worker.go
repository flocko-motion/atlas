package rankedb

import (
	"context"
	"encoding/json"
	"fmt"
)

// WorkerConfig holds the identity of a worker for registration with RankeDB.
type WorkerConfig struct {
	Name    string            `json:"name"`
	Version string            `json:"version"`
	Params  map[string]string `json:"params,omitempty"`
}

// EnsureWorkerConfig creates a worker/config node if it doesn't already exist,
// and returns its node ID. Idempotent by content hash.
func (c *Client) EnsureWorkerConfig(ctx context.Context, cfg WorkerConfig) (string, error) {
	c.log(LogDebug, "ensuring worker config %s v%s", cfg.Name, cfg.Version)
	content, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("marshal worker config: %w", err)
	}
	contentStr := string(content)

	resp, err := c.CreateNode(ctx, CreateNodeRequest{
		Level:          0,
		ContentClass:   "worker",
		ContentType:    "config",
		EncodingClass:  "text",
		EncodingFormat: "json",
		Content:        &contentStr,
	})
	if err != nil {
		return "", fmt.Errorf("create worker config node: %w", err)
	}
	c.log(LogDebug, "worker config node: %s", resp.Id)
	return resp.Id, nil
}

// StartRun registers a worker config and starts a new run, returning both IDs.
// The IDs are also stored on the client for use by Done().
func (c *Client) StartRun(ctx context.Context, cfg WorkerConfig) (configID string, runID string, err error) {
	c.log(LogInfo, "starting run for %s v%s", cfg.Name, cfg.Version)
	configID, err = c.EnsureWorkerConfig(ctx, cfg)
	if err != nil {
		return "", "", err
	}
	runID, err = c.CreateRun(ctx, configID)
	if err != nil {
		return "", "", err
	}
	c.configID = configID
	c.runID = runID
	c.log(LogInfo, "run started (config=%s run=%s)", configID, runID)
	return configID, runID, nil
}
