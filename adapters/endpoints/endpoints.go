// package: endpoints / transport
// type:    interface + factory
// job:     the Endpoint port — bind a transport to core.Handle — plus the factory
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

	"github.com/rankegraph/ranke-db/adapters/endpoints/rest_http"
	"github.com/rankegraph/ranke-db/config/scope"
	"github.com/rankegraph/ranke-db/internal/core"
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

// New builds an endpoint of the transport named by the section's "type", wired to
// the core it drives. REST/HTTP is implemented; MCP/HTTP is pending.
func New(ctx context.Context, cfg scope.Section, c *core.Core) (Endpoints, error) {
	var t string
	if cfg.HasValue("type") {
		var err error
		if t, err = cfg.Get(ctx, "type"); err != nil {
			return nil, fmt.Errorf("endpoints: type: %w", err)
		}
	}
	switch t {
	case "rest", "rest_http":
		return rest_http.New(ctx, cfg, c)
	case "mcp", "mcp_http":
		return nil, fmt.Errorf("endpoints: mcp transport not yet implemented")
	case "":
		return nil, fmt.Errorf("endpoints: no transport type configured")
	default:
		return nil, fmt.Errorf("endpoints: unknown transport %q", t)
	}
}
