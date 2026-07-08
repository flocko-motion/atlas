// package: auth / authn
// type:    interface + factory
// job:     the Auth port — turn a request credential into an account subject — plus the factory that builds a backend from config
// limits:  contract + dispatch; credential checking lives in the backends (-> adapters/auth/noauth, jwt, apikey)
//
// Package auth defines the authentication port (credential in, subject out) and
// builds the configured backend. It settles only WHO the caller is; WHAT they may
// do is the access checker's, keyed on the returned subject. An endpoint adapter
// extracts the raw credential per its transport (a Bearer token, an X-API-Key
// header, …) and hands it here, so the port stays transport-neutral.
package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/flocko-motion/rankedb/adapters/auth/noauth"
	"github.com/flocko-motion/rankedb/config/scope"
)

// ErrUnauthenticated reports that a credential was required but missing or
// invalid — an endpoint maps it to 401.
var ErrUnauthenticated = errors.New("auth: unauthenticated")

// Auth authenticates a request credential. Backends: NoAuth (one implicit
// account), JWT, API key (-> sub-packages).
type Auth interface {
	// Authenticate returns the subject (account id) for credential, or
	// ErrUnauthenticated if it is missing or invalid. NoAuth ignores the
	// credential and returns its configured default subject.
	Authenticate(ctx context.Context, credential string) (subject string, err error)
}

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
