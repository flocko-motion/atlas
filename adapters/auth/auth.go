// package: auth / authn
// type:    interface + factory + dispatcher
// job:     the Auth port — turn a request credential into an authenticated Principal — the factory, and the scheme dispatcher
// limits:  identity only; WHAT a principal may do is access's (-> internal/core/access); credential checking lives in the backends
//
// Package auth defines the authentication port (credential in, Principal out) and
// builds the configured backends. It settles only WHO the caller is — a system
// account, plus any caveats a token carried; WHAT they may do is the access
// checker's, keyed on the returned Principal one step later in core. An endpoint
// extracts the raw credential per its transport (a Bearer token, an X-API-Key
// header, …) and hands it here, so the port stays transport-neutral.
//
// Each backend answers one credential scheme. A Set groups the configured backends
// and dispatches by the scheme an endpoint presents: exactly one credential routes
// to its backend, none falls back to NoAuth, and more than one is ambiguous and
// rejected — the endpoint never guesses a token's kind by inspecting its bytes.
package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/flocko-motion/rankedb/adapters/auth/apikey"
	"github.com/flocko-motion/rankedb/adapters/auth/noauth"
	"github.com/flocko-motion/rankedb/config/scope"
	"github.com/flocko-motion/rankedb/internal/core/access"
)

// ErrUnauthenticated reports that a credential was required but missing or
// invalid — an endpoint maps it to 401.
var ErrUnauthenticated = errors.New("auth: unauthenticated")

// ErrAmbiguousCredentials reports more than one auth scheme on one request. An
// endpoint raises it while extracting, before core runs, and maps it to 400.
var ErrAmbiguousCredentials = errors.New("auth: ambiguous credentials")

// Well-known credential schemes: a backend reports its own via Scheme(), an endpoint
// tags each credential with the same string, and the empty scheme is NoAuth.
const (
	SchemeNone     = ""         // NoAuth: no credential presented
	SchemeAPIKey   = "apikey"   // an API key (e.g. X-API-Key)
	SchemeBearer   = "bearer"   // a JWT bearer token
	SchemeMacaroon = "macaroon" // a macaroon
)

// Credential is one authentication token an endpoint extracted from a request,
// tagged with the scheme it was presented under.
type Credential struct {
	Scheme string
	Token  string
}

// Auth authenticates one credential scheme. Backends: NoAuth, JWT, API key,
// Macaroon (-> sub-packages).
type Auth interface {
	// Authenticate resolves token to a Principal, or returns ErrUnauthenticated.
	// NoAuth ignores the token and returns its configured account.
	Authenticate(ctx context.Context, token string) (access.Principal, error)

	// Scheme reports which Scheme* constant this backend consumes.
	Scheme() string
}

// New builds the backend named by the section's "type", handing it that same section
// to read its secrets from. An empty type defaults to noauth.
func New(ctx context.Context, cfg scope.Section) (Auth, error) {
	var t string
	if cfg.HasValue("type") {
		var err error
		if t, err = cfg.Get(ctx, "type"); err != nil {
			return nil, fmt.Errorf("auth: type: %w", err)
		}
	}
	switch t {
	case "noauth", "":
		return noauth.New(ctx, cfg)
	case "apikey":
		return apikey.New(ctx, cfg)
	default:
		return nil, fmt.Errorf("auth: unknown backend type %q", t)
	}
}

// Set is the configured authenticators indexed by scheme, plus the optional NoAuth
// fallback. It is what an endpoint hands a request's credentials to.
type Set struct {
	byScheme map[string]Auth
	noAuth   Auth
}

// NewSet indexes the backends by scheme, rejecting two backends that claim the
// same scheme (an endpoint could not route between them).
func NewSet(auths []Auth) (*Set, error) {
	s := &Set{byScheme: make(map[string]Auth, len(auths))}
	for _, a := range auths {
		sc := a.Scheme()
		if sc == SchemeNone {
			if s.noAuth != nil {
				return nil, fmt.Errorf("auth: multiple NoAuth backends configured")
			}
			s.noAuth = a
			continue
		}
		if _, dup := s.byScheme[sc]; dup {
			return nil, fmt.Errorf("auth: multiple backends for scheme %q", sc)
		}
		s.byScheme[sc] = a
	}
	return s, nil
}

// Authenticate resolves the one credential an endpoint extracted: a zero value falls
// back to NoAuth, anything else routes by scheme, and neither found is
// ErrUnauthenticated. Multi-scheme requests die at extraction, not here.
func (s *Set) Authenticate(ctx context.Context, cred Credential) (access.Principal, error) {
	if cred.Scheme == SchemeNone {
		if s.noAuth == nil {
			return access.Principal{}, ErrUnauthenticated
		}
		return s.noAuth.Authenticate(ctx, "")
	}
	a, ok := s.byScheme[cred.Scheme]
	if !ok {
		return access.Principal{}, ErrUnauthenticated
	}
	return a.Authenticate(ctx, cred.Token)
}
