// package: endpoints / transport
// type:    interface + factory
// job:     the Endpoint port — bind a transport to an authenticator and carry the read/contribute surface into core — plus the factory
// limits:  contract + dispatch; transports live in sub-packages (-> adapters/endpoints/rest_http, mcp_http)
//
// Package endpoints defines the driving Endpoint port and builds it from config.
// An endpoint pairs a transport (REST/HTTP, MCP/HTTP) with an authenticator
// (-> adapters/auth): it receives client requests in its own wire format,
// extracts the credential and resolves the subject, hands the call to core, and
// renders the result back. The server is event-driven — an endpoint serves
// requests as they arrive — so several endpoints may run at once on different
// ports or protocols.
package endpoints

import (
	"context"
	"fmt"

	"github.com/flocko-motion/rankedb/adapters/auth"
	"github.com/flocko-motion/rankedb/adapters/sequencer"
	"github.com/flocko-motion/rankedb/config/scope"
)

// Endpoints is one configured transport+authenticator listener carrying the full
// read-and-contribute surface. Backends: REST/HTTP (OpenAPI), MCP/HTTP
// (-> sub-packages).
type Endpoints interface {
	// Serve runs the endpoint, handling requests until ctx is cancelled, then
	// shuts down.
	Serve(ctx context.Context) error

	// Close releases the endpoint's transport resources.
	Close() error
}

// New builds an endpoint of the transport named in cfg, wired to the sequencer it
// drives and the full set of authenticators it may accept (a request may carry a
// JWT here, a macaroon there). Stub: not yet implemented.
func New(ctx context.Context, cfg scope.Section, seq sequencer.Sequencer, auths []auth.Auth) (Endpoints, error) {
	return nil, fmt.Errorf("endpoints: transport not yet implemented")
}
