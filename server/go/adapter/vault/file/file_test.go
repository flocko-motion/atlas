package file_test

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"rankedb/adapter/vault/file"
)

func TestSecretAndSigner(t *testing.T) {
	dir := t.TempDir()

	// a plain secret
	if err := os.WriteFile(filepath.Join(dir, "dsn"), []byte("postgres://x"), 0o600); err != nil {
		t.Fatal(err)
	}
	// a PKCS#8 PEM Ed25519 key
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	if err := os.WriteFile(filepath.Join(dir, "signing.pem"), keyPEM, 0o600); err != nil {
		t.Fatal(err)
	}

	v, err := file.New(dir)
	if err != nil {
		t.Fatalf("file.New: %v", err)
	}
	defer v.Close()

	sec, err := v.Secret("dsn")
	if err != nil || string(sec) != "postgres://x" {
		t.Fatalf("Secret = %q, %v; want postgres://x", sec, err)
	}

	signer, err := v.Signer("signing.pem")
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
}

// TestOpenerNeedsNoSecret covers the Opener path: a file vault opens unsealed
// (no secret-zero) and serves the same secrets.
func TestOpenerNeedsNoSecret(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dsn"), []byte("postgres://x"), 0o600); err != nil {
		t.Fatal(err)
	}

	op := file.Opener(dir)
	if op.NeedsSecret() {
		t.Fatal("file vault must open unsealed (NeedsSecret=false)")
	}
	v, err := op.Open(context.Background(), nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer v.Close()

	sec, err := v.Secret("dsn")
	if err != nil || string(sec) != "postgres://x" {
		t.Fatalf("Secret via Opener = %q, %v; want postgres://x", sec, err)
	}
}
