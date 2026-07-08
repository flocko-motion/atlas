package config

import (
	"context"
	"fmt"
	"testing"

	"github.com/flocko-motion/rankedb/adapters/vault"
)

// stubVault is an in-memory vault.Vault double: config depends only on the vault
// port, so its secret resolution is tested against a double, not a real backend
// (the real OpenBao lives in that adapter's own test).
type stubVault map[string]string

func (s stubVault) Secret(_ context.Context, ref string) (string, error) {
	v, ok := s[ref]
	if !ok {
		return "", fmt.Errorf("stub: no secret %q", ref)
	}
	return v, nil
}

// testBox returns a vaultBox already holding v, so resolveValue's vault() path can
// be exercised against a double without building a real backend. Its once is spent
// up front, so get returns v without dialing.
func testBox(v vault.Vault) *vaultBox {
	b := &vaultBox{v: v}
	b.once.Do(func() {})
	return b
}

// TestResolveValue asserts literals pass through, env() references resolve (and
// fail loud when unset), and vault() references resolve through the port (and
// fail loud on a missing key or an absent vault).
func TestResolveValue(t *testing.T) {
	ctx := context.Background()

	if got, err := resolveValue(ctx, "plain", nil); err != nil || got != "plain" {
		t.Fatalf("literal: got %q, err %v", got, err)
	}

	t.Setenv("RANKE_TEST_ENV", "val")
	if got, err := resolveValue(ctx, "env(RANKE_TEST_ENV)", nil); err != nil || got != "val" {
		t.Fatalf("env set: got %q, err %v", got, err)
	}
	if _, err := resolveValue(ctx, "env(RANKE_TEST_UNSET)", nil); err == nil {
		t.Fatal("env unset: want error")
	}

	box := testBox(stubVault{"sig": "s3cr3t"})
	if got, err := resolveValue(ctx, "vault(sig)", box); err != nil || got != "s3cr3t" {
		t.Fatalf("vault hit: got %q, err %v", got, err)
	}
	if _, err := resolveValue(ctx, "vault(absent)", box); err == nil {
		t.Fatal("vault miss: want error")
	}
	if _, err := resolveValue(ctx, "vault(sig)", nil); err == nil {
		t.Fatal("vault ref with no box: want error")
	}
}
