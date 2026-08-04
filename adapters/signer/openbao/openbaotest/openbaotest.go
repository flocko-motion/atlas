// package: openbaotest / crypto
// type:    test-support
// job:     the OpenBao Transit signer's conformance setup hook — a real OpenBao via podman, transit enabled
// limits:  a test helper; it skips when podman is absent (-> adapters/signer conformance, tools/podman)
//
// Package openbaotest is the OpenBao-specific setup for the signer conformance
// suite. It lives beside the openbao signer backend so all the OpenBao knowledge
// (dev-server flags, unseal wait, enabling the Transit engine) stays here; the
// port-level suite just calls Setup and runs the shared checks against the config
// it returns.
package openbaotest

import (
	"context"
	"testing"
	"time"

	openbao "github.com/openbao/openbao/api/v2"

	"github.com/flocko-motion/rankedb/config/scope"
	"github.com/flocko-motion/rankedb/tools/podman"
)

const token = "root-token-for-test"

// Setup spins a real OpenBao dev server via podman, enables the Transit engine,
// and returns the signer config plus a teardown. It skips when podman is absent.
func Setup(t *testing.T) (scope.Section, func()) {
	t.Helper()
	addr, teardown := podman.Run(t, podman.Spec{
		Image: "ghcr.io/openbao/openbao:latest",
		Port:  8200,
		Env:   map[string]string{"BAO_DEV_ROOT_TOKEN_ID": token},
		Args:  []string{"server", "-dev", "-dev-listen-address=0.0.0.0:8200"},
	})

	address := "http://" + addr
	conf := openbao.DefaultConfig()
	conf.Address = address
	client, err := openbao.NewClient(conf)
	if err != nil {
		teardown()
		t.Fatalf("openbao client: %v", err)
	}
	client.SetToken(token)

	t.Log("▸ waiting for OpenBao to unseal")
	deadline := time.Now().Add(30 * time.Second)
	for {
		if h, err := client.Sys().Health(); err == nil && h != nil && h.Initialized && !h.Sealed {
			break
		}
		if time.Now().After(deadline) {
			teardown()
			t.Fatal("OpenBao did not unseal in time")
		}
		time.Sleep(300 * time.Millisecond)
	}

	t.Log("▸ enabling the transit engine")
	if err := client.Sys().MountWithContext(context.Background(), "transit", &openbao.MountInput{Type: "transit"}); err != nil {
		teardown()
		t.Fatalf("enable transit: %v", err)
	}

	return scope.Literal(map[string]string{
		"type":    "openbao",
		"address": address,
		"token":   token,
		"mount":   "transit",
	}), teardown
}
