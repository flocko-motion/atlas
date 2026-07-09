// package: endpoints / transport
// type:    factory
// job:     build one endpoint from its config section, wired to the core API and the authenticators
// limits:  dispatch only; the transport lives in the backend (-> adapters/endpoints/rest_http)
//
// This file is the endpoint port's composition seam. In the bootstrap order an
// endpoint is built last: it is handed the core API it drives (coreapi.API) and
// the full set of authenticators it may accept (a request may carry a JWT here, a
// macaroon there), then dispatched to the transport backend named in its section.
package endpoints

import (
	"context"
	"fmt"

	"github.com/flocko-motion/rankedb/adapters/auth"
	"github.com/flocko-motion/rankedb/adapters/endpoints/coreapi"
	"github.com/flocko-motion/rankedb/adapters/endpoints/rest_http"
	"github.com/flocko-motion/rankedb/config/scope"
)

// New builds the endpoint of the transport named by the section's "type" value,
// handing the backend the core API it drives and the authenticators it may
// accept. It returns an error for an empty or unknown type. Backends: rest
// (REST/HTTP); mcp (MCP/HTTP) lands here as it is built.
func New(ctx context.Context, cfg scope.Section, core coreapi.API, auths []auth.Auth) (Endpoints, error) {
	var t string
	if cfg.HasValue("type") {
		var err error
		if t, err = cfg.GetValue("type").Get(ctx); err != nil {
			return nil, fmt.Errorf("endpoints: type: %w", err)
		}
	}
	switch t {
	case "rest", "rest_http", "http":
		return rest_http.New(ctx, cfg, core, auths)
	case "":
		return nil, fmt.Errorf("endpoints: no transport type configured")
	default:
		return nil, fmt.Errorf("endpoints: unknown transport type %q", t)
	}
}
