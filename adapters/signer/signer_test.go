package signer

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"testing"

	"github.com/flocko-motion/rankedb/adapters/signer/inmemory/inmemorytest"
	"github.com/flocko-motion/rankedb/config/scope"
)

// backend is one signer backend under conformance. setup is its hook: it builds
// the backend's config — spinning up a real counterpart when it needs one (and
// t.Skip-ing when that is unavailable) — and returns a teardown. The driver calls
// every backend's setup the same way; each backend's setup lives beside the
// backend, out of this port test and out of the production binary.
type backend struct {
	name  string
	setup func(t *testing.T) (scope.Section, func())
}

var backends = []backend{
	{name: "inmemory", setup: inmemorytest.Setup},
}

// TestConformance runs every backend through its setup, the test-view
// constructor, and the shared suite.
func TestConformance(t *testing.T) {
	for _, b := range backends {
		t.Run(b.name, func(t *testing.T) {
			cfg, teardown := b.setup(t)
			defer teardown()
			s, err := newTestSigner(context.Background(), cfg)
			if err != nil {
				t.Fatalf("newTestSigner: %v", err)
			}
			conform(t, s)
		})
	}
}

// conform is the shared signer contract: the prepared key is the one the signer
// signs under, a valid signature verifies, and a corrupted one does not.
func conform(t *testing.T, s testSigner) {
	t.Helper()
	ctx := context.Background()

	pub, err := s.PrepareKey(ctx, "conformance")
	if err != nil {
		t.Fatalf("PrepareKey: %v", err)
	}
	key := pub.(ed25519.PublicKey)
	t.Logf("▸ prepared ed25519 key ed25519:%s", base64.RawStdEncoding.EncodeToString(key))

	got, err := s.Public(ctx)
	if err != nil {
		t.Fatalf("Public: %v", err)
	}
	if !got.(ed25519.PublicKey).Equal(key) {
		t.Fatal("Public() does not match the prepared key")
	}
	t.Log("▸ Public() reports the prepared key")

	hash := []byte("ranke-db signer conformance")
	sig, err := s.Sign(ctx, hash)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	t.Logf("▸ signed a %d-byte hash → %d-byte signature", len(hash), len(sig))
	if !ed25519.Verify(key, hash, sig) {
		t.Fatal("valid signature did not verify")
	}
	t.Log("▸ signature verifies against the public key")

	sig[0] ^= 0xff
	if ed25519.Verify(key, hash, sig) {
		t.Fatal("corrupted signature verified")
	}
	t.Log("▸ corrupted signature is rejected")
}

// TestNewRejects covers the configs New must reject: an empty or unknown backend
// type, and inmemory key material that is missing, non-PEM, or not ed25519.
func TestNewRejects(t *testing.T) {
	ctx := context.Background()
	bad := map[string]scope.Section{
		"empty type":           scope.Literal(map[string]string{}),
		"unknown type":         scope.Literal(map[string]string{"type": "nope"}),
		"inmemory missing key": scope.Literal(map[string]string{"type": "inmemory"}),
		"inmemory non-PEM key": scope.Literal(map[string]string{"type": "inmemory", "key": "not-a-pem"}),
		"inmemory non-ed25519": scope.Literal(map[string]string{"type": "inmemory", "key": inmemorytest.ECDSAPEM(t)}),
	}
	for name, cfg := range bad {
		if _, err := New(ctx, cfg); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}
