// package: noauthtest / authn
// type:    test-support
// job:     the noauth backend's conformance setup hook, beside the backend
// limits:  a test helper; only the conformance driver imports it (-> adapters/auth)
package noauthtest

import (
	"testing"

	"github.com/flocko-motion/rankedb/config/scope"
)

// Setup returns a noauth config that authenticates every credential as account, the
// empty string as the credential to present (noauth ignores it by design, so any
// value would do), and a no-op teardown — the backend has no counterpart to spin up.
func Setup(_ *testing.T, account string) (scope.Section, string, func()) {
	return scope.Literal(map[string]string{"type": "noauth", "subject": account}), "", func() {}
}
