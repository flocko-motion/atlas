package config

import (
	"context"
	"fmt"
	"testing"
	"time"

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
	b := newVaultBox(nil)
	b.v = v
	b.once.Do(func() {})
	return b
}

// countingVault records how many times Secret is called, to observe caching.
type countingVault struct {
	val string
	n   *int
}

func (c countingVault) Secret(context.Context, string) (string, error) {
	*c.n++
	return c.val, nil
}

// flakyVault errors while *fail is set, to observe serve-stale behaviour.
type flakyVault struct {
	val  string
	fail *bool
}

func (f flakyVault) Secret(context.Context, string) (string, error) {
	if *f.fail {
		return "", fmt.Errorf("flaky: vault unreachable")
	}
	return f.val, nil
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

// TestVaultBoxCaches asserts a resolved secret is cached for the TTL — repeated
// reads within it hit the vault once — and re-fetched once the TTL lapses.
func TestVaultBoxCaches(t *testing.T) {
	ctx := context.Background()
	n := 0
	b := testBox(countingVault{val: "s3cr3t", n: &n})
	now := time.Unix(1000, 0)
	b.now = func() time.Time { return now }
	b.ttl = time.Minute

	for i := 0; i < 3; i++ {
		if got, err := b.secret(ctx, "key"); err != nil || got != "s3cr3t" {
			t.Fatalf("read %d: got %q, err %v", i, got, err)
		}
	}
	if n != 1 {
		t.Fatalf("within TTL: %d vault calls, want 1", n)
	}

	now = now.Add(2 * time.Minute) // lapse the TTL
	if _, err := b.secret(ctx, "key"); err != nil {
		t.Fatalf("post-TTL read: %v", err)
	}
	if n != 2 {
		t.Fatalf("after TTL: %d vault calls, want 2", n)
	}
}

// TestVaultBoxServesStaleOnError asserts a cached secret survives a vault outage:
// once primed, an expired entry falls back to the last-known-good value when the
// re-fetch fails, and a ref never cached still errors.
func TestVaultBoxServesStaleOnError(t *testing.T) {
	ctx := context.Background()
	fail := false
	b := testBox(flakyVault{val: "s3cr3t", fail: &fail})
	now := time.Unix(1000, 0)
	b.now = func() time.Time { return now }
	b.ttl = time.Minute

	if got, err := b.secret(ctx, "key"); err != nil || got != "s3cr3t" {
		t.Fatalf("prime: got %q, err %v", got, err)
	}

	fail = true                    // vault goes down
	now = now.Add(2 * time.Minute) // and the cached entry expires
	if got, err := b.secret(ctx, "key"); err != nil || got != "s3cr3t" {
		t.Fatalf("stale read: got %q, err %v, want the last-known-good value", got, err)
	}

	if _, err := b.secret(ctx, "never-cached"); err == nil {
		t.Fatal("a ref never cached must error when the vault is down")
	}
}
