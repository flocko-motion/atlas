package config

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"
)

// testKeyPEM generates a throwaway Ed25519 private key as PKCS#8 PEM. Tests
// legitimately fabricate a key to exercise loading; production keys are always
// provided by config, never generated.
func testKeyPEM(t *testing.T) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

// TestBuildResolvesAndWires loads a config whose signer key is delegated to an
// env var, and asserts Build resolves it, loads the key into a working signer,
// and wires the noauth subject — proving the scope handoff end to end.
func TestBuildResolvesAndWires(t *testing.T) {
	t.Setenv("RANKE_TEST_SIGNER_KEY", testKeyPEM(t))

	const cfgJSON = `{
		"signer": {"type": "inmemory", "key": "env(RANKE_TEST_SIGNER_KEY)"},
		"auth":   {"type": "noauth", "subject": "ops"}
	}`

	c, err := Load(strings.NewReader(cfgJSON))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	app, err := c.Build(context.Background(), nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if app.Signer == nil {
		t.Fatal("nil signer")
	}
	if _, ok := app.Signer.Public().(ed25519.PublicKey); !ok {
		t.Fatalf("signer public key = %T, want ed25519.PublicKey", app.Signer.Public())
	}
	digest := make([]byte, 32)
	if _, err := app.Signer.Sign(rand.Reader, digest, crypto.Hash(0)); err != nil {
		t.Fatalf("Sign: %v", err)
	}

	subject, err := app.Auth.Authenticate(context.Background(), "")
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if subject != "ops" {
		t.Fatalf("subject = %q, want %q", subject, "ops")
	}
}

// TestBuildMissingEnvFails asserts an unset env() delegation fails Build loud
// rather than yielding an empty key.
func TestBuildMissingEnvFails(t *testing.T) {
	const cfgJSON = `{"signer": {"type": "inmemory", "key": "env(RANKE_TEST_ABSENT)"}}`
	c, err := Load(strings.NewReader(cfgJSON))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := c.Build(context.Background(), nil); err == nil {
		t.Fatal("Build succeeded with an unset env delegation; want error")
	}
}
