// package: endpoints / transport
// type:    interface + factory
// job:     the Endpoint port — bind a transport to an authenticator and carry the read/contribute surface into core — plus the factory
// limits:  contract + dispatch; transports live in sub-packages (-> adapters/endpoints/rest_http, mcp_http)
//
// Package endpoints defines the driving Endpoint port and builds it from config.
// An endpoint binds a transport (REST/HTTP, MCP/HTTP) to core: it receives client
// requests in its own wire format, builds a core.Request from them — extracting
// each auth credential its transport carries — hands it to core.Handle, and
// renders the enriched result back. Authentication and authorization happen inside
// core, keyed on the credentials the endpoint tagged; the endpoint stays pure
// transport. The server is event-driven — an endpoint serves requests as they
// arrive — so several endpoints may run at once on different ports or protocols.
package endpoints

import (
	"context"
	"fmt"

	"github.com/flocko-motion/rankedb/config/scope"
	"github.com/flocko-motion/rankedb/internal/core"
)

// Endpoints is one configured transport listener carrying the full read-and-
// contribute surface into core. Backends: REST/HTTP (OpenAPI), MCP/HTTP
// (-> sub-packages).
type Endpoints interface {
	// Serve runs the endpoint, handling requests until ctx is cancelled, then
	// shuts down.
	Serve(ctx context.Context) error

	// Close releases the endpoint's transport resources.
	Close() error
}

// New builds an endpoint of the transport named in cfg, wired to the core it
// drives. Stub: not yet implemented.
func New(ctx context.Context, cfg scope.Section, c *core.Core) (Endpoints, error) {
	return nil, fmt.Errorf("endpoints: transport not yet implemented")
}
