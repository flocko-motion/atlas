// package: rest_http / transport
// type:    adapter
// job:     REST/HTTP endpoint backend — the Endpoints port and its server lifecycle
// limits:  transport + translation only; all capability lives behind core.Core (-> internal/core)
//
// Package rest_http implements the generated openapi.ServerInterface: it lifts the raw
// credential off the wire, builds a core.Request, and renders what core.Handle answers.
// Auth and access live in core; this is pure transport.
//
// By topic: auth.go (wire credential), respond.go (writers, error → status),
// endpoints_read.go, endpoints_contribute.go, endpoints_system.go,
// endpoints_verification.go — the Server's own lifecycle here.
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

// Server is a running REST/HTTP endpoint driving core.Handle.
type Server struct {
	core *core.Core
	srv  *http.Server
}

// New builds the endpoint from its config section; "addr" listens (default ":8080").
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
	cors, err := corsFrom(ctx, cfg)
	if err != nil {
		return nil, err
	}
	// "explorer" opts out of a build's embedded UI (default: served) — ignored, either
	// way, on a binary built without -tags explorer, which has nothing to serve regardless.
	explorer := true
	if cfg.HasValue("explorer") {
		v, err := cfg.Get(ctx, "explorer")
		if err != nil {
			return nil, fmt.Errorf("rest_http: explorer: %w", err)
		}
		explorer = v != "false"
	}

	s := &Server{core: c}
	// /explorer sits outside the generated router: it's a static asset, not part of the
	// OpenAPI contract, and the generated handler owns "/" as its own catch-all.
	mux := http.NewServeMux()
	mux.Handle("/explorer", explorerHandler(explorer))
	mux.Handle("/", openapi.Handler(s))
	// CORS outermost: a preflight carries no credential and must be answered before the
	// credential is extracted, since a browser sends it without one.
	handler := cors.withCORS(s.withCredential(mux))
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
