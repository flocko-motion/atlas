// package: inmemorytest / crypto
// type:    test-support
// job:     the inmemory signer's conformance setup hook and key fixtures, beside the backend
// limits:  a test helper; only the conformance driver imports it (-> adapters/signer)
//
// Package inmemorytest is the inmemory signer's setup hook for the signer
// conformance suite. It lives beside the backend (not in the port test) and out
// of the production binary, mirroring openbaotest — the difference is only that
// inmemory's counterpart is in-process, so setup just mints a key and there is
// nothing to tear down.
package inmemorytest

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/flocko-motion/rankedb/config/scope"
)

// Setup returns the inmemory signer's conformance config — a freshly generated
// ed25519 key as the "key" value — and a no-op teardown (the counterpart is
// in-process).
func Setup(t *testing.T) (scope.Section, func()) {
	t.Helper()
	return scope.Literal(map[string]string{"type": "inmemory", "key": Ed25519PEM(t)}), func() {}
}

// Ed25519PEM generates an Ed25519 private key as PKCS#8 PEM.
func Ed25519PEM(t *testing.T) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ed25519: %v", err)
	}
	return pkcs8PEM(t, priv)
}

// ECDSAPEM generates an ECDSA P-256 key as PKCS#8 PEM — a valid PKCS#8 key that
// is not ed25519, for exercising the inmemory type check.
func ECDSAPEM(t *testing.T) string {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ecdsa: %v", err)
	}
	return pkcs8PEM(t, priv)
}

func pkcs8PEM(t *testing.T, key any) string {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal PKCS#8: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}
