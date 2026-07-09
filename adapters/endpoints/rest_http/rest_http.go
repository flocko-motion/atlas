// package: rest_http / transport
// type:    adapter
// job:     REST/HTTP endpoint backend (OpenAPI) — implement endpoints.Endpoints, own the HTTP server lifecycle
// limits:  transport + translation only; all capability lives behind coreapi.API (-> adapters/endpoints/coreapi)
//
// Package rest_http serves the ranke-db REST API over HTTP. It implements the
// generated api.ServerInterface (from openapi/openapi.yaml) by, for each request,
// resolving the caller's Subject through the auth port, translating the wire
// request into a coreapi.API call, and rendering the domain result — or the
// mapped status for a sentinel error — back onto the response. It holds a
// coreapi.API and drives it; it never reaches past that interface.
//
// The package is split by topic:
//
//	rest_http.go              the Server and its http.Server lifecycle (this file)
//	auth.go                   credential → Subject, stashed in the request context
//	respond.go                response writers and the sentinel-error → status map
//	endpoints_read.go         query and the cacheable by-id reads
//	endpoints_contribute.go   contribute
//	endpoints_system.go       health and storage-layer introspection
//	endpoints_verification.go the verification run API
package rest_http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/flocko-motion/rankedb/adapters/auth"
	"github.com/flocko-motion/rankedb/adapters/endpoints/coreapi"
	"github.com/flocko-motion/rankedb/api"
	"github.com/flocko-motion/rankedb/config/scope"
)

// Server is a running REST/HTTP endpoint: an http.Server whose handlers call into
// the core through coreapi.API.
type Server struct {
	core  coreapi.API
	auths []auth.Auth
	srv   *http.Server
}

// New builds the REST/HTTP endpoint from its config section, wired to the core it
// drives (coreapi.API) and the authenticators it may accept. The section's "addr"
// sets the listen address (default ":8080").
//
// TODO(endpoint-boundary): BROKEN signature — this takes (coreapi.API, []auth.Auth)
// from the contributor's auth-in-endpoint model. On this branch the endpoint is
// pure transport and auth/access live inside core, so the factory hands one
// *core.Core (see adapters/endpoints/endpoints.go: New(ctx, cfg, *core.Core) and
// config.buildEndpoint). This constructor is not dispatched to yet — endpoints.New
// still returns the not-implemented stub — and must be rewired to core.Core, with
// the []auth.Auth parameter and the auths field dropped once auth moves fully into
// core.
func New(ctx context.Context, cfg scope.Section, core coreapi.API, auths []auth.Auth) (*Server, error) {
	addr := ":8080"
	if cfg.HasValue("addr") {
		a, err := cfg.Get(ctx, "addr")
		if err != nil {
			return nil, fmt.Errorf("rest_http: addr: %w", err)
		}
		if a != "" {
			addr = a
		}
	}
	s := &Server{core: core, auths: auths}
	handler := s.withAuth(api.Handler(s))
	s.srv = &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	return s, nil
}

// Serve runs the endpoint until ctx is cancelled, then shuts down gracefully.
func (s *Server) Serve(ctx context.Context) error {
	errc := make(chan error, 1)
	go func() { errc <- s.srv.ListenAndServe() }()
	select {
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		return s.shutdown()
	}
}

// Close shuts the endpoint down.
func (s *Server) Close() error { return s.shutdown() }

func (s *Server) shutdown() error {
	sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.srv.Shutdown(sctx)
}
