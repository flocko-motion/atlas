package config

import (
	"context"
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
		"auth":   [{"type": "noauth", "subject": "ops"}]
	}`

	app, err := Run(context.Background(), strings.NewReader(cfgJSON), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if app.Signer == nil {
		t.Fatal("nil signer")
	}
	pub, err := app.Signer.Public(context.Background())
	if err != nil {
		t.Fatalf("Public: %v", err)
	}
	if _, ok := pub.(ed25519.PublicKey); !ok {
		t.Fatalf("signer public key = %T, want ed25519.PublicKey", pub)
	}
	digest := make([]byte, 32)
	if _, err := app.Signer.Sign(context.Background(), digest); err != nil {
		t.Fatalf("Sign: %v", err)
	}

	if len(app.Auth) != 1 {
		t.Fatalf("auth count = %d, want 1", len(app.Auth))
	}
	subject, err := app.Auth[0].Authenticate(context.Background(), "")
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
	if _, err := Run(context.Background(), strings.NewReader(cfgJSON), nil); err == nil {
		t.Fatal("Run succeeded with an unset env delegation; want error")
	}
}

// TestVerify exercises the two verify depths: syntax is offline (needs no env),
// resolve additionally requires every reference to resolve.
func TestVerify(t *testing.T) {
	const good = `{"signer": {"type": "inmemory", "key": "env(RANKE_TEST_VERIFY_KEY)"}}`

	if err := Verify(context.Background(), strings.NewReader(good), nil, LevelSyntax); err != nil {
		t.Fatalf("Verify syntax: %v", err)
	}
	if err := Verify(context.Background(), strings.NewReader(`{"nope": {}}`), nil, LevelSyntax); err == nil {
		t.Fatal("Verify syntax accepted an unknown section")
	}
	if err := Verify(context.Background(), strings.NewReader(good), nil, LevelResolve); err == nil {
		t.Fatal("Verify resolve accepted an unset env reference")
	}
	t.Setenv("RANKE_TEST_VERIFY_KEY", "x")
	if err := Verify(context.Background(), strings.NewReader(good), nil, LevelResolve); err != nil {
		t.Fatalf("Verify resolve: %v", err)
	}
}

// TestVerifyConnect exercises the connect depth: a shape-valid config with an
// unknown backend passes syntax but fails connect, while a valid mem+signer
// config assembles cleanly.
func TestVerifyConnect(t *testing.T) {
	t.Setenv("RANKE_TEST_CONNECT_KEY", testKeyPEM(t))
	const good = `{"signer": {"type": "inmemory", "key": "env(RANKE_TEST_CONNECT_KEY)"}, "storage": {"type": "mem"}}`
	if err := Verify(context.Background(), strings.NewReader(good), nil, LevelConnect); err != nil {
		t.Fatalf("Verify connect: %v", err)
	}

	const badBackend = `{"storage": {"type": "bogus"}}`
	if err := Verify(context.Background(), strings.NewReader(badBackend), nil, LevelSyntax); err != nil {
		t.Fatalf("Verify syntax rejected a shape-valid config: %v", err)
	}
	if err := Verify(context.Background(), strings.NewReader(badBackend), nil, LevelConnect); err == nil {
		t.Fatal("Verify connect accepted an unknown storage backend")
	}
}
