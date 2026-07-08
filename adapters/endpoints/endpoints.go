// package: endpoints / transport
// type:    interface
// job:     the Endpoint port — bind a transport to an authenticator and carry the read/contribute surface into core
// limits:  contract only; backends live in sub-packages (-> adapters/endpoints/rest_http, mcp_http)
//
// Package endpoints defines the driving Endpoint port. An endpoint pairs a
// transport (REST/HTTP, MCP/HTTP) with an authenticator (-> adapters/auth): it
// receives client requests in its own wire format, extracts the credential and
// resolves the subject, hands the call to core, and renders the result back in
// that format. The server is event-driven — an endpoint serves requests as they
// arrive — so several endpoints may run at once on different ports or protocols.
package endpoints

import "context"

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
