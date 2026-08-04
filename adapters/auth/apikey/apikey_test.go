package apikey

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func digest(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// TestAuthenticate covers the recognition logic directly: a known key resolves to
// its account, and an unknown, empty, or too-short key is rejected. (New's array
// parsing and the scheme-routed flow are covered by the config-level tests, which
// build the backend through the real array-capable config section.)
func TestAuthenticate(t *testing.T) {
	const key = "webapp-key-0123456789" // >= minKeyLength
	a := &Auth{byDigest: map[string]string{digest(key): "webapp"}}
	ctx := context.Background()

	if p, err := a.Authenticate(ctx, key); err != nil || p.Account != "webapp" {
		t.Fatalf("valid key: account=%q err=%v, want webapp/nil", p.Account, err)
	}
	if _, err := a.Authenticate(ctx, "wrong-key-0123456789"); err == nil {
		t.Fatal("unknown key: want error")
	}
	if _, err := a.Authenticate(ctx, ""); err == nil {
		t.Fatal("empty key: want error")
	}
	if _, err := a.Authenticate(ctx, "short"); err == nil {
		t.Fatal("too-short key: want error")
	}
}

func TestScheme(t *testing.T) {
	if got := (&Auth{}).Scheme(); got != "apikey" {
		t.Fatalf("Scheme() = %q, want apikey", got)
	}
}
