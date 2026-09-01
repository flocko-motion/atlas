package openbao

import (
	"context"
	"testing"
	"time"

	openbao "github.com/openbao/openbao/api/v2"

	"github.com/rankegraph/ranke-db/config/scope"
	"github.com/rankegraph/ranke-db/tools/podman"
)

const testToken = "root-token-for-test"

// step narrates a phase of the test; visible under `go test -v`.
func step(t *testing.T, format string, args ...any) {
	t.Helper()
	t.Logf("▸ "+format, args...)
}

// TestSecret spins up a real OpenBao dev server via podman, seeds a KV v2 secret,
// and asserts the adapter reads it back through the vault.Vault port. It skips
// when podman is unavailable so the offline gate stays green — an adapter's only
// meaningful test drives its real counterpart, never a mock.
func TestSecret(t *testing.T) {
	ctx := context.Background()
	addr, teardown := podman.Run(t, podman.Spec{
		Image: "ghcr.io/openbao/openbao:latest",
		Port:  8200,
		Env:   map[string]string{"BAO_DEV_ROOT_TOKEN_ID": testToken},
		Args:  []string{"server", "-dev", "-dev-listen-address=0.0.0.0:8200"},
	})
	t.Cleanup(teardown)

	address := "http://" + addr
	step(t, "waiting for OpenBao to unseal at %s", address)
	seed := newClient(t, address)
	waitReady(t, seed)

	step(t, "seeding KV v2 secret secret/ranke/signing")
	if _, err := seed.KVv2("secret").Put(ctx, "ranke/signing", map[string]any{"key": "s3cr3t"}); err != nil {
		t.Fatalf("seed secret: %v", err)
	}

	step(t, "reading vault(ranke/signing#key) through the adapter")
	v, err := New(ctx, scope.Literal(map[string]string{
		"type":    "openbao",
		"address": address,
		"token":   testToken,
		"mount":   "secret",
	}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	got, err := v.Secret(ctx, "ranke/signing#key")
	if err != nil {
		t.Fatalf("Secret: %v", err)
	}
	if got != "s3cr3t" {
		t.Fatalf("Secret = %q, want %q", got, "s3cr3t")
	}
	step(t, "adapter returned the seeded secret intact — OK")
}

func newClient(t *testing.T, address string) *openbao.Client {
	t.Helper()
	conf := openbao.DefaultConfig()
	conf.Address = address
	c, err := openbao.NewClient(conf)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	c.SetToken(testToken)
	return c
}

func waitReady(t *testing.T, c *openbao.Client) {
	t.Helper()
	start := time.Now()
	deadline := start.Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if h, err := c.Sys().Health(); err == nil && h != nil && h.Initialized && !h.Sealed {
			step(t, "OpenBao ready after %s", time.Since(start).Round(100*time.Millisecond))
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
	t.Fatal("OpenBao did not become ready in time")
}
