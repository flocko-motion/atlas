package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	schemafapi "github.com/flocko-motion/schemaf/api"

	"rankedb/access"
	"rankedb/adapter/config"
	configmem "rankedb/adapter/config/mem"
	grantsmem "rankedb/adapter/grants/mem"
	"rankedb/assembler"
	"rankedb/core"
)

// TestServeEndToEnd drives a real HTTP request through schemaf's mux (auth
// middleware + registered routes) into core — the positive path that the
// ctx-subject-private unit tests can't reach.
func TestServeEndToEnd(t *testing.T) {
	schemafapi.InitAuth([]byte("test-signing-key"))
	schemafapi.Reset()
	Provider()

	entries := config.Entries{
		"tenants.acme.archives.main.title":             "Main",
		"tenants.acme.archives.main.state":             "running",
		"tenants.acme.archives.main.storage.backend":   "mem",
		"tenants.acme.archives.main.sequencer.backend": "mem",
	}
	c := core.New(access.New([]string{"root"}, grantsmem.New()), configmem.NewFrom(entries), assembler.Deps{})
	if err := c.Reconcile(context.Background()); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	Use(c)

	srv := httptest.NewServer(schemafapi.NewMux())
	defer srv.Close()

	get := func(t *testing.T, subject string) *http.Response {
		t.Helper()
		req, _ := http.NewRequest("GET", srv.URL+"/api/archives/acme/main", nil)
		if subject != "" {
			tok, err := schemafapi.IssueToken(subject, time.Now().Add(time.Hour))
			if err != nil {
				t.Fatalf("IssueToken: %v", err)
			}
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		return resp
	}

	// root sees the archive → 200 with status.
	resp := get(t, "root")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("root GET = %d, want 200", resp.StatusCode)
	}
	var body GetArchiveResp
	_ = json.NewDecoder(resp.Body).Decode(&body)
	resp.Body.Close()
	if body.Title != "Main" || body.Current != "running" {
		t.Fatalf("body = %+v, want Main/running", body)
	}

	// No token → 401 (schemaf auth middleware, before the handler).
	resp = get(t, "")
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no-token GET = %d, want 401", resp.StatusCode)
	}

	// A subject with no visibility into the tenant → 404 (existence hidden).
	resp = get(t, "stranger")
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("stranger GET = %d, want 404 (hidden)", resp.StatusCode)
	}
}
