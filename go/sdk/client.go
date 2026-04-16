// Package rankedb provides a worker-friendly wrapper around the generated RankeDB API client.
package rankedb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/flocko-motion/rankedb/go/apiclient"
)

// Client wraps the generated API client with worker convenience methods.
type Client struct {
	api    *apiclient.ClientWithResponses
	server string
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
func (c *Client) CreateNode(ctx context.Context, req CreateNodeRequest) (*apiclient.NodeResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
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

// CreateRun registers a new worker run and returns the run_id.
func (c *Client) CreateRun(ctx context.Context, workerConfigID string) (string, error) {
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
