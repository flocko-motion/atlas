// Auth port conformance: every backend, built through auth.New rather than
// constructed directly, held to the contract auth.go's own doc comment describes.
package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/rankegraph/ranke-db/adapters/auth/apikey/apikeytest"
	"github.com/rankegraph/ranke-db/adapters/auth/jwt/jwttest"
	"github.com/rankegraph/ranke-db/adapters/auth/macaroon/macaroontest"
	"github.com/rankegraph/ranke-db/adapters/auth/noauth/noauthtest"
	"github.com/rankegraph/ranke-db/config/scope"
)

// conformanceAccount is the one identity every backend's setup provisions its own
// credential around, so the shared suite exercises one identity regardless of
// which backend built it.
const conformanceAccount = "conformance-account"

// backend is one auth backend under conformance. setup builds a config that
// resolves account, and returns the credential to present it under — a bare
// string for noauth/apikey, a signed JWT for jwt, whatever form this backend's
// own protocol actually uses — plus a teardown. Each backend's setup lives
// beside it, out of this port test and out of the production binary. validates
// is false only for a backend that accepts any credential by design (NoAuth) —
// "a bad token is rejected" is not a case every backend can be held to.
type backend struct {
	name       string
	setup      func(t *testing.T, account string) (cfg scope.Section, token string, teardown func())
	validates  bool
	wantScheme string
}

var backends = []backend{
	{name: "noauth", setup: noauthtest.Setup, validates: false, wantScheme: SchemeNone},
	{name: "apikey", setup: apikeytest.Setup, validates: true, wantScheme: SchemeAPIKey},
	{name: "jwt", setup: jwttest.Setup, validates: true, wantScheme: SchemeBearer},
	{name: "jwt-jwks", setup: jwttest.SetupJWKS, validates: true, wantScheme: SchemeBearer},
	{name: "macaroon", setup: macaroontest.Setup, validates: true, wantScheme: SchemeMacaroon},
}

// TestConformance runs every backend through its setup, auth.New — the factory a
// backend must be reachable through, which is itself part of the contract — and
// the shared suite. A backend that owns a background resource (jwt-jwks's refresh
// loop) is closed afterward via an optional Close(), not part of the Auth port
// itself — the port has no shutdown concept, only this test needs one.
func TestConformance(t *testing.T) {
	for _, b := range backends {
		t.Run(b.name, func(t *testing.T) {
			cfg, token, teardown := b.setup(t, conformanceAccount)
			defer teardown()
			a, err := New(context.Background(), cfg)
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			if closer, ok := a.(interface{ Close() }); ok {
				defer closer.Close()
			}
			conform(t, a, token, conformanceAccount, b.validates, b.wantScheme)
		})
	}
}

// conform is the shared auth contract.
func conform(t *testing.T, a Auth, token, account string, validates bool, wantScheme string) {
	t.Helper()
	ctx := context.Background()

	p, err := a.Authenticate(ctx, token)
	if err != nil {
		t.Fatalf("Authenticate(valid): %v", err)
	}
	if p.Account == "" {
		t.Fatal("Authenticate(valid) returned an empty Account")
	}
	if p.Account != account {
		t.Fatalf("Authenticate(valid).Account = %q, want %q", p.Account, account)
	}
	t.Logf("▸ valid credential authenticates as %q", p.Account)

	if validates {
		if _, err := a.Authenticate(ctx, "garbage-credential-matching-nothing"); !errors.Is(err, ErrUnauthenticated) {
			t.Fatalf("Authenticate(garbage) = %v, want ErrUnauthenticated", err)
		}
		t.Log("▸ garbage credential is rejected with ErrUnauthenticated")
	} else {
		t.Log("▸ backend accepts any credential by design — no rejection to check")
	}

	scheme := a.Scheme()
	if scheme != wantScheme {
		t.Fatalf("Scheme() = %q, want %q", scheme, wantScheme)
	}
	if got := a.Scheme(); got != scheme {
		t.Fatalf("Scheme() is not stable across calls: %q then %q", scheme, got)
	}
	t.Logf("▸ Scheme() = %q, stable across calls", scheme)
}

// TestSetRoutesByScheme pins Set's own dispatch, over backends built the same way
// TestConformance builds them: a credential routes to the backend whose Scheme
// matches, and the zero-value credential falls back to NoAuth. The third behaviour
// auth.go's header names — more than one credential is ambiguous — is not this
// port's surface to test: Set.Authenticate takes one Credential, and
// ErrAmbiguousCredentials is raised while an endpoint extracts several from a real
// request (adapters/endpoints/rest_http), before a Set ever sees them.
func TestSetRoutesByScheme(t *testing.T) {
	ctx := context.Background()

	noAuthCfg, _, noAuthDown := noauthtest.Setup(t, "anon")
	defer noAuthDown()
	noAuthBackend, err := New(ctx, noAuthCfg)
	if err != nil {
		t.Fatalf("New(noauth): %v", err)
	}

	apiKeyCfg, apiKeyToken, apiKeyDown := apikeytest.Setup(t, conformanceAccount)
	defer apiKeyDown()
	apiKeyBackend, err := New(ctx, apiKeyCfg)
	if err != nil {
		t.Fatalf("New(apikey): %v", err)
	}

	set, err := NewSet([]Auth{noAuthBackend, apiKeyBackend})
	if err != nil {
		t.Fatalf("NewSet: %v", err)
	}

	if p, err := set.Authenticate(ctx, Credential{Scheme: SchemeAPIKey, Token: apiKeyToken}); err != nil || p.Account != conformanceAccount {
		t.Fatalf("routed apikey credential: account=%q err=%v, want %q/nil", p.Account, err, conformanceAccount)
	}
	t.Log("▸ a credential routes to the backend whose Scheme matches")

	if p, err := set.Authenticate(ctx, Credential{}); err != nil || p.Account != "anon" {
		t.Fatalf("zero-value credential: account=%q err=%v, want anon/nil", p.Account, err)
	}
	t.Log("▸ the zero-value credential falls back to NoAuth")

	if _, err := set.Authenticate(ctx, Credential{Scheme: SchemeMacaroon, Token: "x"}); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("scheme with no backend configured: err=%v, want ErrUnauthenticated", err)
	}
	t.Log("▸ a scheme with no backend configured is unauthenticated, not a panic")
}
