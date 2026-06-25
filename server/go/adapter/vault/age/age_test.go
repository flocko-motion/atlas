package age_test

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"rankedb/adapter/config"
	cmem "rankedb/adapter/config/mem"
	"rankedb/adapter/vault/age"
)

func TestEncryptStoreOpenUse(t *testing.T) {
	ctx := context.Background()
	const master = "correct horse battery staple"

	// Build a secret bundle: a connection string + a PEM signing key.
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, _ := x509.MarshalPKCS8PrivateKey(priv)
	keyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	secrets := map[string]string{"dsn": "postgres://x", "signing": keyPEM}

	// Encrypt → store in a config cell (vault.value.age) → open the vault.
	ct, err := age.Encrypt(master, secrets)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	cell := config.NewCell(cmem.New(), "vault.value.age")
	if err := cell.Set(ctx, ct); err != nil {
		t.Fatalf("cell.Set: %v", err)
	}

	v, err := age.New(ctx, master, cell)
	if err != nil {
		t.Fatalf("age.New: %v", err)
	}
	defer v.Close()

	sec, err := v.Secret("dsn")
	if err != nil || string(sec) != "postgres://x" {
		t.Fatalf("Secret(dsn) = (%q, %v), want postgres://x", sec, err)
	}

	signer, err := v.Signer("signing")
	if err != nil {
		t.Fatalf("Signer: %v", err)
	}
	msg := []byte("a branch table claim")
	sig, err := signer.Sign(rand.Reader, msg, crypto.Hash(0))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if !ed25519.Verify(pub, msg, sig) {
		t.Fatal("signature did not verify")
	}
}

func TestWrongMasterFails(t *testing.T) {
	ctx := context.Background()
	ct, err := age.Encrypt("right-master", map[string]string{"k": "v"})
	if err != nil {
		t.Fatal(err)
	}
	cell := config.NewCell(cmem.New(), "vault.value.age")
	if err := cell.Set(ctx, ct); err != nil {
		t.Fatal(err)
	}
	if _, err := age.New(ctx, "wrong-master", cell); err == nil {
		t.Fatal("expected decryption to fail with the wrong master")
	}
}
