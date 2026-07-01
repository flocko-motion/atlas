package config

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"filippo.io/age"
)

// encryptForTest age-encrypts plaintext under a scrypt passphrase, returning the
// binary age stream — the format an operator commits when a config carries an
// inline secret.
func encryptForTest(t *testing.T, plaintext, passphrase string) []byte {
	t.Helper()
	recip, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		t.Fatalf("NewScryptRecipient: %v", err)
	}
	recip.SetWorkFactor(10) // keep the test fast
	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, recip)
	if err != nil {
		t.Fatalf("age.Encrypt: %v", err)
	}
	if _, err := io.WriteString(w, plaintext); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return buf.Bytes()
}

// TestDecryptRoundTrip encrypts a config and decrypts it through each non-prompt
// passphrase source, asserting the plaintext returns intact.
func TestDecryptRoundTrip(t *testing.T) {
	const plaintext = `{"auth":{"type":"noauth","subject":"ops"}}`
	const pass = "correct horse battery staple"
	enc := encryptForTest(t, plaintext, pass)

	t.Run("env", func(t *testing.T) {
		t.Setenv("RANKE_TEST_AGE_KEY", pass)
		src, err := PassphraseFrom("env:RANKE_TEST_AGE_KEY", nil)
		if err != nil {
			t.Fatalf("PassphraseFrom: %v", err)
		}
		got, err := Decrypt(enc, src)
		if err != nil {
			t.Fatalf("Decrypt: %v", err)
		}
		if string(got) != plaintext {
			t.Fatalf("decrypted = %q, want %q", got, plaintext)
		}
	})

	t.Run("stdin", func(t *testing.T) {
		src, err := PassphraseFrom("stdin", strings.NewReader(pass+"\n"))
		if err != nil {
			t.Fatalf("PassphraseFrom: %v", err)
		}
		got, err := Decrypt(enc, src)
		if err != nil {
			t.Fatalf("Decrypt: %v", err)
		}
		if string(got) != plaintext {
			t.Fatalf("decrypted = %q, want %q", got, plaintext)
		}
	})
}

// TestDecryptPlaintextPassthrough asserts a plaintext config is returned
// unchanged and needs no key source.
func TestDecryptPlaintextPassthrough(t *testing.T) {
	plain := []byte(`{"auth":{"type":"noauth"}}`)
	got, err := Decrypt(plain, nil)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("plaintext passthrough = %q, want %q", got, plain)
	}
}

// TestDecryptEncryptedNoKey asserts an encrypted config with no key source
// fails loud rather than silently.
func TestDecryptEncryptedNoKey(t *testing.T) {
	enc := encryptForTest(t, `{}`, "pw")
	if _, err := Decrypt(enc, nil); err == nil {
		t.Fatal("Decrypt of encrypted config with nil source = nil error, want error")
	}
}

// TestPassphraseFromBadSpec rejects an unknown source spec.
func TestPassphraseFromBadSpec(t *testing.T) {
	if _, err := PassphraseFrom("literal-secret", nil); err == nil {
		t.Fatal("PassphraseFrom(literal) = nil error; a CLI literal must be rejected")
	}
}
