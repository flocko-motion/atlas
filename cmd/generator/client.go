// package: main / cmd
// type:    logic
// job:     the REST client the generator contributes through
// limits:  transport only; the graph it carries is the grower's (-> graph.go)
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/flocko-motion/ranke-go"

	"github.com/flocko-motion/rankedb/openapi"
)

// mediaCBORSeq is what POST /contribute reads: an RFC 8742 CBOR sequence, self-framing
// so a contribution of any size streams.
const mediaCBORSeq = "application/cbor-seq"

// client talks to one ranke-db over HTTP, presenting at most one credential — the
// endpoint rejects two as ambiguous.
type client struct {
	base   string
	token  string
	apiKey string
	http   *http.Client
}

// newClient points a client at a base URL, defaulting a bare host:port to http.
func newClient(base, token, apiKey string) *client {
	if !strings.Contains(base, "://") {
		base = "http://" + base
	}
	return &client{
		base:   strings.TrimRight(base, "/"),
		token:  token,
		apiKey: apiKey,
		http:   &http.Client{Timeout: 60 * time.Second},
	}
}

// contribute merges one contribution and returns the new branch-table head with the
// ids that landed.
func (c *client) contribute(ctx context.Context, body []byte) (openapi.ContributionResult, error) {
	var res openapi.ContributionResult
	out, err := c.do(ctx, http.MethodPost, "/contribute", mediaCBORSeq, body)
	if err != nil {
		return res, err
	}
	if err := json.Unmarshal(out, &res); err != nil {
		return res, fmt.Errorf("decode contribution result: %w", err)
	}
	return res, nil
}

// head reads a branch's current head claim id.
func (c *client) head(ctx context.Context, branch string) (string, error) {
	out, err := c.do(ctx, http.MethodGet, "/"+url.PathEscape(branch)+"/head", "", nil)
	if err != nil {
		return "", err
	}
	var res openapi.BranchHead
	if err := json.Unmarshal(out, &res); err != nil {
		return "", fmt.Errorf("decode branch head: %w", err)
	}
	return res.Head, nil
}

// waitReady polls /health until the server answers, so a script can launch a server
// and seed it without sleeping a guessed interval. A refusal that is not the listener
// missing — an auth failure, say — stops the wait: retrying it would never succeed.
func (c *client) waitReady(ctx context.Context, within time.Duration) error {
	deadline := time.Now().Add(within)
	for {
		_, err := c.do(ctx, http.MethodGet, "/health", "", nil)
		if err == nil {
			return nil
		}
		var refused *apiRefusal
		if errors.As(err, &refused) {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("no answer from %s/health within %s: %w", c.base, within, err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// do sends one request with whatever credential was configured and returns the body,
// mapping a non-2xx onto the contract's Error payload.
func (c *client) do(ctx context.Context, method, path, contentType string, body []byte) ([]byte, error) {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, rdr)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	switch {
	case c.token != "":
		req.Header.Set("Authorization", "Bearer "+c.token)
	case c.apiKey != "":
		req.Header.Set("X-API-Key", c.apiKey)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	out, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s %s: %w", method, path, err)
	}
	if res.StatusCode/100 != 2 {
		return nil, refusal(res.StatusCode, out)
	}
	return out, nil
}

// apiRefusal is an answer from a server that declined: the status it gave, and the
// contract's error code where it sent one. Distinguishing it from a transport failure
// is what lets waitReady tell "not up yet" from "up, and saying no".
type apiRefusal struct {
	status int
	code   string
	msg    string
}

func (e *apiRefusal) Error() string {
	if e.code != "" {
		return fmt.Sprintf("%s (http %d): %s", e.code, e.status, e.msg)
	}
	return fmt.Sprintf("http %d: %s", e.status, e.msg)
}

// refusal reads the contract's Error payload, falling back to the raw body when the
// answer came from something else — a proxy, say.
func refusal(status int, body []byte) error {
	var e openapi.Error
	if json.Unmarshal(body, &e) == nil && e.Code != "" {
		return &apiRefusal{status: status, code: e.Code, msg: e.Error}
	}
	return &apiRefusal{status: status, msg: strings.TrimSpace(string(body))}
}

// encodeContribution writes claims as ranke's contribution stream, each record naming
// the branch its claim joins.
func encodeContribution(branch string, claims []ranke.Claim) ([]byte, error) {
	var buf bytes.Buffer
	w := ranke.NewWireWriter(&buf)
	for _, claim := range claims {
		if err := w.WriteClaim(branch, claim); err != nil {
			return nil, fmt.Errorf("encode claim %s: %w", claim.ID(), err)
		}
	}
	return buf.Bytes(), nil
}
