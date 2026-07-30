// package: podman / tools
// type:    test-support
// job:     run a throwaway container for an adapter's real-counterpart test, on a free port, torn down after
// limits:  a test helper; it skips without podman and waits for the port, not for readiness
//
// Package podman is the shared boilerplate for the real-counterpart adapter tests
// (OpenBao for the vault and signer ports, and more to come). An adapter's only
// meaningful test drives its real backend, spun up here via podman; Run publishes
// the container on a free host port, returns that address plus a teardown, and
// skips the test when podman is not installed so the offline gate stays green.
package podman

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

// Spec describes a container to run for a test.
type Spec struct {
	Image string            // image reference, e.g. ghcr.io/openbao/openbao:latest
	Port  int               // container port to publish
	Env   map[string]string // environment variables
	Args  []string          // command + args after the image
}

// Run starts spec's container on a free host port and returns its host address
// (127.0.0.1:PORT) plus a teardown that removes it. It skips the test when podman
// is not installed, and waits until the published port accepts connections; the
// caller performs any service-specific readiness (e.g. an unseal/health check).
func Run(t testing.TB, spec Spec) (addr string, teardown func()) {
	t.Helper()
	if _, err := exec.LookPath("podman"); err != nil {
		t.Skipf("podman not found; skipping %s test", spec.Image)
	}

	port := freePort(t)
	addr = fmt.Sprintf("127.0.0.1:%d", port)
	name := fmt.Sprintf("ranke-test-%d", port)

	args := []string{"run", "--rm", "-d", "--name", name, "-p", addr + ":" + strconv.Itoa(spec.Port)}
	for k, v := range spec.Env {
		args = append(args, "-e", k+"="+v)
	}
	args = append(args, spec.Image)
	args = append(args, spec.Args...)
	if out, err := exec.Command("podman", args...).CombinedOutput(); err != nil {
		t.Fatalf("podman run %s: %v: %s", spec.Image, err, out)
	}
	teardown = func() { _ = exec.Command("podman", "rm", "-f", name).Run() }

	waitPort(t, addr, teardown)
	return addr, teardown
}

func freePort(t testing.TB) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("free port: %v", err)
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port
}

func waitPort(t testing.TB, addr string, teardown func()) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			_ = conn.Close()
			return
		}
		if time.Now().After(deadline) {
			teardown()
			t.Fatalf("container port %s did not open in time", addr)
		}
		time.Sleep(200 * time.Millisecond)
	}
}
