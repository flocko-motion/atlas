// package: apikeytest / authn
// type:    test-support
// job:     the apikey backend's conformance setup hook, beside the backend
// limits:  a test helper; only the conformance driver imports it (-> adapters/auth)
package apikeytest

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/flocko-motion/rankedb/config/scope"
)

// conformanceKey is the credential Setup provisions its config around — fixed rather
// than generated, since apikey needs no entropy to prove the backend recognises a
// key it was configured with.
const conformanceKey = "apikeytest-conformance-key-0123456789"

// Setup returns an apikey config accepting conformanceKey as account (via its
// digest, the only form apikey's config ever carries), that same key to present to
// Authenticate, and a no-op teardown — the backend has no external counterpart.
func Setup(_ *testing.T, account string) (scope.Section, string, func()) {
	sum := sha256.Sum256([]byte(conformanceKey))
	cfg := scope.LiteralArray(
		map[string]string{"type": "apikey"},
		map[string][]scope.Section{
			"keys": {scope.Literal(map[string]string{"account": account, "sha256": hex.EncodeToString(sum[:])})},
		},
	)
	return cfg, conformanceKey, func() {}
}
