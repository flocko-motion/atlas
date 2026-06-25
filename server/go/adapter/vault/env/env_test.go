package env_test

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"testing"

	"rankedb/adapter/vault"
	"rankedb/adapter/vault/env"
)

func TestSecretSignerAndOpener(t *testing.T) {
	const prefix = "RANKE_TEST_VAULT_"
	t.Setenv(prefix+"DSN", "postgres://x")

	// a PKCS#8 PEM Ed25519 key, delivered via an env var
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(prefix+"KEY", string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})))

	v := env.New(prefix)
	defer v.Close()

	sec, err := v.Secret("DSN")
	if err != nil || string(sec) != "postgres://x" {
		t.Fatalf("Secret = %q, %v; want postgres://x", sec, err)
	}

	signer, err := v.Signer("KEY")
	if err != nil {
		t.Fatalf("Signer: %v", err)
	}
	msg := []byte("a branch table claim")
	sig, err := signer.Sign(rand.Reader, msg, crypto.Hash(0)) // Ed25519: no prehash
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if !ed25519.Verify(pub, msg, sig) {
		t.Fatal("signature did not verify against the generated public key")
	}

	// A missing var reports ErrNotFound, not an empty secret.
	if _, err := v.Secret("ABSENT"); !errors.Is(err, vault.ErrNotFound) {
		t.Fatalf("Secret(absent) err = %v; want ErrNotFound", err)
	}

	// Opener: the env vault needs no secret-zero — it opens unsealed.
	op := env.Opener(prefix)
	if op.NeedsSecret() {
		t.Fatal("env vault must open unsealed (NeedsSecret=false)")
	}
	v2, err := op.Open(context.Background(), nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer v2.Close()
	if sec, err := v2.Secret("DSN"); err != nil || string(sec) != "postgres://x" {
		t.Fatalf("Secret via Opener = %q, %v", sec, err)
	}
}
