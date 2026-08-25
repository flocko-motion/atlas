package rest_http

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ranke "github.com/flocko-motion/ranke-go"

	"github.com/flocko-motion/rankedb/adapters/auth"
	"github.com/flocko-motion/rankedb/adapters/sequencer"
	"github.com/flocko-motion/rankedb/adapters/signer"
	"github.com/flocko-motion/rankedb/adapters/storage"
	"github.com/flocko-motion/rankedb/config/scope"
	"github.com/flocko-motion/rankedb/internal/core"
	"github.com/flocko-motion/rankedb/internal/core/access"
)

// newTestServer builds the endpoint over a core with no driven ports: every request
// authenticates as "ops", and a routed one reaches execution and finds no archive (501).
// That 501 is what tells "routed" from "not routed" — a 404 is the mux's own.
func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	return newServerFor(t, nil, nil)
}

// everyReadRight is what the test subject holds unless a case narrows it: each reserved
// scope has to be granted by name, so the default names them all.
var everyReadRight = []string{"CR *", "C $branches", "R $universe", "R $archive", "R $branches"}

// newServerFor builds the endpoint over a core bound to the given driven ports.
func newServerFor(t *testing.T, seq sequencer.Sequencer, store storage.Storage) http.Handler {
	t.Helper()
	return newGrantedServer(t, everyReadRight, seq, store)
}

// newGrantedServer builds the endpoint over a core admitting one account with exactly the
// grants given, so a case can withhold one right and see what stops working.
func newGrantedServer(t *testing.T, grants []string, seq sequencer.Sequencer, store storage.Storage) http.Handler {
	t.Helper()
	return newServerWith(t, grants, map[string]string{"addr": ":0"}, seq, store)
}

// newServerWith builds the endpoint from a transport config of the case's choosing, so a
// case can declare things like allowedOrigins without a second copy of the wiring.
func newServerWith(
	t *testing.T,
	grants []string,
	transport map[string]string,
	seq sequencer.Sequencer,
	store storage.Storage,
) http.Handler {
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
	chk, err := access.New(map[string][]string{"ops": grants})
	if err != nil {
		t.Fatalf("access.New: %v", err)
	}
	s, err := New(ctx, scope.Literal(transport), core.New(set, chk, seq, store))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s.srv.Handler
}

// newServingServer builds the endpoint over a real stack — an in-memory universe, a dev
// sequencer, a real signer — for the routes whose behaviour needs an archive to exist.
func newServingServer(t *testing.T) http.Handler {
	t.Helper()
	h, _ := newServingStack(t, everyReadRight)
	return h
}

// newServingStack builds a serving endpoint and hands back the Universe behind it, so a
// test can put a claim where only the unconfined scope will find it.
func newServingStack(t *testing.T, grants []string) (http.Handler, ranke.Universe) {
	t.Helper()
	ctx := context.Background()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	key := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	sig, err := signer.New(ctx, scope.Literal(map[string]string{"type": "inmemory", "key": string(key)}))
	if err != nil {
		t.Fatalf("signer.New: %v", err)
	}
	store := ranke.NewMemoryUniverse()
	seq, err := sequencer.New(ctx, scope.Literal(map[string]string{"type": "dev"}), store, sig, nil)
	if err != nil {
		t.Fatalf("sequencer.New: %v", err)
	}
	return newGrantedServer(t, grants, seq, store), store
}

// TestRoutes pins that every route the contract declares is actually reachable —
// the one thing a generated router can silently get wrong, and which no unit test on
// the mapping would catch. A routed request lands on the execute stub (501); an
// unrouted one is the mux's own 404, so the two are told apart by status.
func TestRoutes(t *testing.T) {
	h := newTestServer(t)
	id := testClaimID

	for _, tc := range []struct {
		method, path, body string
	}{
		{http.MethodGet, "/health", ""},
		{http.MethodPost, "/query", `{"select": {"branch": "foo"}}`},
		{http.MethodPost, "/contribute?branch=foo", ""},
		{http.MethodGet, "/branches", ""},
		{http.MethodGet, "/branches/foo/head", ""},
		{http.MethodGet, "/branches/foo/info", ""},
		{http.MethodGet, "/archive/info", ""},
		{http.MethodGet, "/branches/foo/claims/" + id, ""},
		{http.MethodGet, "/branches/foo/claims/" + id + "/content", ""},
		{http.MethodGet, "/archive/claims/" + id, ""},
		{http.MethodGet, "/archive/claims/" + id + "/content", ""},
		{http.MethodGet, "/universe/claims/" + id, ""},
		{http.MethodGet, "/universe/claims/" + id + "/content", ""},
		{http.MethodGet, "/system/layers", ""},
		{http.MethodGet, "/system/verifications", ""},
		{http.MethodPost, "/system/verifications", `{"closure": "foo"}`},
		{http.MethodGet, "/system/verifications/r1", ""},
		{http.MethodDelete, "/system/verifications/r1", ""},
		{http.MethodPost, "/system/verifications/r1/cancel", ""},
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

// TestVerificationHeaders pins the headers the contract promises on the run API: a 202
// names where to poll and how soon, and a running report repeats the hint. Without these
// a client has to guess a URL it was told it would be given.
func TestVerificationHeaders(t *testing.T) {
	h := newServingServer(t)

	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/system/verifications", strings.NewReader(`{"closure":"$archive"}`))
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202: %s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "/system/verifications/") || loc == "/system/verifications/" {
		t.Fatalf("Location = %q, want the report's own route", loc)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("no Retry-After on 202 — a client is told to poll but not how soon")
	}

	// The Location must actually resolve.
	get := httptest.NewRecorder()
	h.ServeHTTP(get, httptest.NewRequest(http.MethodGet, loc, nil))
	if get.Code != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200: %s", loc, get.Code, get.Body.String())
	}
}

// TestBusyCarriesRetryAfter pins that a refusal for want of a free slot says when to come
// back. A 429 without it tells a client to retry and leaves the interval to guesswork.
func TestBusyCarriesRetryAfter(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, core.CatBusy, "too many runs")

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("no Retry-After on 429")
	}

	// Other categories carry none: there is nothing to wait for.
	other := httptest.NewRecorder()
	writeError(other, core.CatNotFound, "nope")
	if got := other.Header().Get("Retry-After"); got != "" {
		t.Fatalf("Retry-After = %q on 404, want none", got)
	}
}
