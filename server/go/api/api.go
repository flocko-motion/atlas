// package: api / rest
// type:    adapter
// job:     schemaf REST endpoints over core — decode, gate, delegate to core, encode; map core/access errors to HTTP status
// limits:  thin adapters only (all logic is in core); endpoints are registered via schemaf codegen (the generated Provider), wired in main
//
// We manage capabilities, not identity: the JWT subject (schemaf) is who you
// are; these endpoints expose what a subject may do/see over core. Endpoints
// are zero-value structs (schemaf instantiates them), so the core they serve is
// injected package-level via Use, the framework's convention.
package api

import (
	"errors"
	"fmt"

	schemafapi "github.com/flocko-motion/schemaf/api"

	"rankedb/access"
	"rankedb/core"
)

// svc is the core the endpoints delegate to, injected at startup via Use.
var svc *core.Core

// Use wires the core the endpoints serve. Call once at startup before serving.
func Use(c *core.Core) { svc = c }

// mapErr translates core/access errors into schemaf's status sentinels
// (schemaf maps these to HTTP status; anything else → 500):
//   - *access.Denied            → 403, carrying the subject id (onboarding)
//   - core.ErrNotFound          → 404 (also covers not-visible — existence hiding)
//   - core.ErrReadOnly          → 409 (write refused: archive is read-only)
//   - core.ErrUnavailable       → 503 (archive not serving)
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, core.ErrNotFound):
		return schemafapi.ErrNotFound
	case errors.Is(err, core.ErrReadOnly):
		return schemafapi.ErrConflict
	case errors.Is(err, core.ErrUnavailable):
		return schemafapi.ErrUnavailable
	}
	var denied *access.Denied
	if errors.As(err, &denied) {
		return fmt.Errorf("denied for subject %q: %w", denied.Subject, schemafapi.ErrForbidden)
	}
	return err
}
