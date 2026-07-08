// package: auth / authn
// type:    factory
// job:     build the configured auth backend from the narrow auth.Config view
// limits:  dispatch only; credential checking lives in the backend (-> adapters/auth/noauth)
//
// This file is the auth port's composition seam. It takes only the auth
// instance's scope — a resolved key/value object holding no field outside the
// auth section — and dispatches to the chosen backend, passing the scope on.
package auth

import (
	"fmt"

	"github.com/flocko-motion/rankedb/adapters/auth/noauth"
	"github.com/flocko-motion/rankedb/config/scope"
)

// New builds the auth backend named by the scope's "type" value, handing the
// backend the same scope to read its secrets from. An empty type defaults to
// noauth. The credential-checking backends (jwt, apikey) land here as they are
// added.
func New(cfg scope.Config) (Auth, error) {
	switch t := cfg.String("type"); t {
	case "noauth", "":
		return noauth.New(cfg), nil
	default:
		return nil, fmt.Errorf("auth: unknown backend type %q", t)
	}
}
