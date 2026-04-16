// Package rankedb provides a worker-friendly wrapper around the generated RankeDB API client.
package rankedb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/flocko-motion/rankedb/worker/apiclient"
)

// Client wraps the generated API client with worker convenience methods.
type Client struct {
	api    *apiclient.ClientWithResponses
	server string
	DryRun bool // when true, print write calls instead of sending them
}

// NewClient creates a RankeDB worker client for the given server URL.
func NewClient(server string) (*Client, error) {
	api, err := apiclient.NewClientWithResponses(server)
	if err != nil {
		return nil, fmt.Errorf("create api client: %w", err)
	}
	return &Client{api: api, server: server}, nil
}

// CreateNode creates a node with edges in a single atomic transaction.
// This uses a manual POST because the generated client doesn't support
// a request body for this endpoint (it's a RawRoute in schemaf).
// In DryRun mode, prints the request and returns a placeholder response.
func (c *Client) CreateNode(ctx context.Context, req CreateNodeRequest) (*apiclient.NodeResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	if c.DryRun {
		fmt.Printf("[DRY RUN] POST /api/nodes\n%s\n\n", body)
		id := fmt.Sprintf("dry-run-%d", dryRunCounter)
		dryRunCounter++
		return &apiclient.NodeResponse{Id: id}, nil
	}

	resp, err := http.Post(c.server+"/api/nodes", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("POST /api/nodes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("POST /api/nodes returned %d: %s", resp.StatusCode, string(b))
	}

	var node apiclient.NodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &node, nil
}

var dryRunCounter int

// QueueParams defines filters for finding unprocessed nodes.
type QueueParams struct {
	ContentClass  string // required: content class to look for (e.g. "source")
	ContentType   string // required: content type to look for (e.g. "bulk")
	NotConsumedBy string // content class that should NOT already have derived from this node
	Limit         int    // max results (default 100)
}

// Queue returns nodes that haven't been processed yet by a worker of the given output class.
func (c *Client) Queue(ctx context.Context, params QueueParams) ([]apiclient.NodeResponse, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 100
	}

	query := fmt.Sprintf("/api/queue?content_class=%s&content_type=%s&not_consumed_by=%s&limit=%d",
		params.ContentClass, params.ContentType, params.NotConsumedBy, limit)

	resp, err := http.Get(c.server + query)
	if err != nil {
		return nil, fmt.Errorf("GET /api/queue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GET /api/queue returned %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		Nodes []apiclient.NodeResponse `json:"nodes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode queue response: %w", err)
	}
	return result.Nodes, nil
}

// MarkProcessed marks a source node as processed by this worker.
// Creates a minimal L1 observation/processed node with provenance.
// The queue will no longer offer this source to the same worker type.
func (c *Client) MarkProcessed(ctx context.Context, sourceID, configID, runID string, reason string) error {
	content := reason
	if content == "" {
		content = "processed"
	}
	_, err := c.CreateNode(ctx, CreateNodeRequest{
		Level:          1,
		ContentClass:   "observation",
		ContentType:    "processed",
		EncodingClass:  "text",
		EncodingFormat: "plain",
		Content:        &content,
		Edges: []EdgeSpec{
			{Type: "provenance/input", TargetNodeID: sourceID, RunID: &runID},
			{Type: "provenance/worker", TargetNodeID: configID, RunID: &runID},
		},
	})
	return err
}

// GetNode fetches a node by ID.
func (c *Client) GetNode(ctx context.Context, id string) (*apiclient.NodeResponse, error) {
	resp, err := c.api.GetApiNodesIdWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get node %s: %w", id, err)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("get node %s: status %d", id, resp.StatusCode())
	}
	return resp.JSON200, nil
}

// GetNodeContent downloads the raw content bytes of a node.
func (c *Client) GetNodeContent(ctx context.Context, id string) ([]byte, error) {
	resp, err := http.Get(c.server + "/api/nodes/" + id + "/content")
	if err != nil {
		return nil, fmt.Errorf("get content %s: %w", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get content %s: status %d: %s", id, resp.StatusCode, string(b))
	}

	return io.ReadAll(resp.Body)
}

// CreateRun registers a new worker run and returns the run_id.
// In DryRun mode, returns a placeholder run ID.
func (c *Client) CreateRun(ctx context.Context, workerConfigID string) (string, error) {
	if c.DryRun {
		fmt.Printf("[DRY RUN] POST /api/runs {worker_config_id: %s}\n\n", workerConfigID)
		return "dry-run-id", nil
	}

	resp, err := c.api.PostApiRunsWithResponse(ctx, apiclient.CreateRunReq{
		WorkerConfigId: workerConfigID,
	})
	if err != nil {
		return "", fmt.Errorf("create run: %w", err)
	}
	if resp.JSON200 == nil {
		return "", fmt.Errorf("create run: status %d", resp.StatusCode())
	}
	return resp.JSON200.RunId, nil
}
