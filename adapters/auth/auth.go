// package: auth / authn
// type:    interface
// job:     the Auth port — turn a request credential into an account subject (authentication; authorization is the access checker's job)
// limits:  contract only; backends live in sub-packages (-> adapters/auth/noauth, jwt, apikey)
//
// Package auth defines the authentication port: credential in, subject out. It
// settles only WHO the caller is; WHAT they may do is the access checker's,
// keyed on the returned subject. An endpoint adapter extracts the raw
// credential per its transport (a Bearer token, an X-API-Key header, …) and
// hands it here, so the port stays transport-neutral.
package auth

import (
	"context"
	"errors"
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
