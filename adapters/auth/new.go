// package: auth / authn
// type:    factory
// job:     build the configured auth backend from the narrow auth section
// limits:  dispatch only; credential checking lives in the backend (-> adapters/auth/noauth)
//
// This file is the auth port's composition seam. It takes only the auth
// instance's section — a navigable view holding no field outside the auth
// section — and dispatches to the chosen backend, passing the section on.
package auth

import (
	"context"
	"fmt"

	"github.com/flocko-motion/rankedb/adapters/auth/noauth"
	"github.com/flocko-motion/rankedb/config/scope"
)

// New builds the auth backend named by the section's "type" value, handing the
// backend the same section to read its secrets from. An empty type defaults to
// noauth. The credential-checking backends (jwt, apikey) land here as they are
// added.
func New(ctx context.Context, cfg scope.Section) (Auth, error) {
	var t string
	if cfg.HasValue("type") {
		var err error
		if t, err = cfg.GetValue("type").Get(ctx); err != nil {
			return nil, fmt.Errorf("auth: type: %w", err)
		}
	}
	switch t {
	case "noauth", "":
		return noauth.New(ctx, cfg)
	default:
		return nil, fmt.Errorf("auth: unknown backend type %q", t)
	}
}
