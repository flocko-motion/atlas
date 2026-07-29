package rest_http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flocko-motion/rankedb/adapters/auth"
	"github.com/flocko-motion/rankedb/config/scope"
	"github.com/flocko-motion/rankedb/internal/core"
	"github.com/flocko-motion/rankedb/internal/core/access"
)

// newTestServer builds the endpoint over a core whose NoAuth backend authenticates
// every request as "ops", granted R on every branch and on the reserved scopes. The
// driven ports are nil: execution is a stub, so a routed request reaches it and
// answers 501 — which is exactly what distinguishes "routed" from "not routed".
func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	ctx := context.Background()

	a, err := auth.New(ctx, scope.Literal(map[string]string{"type": "noauth", "subject": "ops"}))
	if err != nil {
		t.Fatalf("auth.New: %v", err)
	}
	set, err := auth.NewSet([]auth.Auth{a})
	if err != nil {
		t.Fatalf("auth.NewSet: %v", err)
	}
	chk, err := access.New(map[string][]string{"ops": {"CR *", "R $universe", "R $archive"}})
	if err != nil {
		t.Fatalf("access.New: %v", err)
	}
	s, err := New(ctx, scope.Literal(map[string]string{"addr": ":0"}), core.New(set, chk, nil, nil))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s.srv.Handler
}

// TestRoutes pins that every route the contract declares is actually reachable —
// the one thing a generated router can silently get wrong, and which no unit test on
// the mapping would catch. A routed request lands on the execute stub (501); an
// unrouted one is the mux's own 404, so the two are told apart by status.
func TestRoutes(t *testing.T) {
	h := newTestServer(t)
	const id = testClaimID

	for _, tc := range []struct {
		method, path, body string
	}{
		{http.MethodGet, "/health", ""},
		{http.MethodPost, "/query", `{"select": {"branch": "foo"}}`},
		{http.MethodPost, "/contribute?branch=foo", ""},
		{http.MethodGet, "/foo/head", ""},
		{http.MethodGet, "/foo/claim/" + id, ""},
		{http.MethodGet, "/foo/claim/" + id + "/content", ""},
		{http.MethodGet, "/$universe/claim/" + id, ""},
		{http.MethodGet, "/$universe/claim/" + id + "/content", ""},
		{http.MethodGet, "/system/layers", ""},
		{http.MethodGet, "/system/verification", ""},
		{http.MethodPost, "/system/verification", `{"closure": "foo"}`},
		{http.MethodGet, "/system/verification/r1", ""},
		{http.MethodDelete, "/system/verification/r1", ""},
		{http.MethodPost, "/system/verification/r1/cancel", ""},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.body != "" {
				r.Header.Set("Content-Type", "application/json")
			}
			h.ServeHTTP(rec, r)
			if rec.Code == http.StatusNotFound {
				t.Fatalf("status = 404, want the route to be reachable: %s", rec.Body.String())
			}
		})
	}
}

// TestQueryRejectsAtTheBoundary pins the two RankeQL rules the JSON schema cannot
// state. They are enforced before the request reaches core, so each is a 400 rather
// than the 501 a routed-but-unimplemented read returns.
func TestQueryRejectsAtTheBoundary(t *testing.T) {
	h := newTestServer(t)

	for _, tc := range []struct{ name, body string }{
		{"no scope", `{"select": {}}`},
		{"universe without a head", `{"select": {"branch": "$universe"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/query", strings.NewReader(tc.body))
			r.Header.Set("Content-Type", "application/json")
			h.ServeHTTP(rec, r)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

// TestNoCypherRoute pins the contract's refusal to carry a client Cypher string: the
// route the pre-hexagonal API exposed must not resolve.
func TestNoCypherRoute(t *testing.T) {
	h := newTestServer(t)
	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/branches/foo/gql", strings.NewReader(`{"query": "MATCH (n) RETURN n"}`))
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 — there is no client Cypher endpoint", rec.Code)
	}
}
