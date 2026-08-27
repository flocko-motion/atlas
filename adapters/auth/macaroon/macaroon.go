// package: macaroon / authn
// type:    adapter
// job:     authenticate a macaroon, translating its caveats into attenuated grants
// limits:  verification only — this server never mints or attenuates one (-> auth.New)
//
// Tokens are untrusted external input; this backend verifies, never mints. Each
// first-party caveat is one "RIGHTS glob" grant and one attenuation step, ALL of
// which must hold; an unparseable or third-party caveat refuses the whole token.
package macaroon

import (
	"context"
	"errors"
	"fmt"

	joemacaroon "gopkg.in/macaroon.v2"

	"github.com/flocko-motion/rankedb/adapters/auth/autherr"
	"github.com/flocko-motion/rankedb/config/scope"
	"github.com/flocko-motion/rankedb/internal/core/access"
)

// Auth is the macaroon backend: the root key its signature chains verify
// against. The key is symmetric — holding it can mint tokens too, not just
// verify them, so a leaked config is a forging key living in two places.
type Auth struct {
	rootKey []byte
}

// New builds the backend from the section's "root_key" — the symmetric secret
// shared with whatever external system mints macaroons for this server.
func New(ctx context.Context, cfg scope.Section) (*Auth, error) {
	key, err := cfg.Get(ctx, "root_key")
	if err != nil {
		return nil, fmt.Errorf("macaroon: root_key: %w", err)
	}
	if key == "" {
		return nil, errors.New("macaroon: root_key must not be empty")
	}
	return &Auth{rootKey: []byte(key)}, nil
}

// Authenticate decodes and verifies token against the root key, parsing each
// first-party caveat as a Grant (see the package doc) — id becomes Account.
func (a *Auth) Authenticate(_ context.Context, token string) (access.Principal, error) {
	raw, err := joemacaroon.Base64Decode([]byte(token))
	if err != nil {
		return access.Principal{}, autherr.ErrUnauthenticated
	}
	var m joemacaroon.Macaroon
	if err := m.UnmarshalBinary(raw); err != nil {
		return access.Principal{}, autherr.ErrUnauthenticated
	}

	var caveats []access.Grant
	checkCaveat := func(condition string) error {
		g, err := access.ParseGrant(condition)
		if err != nil {
			return err
		}
		caveats = append(caveats, g)
		return nil
	}
	// nil discharges: a third-party caveat can never be satisfied, so it always
	// fails verify0's findDischarge and the whole macaroon is rejected with it.
	if err := m.Verify(a.rootKey, checkCaveat, nil); err != nil {
		return access.Principal{}, autherr.ErrUnauthenticated
	}

	account := string(m.Id())
	if account == "" {
		return access.Principal{}, autherr.ErrUnauthenticated
	}
	return access.Principal{Account: account, Caveats: caveats}, nil
}

// Scheme is the credential scheme consumed — a literal equal to auth.SchemeMacaroon.
func (a *Auth) Scheme() string { return "macaroon" }
