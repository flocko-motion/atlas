// package: rest_http / transport
// type:    adapter
// job:     REST/HTTP endpoint backend — the Endpoints port and its server lifecycle
// limits:  transport + translation only; all capability lives behind core.Core (-> internal/core)
//
// Package rest_http serves the ranke-db REST API over HTTP. It implements the
// generated openapi.ServerInterface (from openapi/openapi.yaml) by, for each request,
// extracting the caller's raw credential from the wire, translating the request
// into a core.Request, handing it to core.Handle — which authenticates, authorizes
// and executes — and rendering the response, or the mapped status for a sentinel
// error, back onto the wire. Auth and access live in core; this package is pure
// transport and never resolves a subject itself.
//
// The package is split by topic:
//
//	rest_http.go              the Server and its http.Server lifecycle (this file)
//	auth.go                   extract the wire credential, carried to the handlers
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

	"github.com/flocko-motion/rankedb/config/scope"
	"github.com/flocko-motion/rankedb/internal/core"
	"github.com/flocko-motion/rankedb/openapi"
)

// Server is a running REST/HTTP endpoint: an http.Server whose handlers translate
// each request into a core.Request and drive it through core.Handle.
type Server struct {
	core *core.Core
	srv  *http.Server
}

// New builds the REST/HTTP endpoint from its config section, wired to the core it
// drives. The section's "addr" sets the listen address (default ":8080").
func New(ctx context.Context, cfg scope.Section, c *core.Core) (*Server, error) {
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
	s := &Server{core: c}
	s.srv = &http.Server{Addr: addr, Handler: s.withCredential(openapi.Handler(s)), ReadHeaderTimeout: 5 * time.Second}
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
