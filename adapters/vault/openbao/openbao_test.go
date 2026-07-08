package openbao

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"testing"
	"time"

	openbao "github.com/openbao/openbao/api/v2"

	"github.com/flocko-motion/rankedb/config/scope"
)

const (
	testImage = "ghcr.io/openbao/openbao:latest"
	testToken = "root-token-for-test"
)

// TestSecret spins up a real OpenBao dev server via podman, seeds a KV v2
// secret, and asserts the adapter reads it back through the vault.Vault port. It
// skips when podman is unavailable so the offline gate stays green — an adapter's
// only meaningful test drives its real counterpart, never a mock.
func TestSecret(t *testing.T) {
	if _, err := exec.LookPath("podman"); err != nil {
		t.Skip("podman not found; skipping OpenBao adapter test")
	}
	ctx := context.Background()

	addr := fmt.Sprintf("127.0.0.1:%d", freePort(t))
	const name = "ranke-openbao-test"
	start := exec.Command("podman", "run", "--rm", "-d",
		"--name", name,
		"-p", addr+":8200",
		"-e", "BAO_DEV_ROOT_TOKEN_ID="+testToken,
		testImage,
		"server", "-dev", "-dev-listen-address=0.0.0.0:8200")
	if out, err := start.CombinedOutput(); err != nil {
		t.Fatalf("podman run: %v: %s", err, out)
	}
	t.Cleanup(func() { _ = exec.Command("podman", "rm", "-f", name).Run() })

	address := "http://" + addr
	seed := newClient(t, address)
	waitReady(t, seed)
	if _, err := seed.KVv2("secret").Put(ctx, "ranke/signing", map[string]any{"key": "s3cr3t"}); err != nil {
		t.Fatalf("seed secret: %v", err)
	}

	// Read it back through the adapter, built from a config section.
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
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if h, err := c.Sys().Health(); err == nil && h != nil && h.Initialized && !h.Sealed {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
	t.Fatal("OpenBao did not become ready in time")
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("free port: %v", err)
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port
}
