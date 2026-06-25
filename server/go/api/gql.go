// package: api / gql
// type:    adapter
// job:     Cypher/GQL query endpoint — read-gated; runs the query if a layer in the archive's stack speaks Cypher, else 501
// limits:  read-only (mutation is minting, never Cypher writes); the Cypher capability + no-backdoor filtering are ranke-go's. STUB: ranke-go has no query surface yet, so this always 501s after gating.
package api

import (
	"net/http"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// GqlEndpoint runs a read-only Cypher/GQL query against an archive — if a layer
// in its stack speaks Cypher. HandleRaw is used so the no-capability case can
// return a precise 501 (schemaf has no 501 sentinel). Stub: always 501 after
// the access/lifecycle gate, until ranke-go ships the capability.
type GqlEndpoint struct{}

// Method is POST (queries are parameterized; GraphQL-style).
func (GqlEndpoint) Method() string { return "POST" }

// Path is the archive's gql sub-resource.
func (GqlEndpoint) Path() string { return "/api/archives/{tenant}/{ra}/gql" }

// Auth requires a valid JWT.
func (GqlEndpoint) Auth() bool { return true }

// HandleRaw gates the request (ReadRA + serving) then reports that no
// Cypher-capable layer is available (501).
func (GqlEndpoint) HandleRaw(w http.ResponseWriter, r *http.Request) error {
	subject, _ := schemafapi.Subject(r.Context())
	if _, err := svc.Reader(r.Context(), subject, r.PathValue("tenant"), r.PathValue("ra")); err != nil {
		return mapErr(err) // 403 / 404 / 503 via the framework's HandleRaw error mapping
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	_, _ = w.Write([]byte(`{"error":"gql unavailable: no Cypher-capable layer in this archive's stack"}`))
	return nil
}
