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
	api       *apiclient.ClientWithResponses
	server    string
	DryRun    bool     // when true, print write calls instead of sending them
	Verbosity LogLevel // LogInfo (default), LogDebug, or LogWarn

	// Set by StartRun — used by Done() to stamp provenance automatically.
	configID string
	runID    string
}

// NewClient creates a RankeDB worker client for the given server URL.
func NewClient(server string) (*Client, error) {
	api, err := apiclient.NewClientWithResponses(server)
	if err != nil {
		return nil, fmt.Errorf("create api client: %w", err)
	}
	c := &Client{api: api, server: server, Verbosity: LogInfo}
	c.log(LogInfo, "connected to %s", server)
	return c, nil
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
		c.log(LogDebug, "[dry-run] POST /api/nodes %s/%s: %s", req.ContentClass, req.ContentType, body)
		id := fmt.Sprintf("dry-run-%d", dryRunCounter)
		dryRunCounter++
		return &apiclient.NodeResponse{Id: id}, nil
	}

	c.log(LogDebug, "POST /api/nodes %s/%s (%d bytes)", req.ContentClass, req.ContentType, len(body))

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
	c.log(LogDebug, "created node %s (%s/%s)", node.Id, req.ContentClass, req.ContentType)
	return &node, nil
}

var dryRunCounter int

// QueueParams defines what source nodes the worker wants.
// Filtering granularity is determined automatically:
//   - Default (after StartRun): by_config — skip sources this exact config already processed.
//   - Override with ByWorker: skip if any config with that worker name processed it.
//   - Override with ByClass: skip if any derived node of that content class exists.
type QueueParams struct {
	ContentClass string // required: content class to look for (e.g. "source")
	ContentType  string // required: content type to look for (e.g. "bulk")
	ByClass      string // override: skip if derived class exists (e.g. "classification")
	ByWorker     string // override: skip if worker name has processed (e.g. "google-contacts")
	Limit        int    // max results (default 100)
}

// Queue returns source nodes that haven't been processed yet.
// Defaults to by_config filtering using the config from StartRun.
// Set ByWorker or ByClass to override with coarser granularity.
func (c *Client) Queue(ctx context.Context, params QueueParams) ([]apiclient.NodeResponse, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 100
	}

	query := fmt.Sprintf("/api/queue?content_class=%s&content_type=%s&limit=%d",
		params.ContentClass, params.ContentType, limit)

	var filterDesc string
	switch {
	case params.ByClass != "":
		query += "&by_class=" + params.ByClass
		filterDesc = "by_class=" + params.ByClass
	case params.ByWorker != "":
		query += "&by_worker=" + params.ByWorker
		filterDesc = "by_worker=" + params.ByWorker
	case c.configID != "":
		query += "&by_config=" + c.configID
		filterDesc = "by_config"
	default:
		filterDesc = "none"
	}

	c.log(LogDebug, "GET /api/queue %s/%s filter=%s limit=%d", params.ContentClass, params.ContentType, filterDesc, limit)

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
	c.log(LogInfo, "queue: %d %s/%s nodes to process", len(result.Nodes), params.ContentClass, params.ContentType)
	return result.Nodes, nil
}

// Done signals that the worker has finished processing a source node.
// For bulk sources, this creates an observation/processed marker so the
// queue won't serve the source again. For non-bulk sources, the L1 output
// nodes already serve as implicit proof of processing — this is a no-op.
// Requires StartRun to have been called first.
func (c *Client) Done(ctx context.Context, source *apiclient.NodeResponse) error {
	if c.runID == "" {
		return fmt.Errorf("Done called before StartRun")
	}
	if source.ContentType != "bulk" {
		c.log(LogDebug, "done with %s (non-bulk, no marker needed)", source.Id)
		return nil
	}
	c.log(LogInfo, "marking bulk source %s as processed", source.Id)
	return c.markProcessed(ctx, source.Id)
}

// markProcessed creates a minimal L1 observation/processed node with provenance.
func (c *Client) markProcessed(ctx context.Context, sourceID string) error {
	content := "processed"
	_, err := c.CreateNode(ctx, CreateNodeRequest{
		Level:          1,
		ContentClass:   "observation",
		ContentType:    "processed",
		EncodingClass:  "text",
		EncodingFormat: "plain",
		Content:        &content,
		Edges: []EdgeSpec{
			{Type: "provenance/input", TargetNodeID: sourceID, RunID: &c.runID},
			{Type: "provenance/worker", TargetNodeID: c.configID, RunID: &c.runID},
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
		c.log(LogInfo, "[dry-run] create run for config %s", workerConfigID)
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
	c.log(LogDebug, "created run %s", resp.JSON200.RunId)
	return resp.JSON200.RunId, nil
}
