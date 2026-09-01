// package: macaroontest / authn
// type:    test-support
// job:     the macaroon backend's conformance setup hook, and a minting helper for fixtures
// limits:  a test helper; only tests import it — this server never mints a macaroon itself
package macaroontest

import (
	"encoding/base64"
	"testing"

	joemacaroon "gopkg.in/macaroon.v2"

	"github.com/rankegraph/ranke-db/config/scope"
)

// RootKey is the shared secret Setup and every fixture built through Mint sign
// against — fixed rather than generated, since a conformance run has no need for
// entropy, only for one config and one minting side to agree on the same key.
const RootKey = "macaroontest-conformance-root-key"

// Setup returns a macaroon config over RootKey, a token minted for account with no
// caveats, and a no-op teardown — the backend has no external counterpart.
func Setup(t *testing.T, account string) (scope.Section, string, func()) {
	t.Helper()
	token := Mint(t, RootKey, account)
	cfg := scope.Literal(map[string]string{"type": "macaroon", "root_key": RootKey})
	return cfg, token, func() {}
}

// Mint builds a macaroon whose id is account, signed under rootKey, with
// conditions added as first-party caveats in order — the minting a real external
// system does, reproduced here only because a test needs something to verify
// against. Not a capability this package's production code offers.
func Mint(t *testing.T, rootKey, account string, conditions ...string) string {
	t.Helper()
	m, err := joemacaroon.New([]byte(rootKey), []byte(account), "", joemacaroon.V2)
	if err != nil {
		t.Fatalf("macaroon.New: %v", err)
	}
	for _, cond := range conditions {
		if err := m.AddFirstPartyCaveat([]byte(cond)); err != nil {
			t.Fatalf("AddFirstPartyCaveat(%q): %v", cond, err)
		}
	}
	raw, err := m.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}
