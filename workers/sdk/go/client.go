// Package rankedb provides a Go client for the RankeDB API.
package rankedb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client talks to a RankeDB server.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a RankeDB client for the given server URL.
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateNode creates a node with edges in a single atomic transaction.
func (c *Client) CreateNode(req CreateNodeRequest) (*NodeResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.HTTPClient.Post(c.BaseURL+"/api/nodes", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("POST /api/nodes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("POST /api/nodes returned %d: %s", resp.StatusCode, string(b))
	}

	var node NodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &node, nil
}

// GetNode fetches a node by ID.
func (c *Client) GetNode(id string) (*NodeResponse, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/api/nodes/" + url.PathEscape(id))
	if err != nil {
		return nil, fmt.Errorf("GET /api/nodes/%s: %w", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GET /api/nodes/%s returned %d: %s", id, resp.StatusCode, string(b))
	}

	var node NodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &node, nil
}

// CreateRun registers a new worker run and returns the run_id.
func (c *Client) CreateRun(workerConfigID string) (string, error) {
	body, err := json.Marshal(map[string]string{"worker_config_id": workerConfigID})
	if err != nil {
		return "", err
	}

	resp, err := c.HTTPClient.Post(c.BaseURL+"/api/runs", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("POST /api/runs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("POST /api/runs returned %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		RunID string `json:"run_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	return result.RunID, nil
}

// ListNodes lists nodes with optional filters.
func (c *Client) ListNodes(params ListNodesParams) ([]NodeResponse, error) {
	u, _ := url.Parse(c.BaseURL + "/api/nodes")
	q := u.Query()
	if params.ContentClass != "" {
		q.Set("content_class", params.ContentClass)
	}
	if params.ContentType != "" {
		q.Set("content_type", params.ContentType)
	}
	if params.ContentSha256 != "" {
		q.Set("content_sha256", params.ContentSha256)
	}
	if params.Limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", params.Limit))
	}
	u.RawQuery = q.Encode()

	resp, err := c.HTTPClient.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("GET /api/nodes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GET /api/nodes returned %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		Nodes []NodeResponse `json:"nodes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return result.Nodes, nil
}
