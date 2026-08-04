package main

import (
	"strings"
	"testing"
)

// TestPassphraseFrom covers the age key source specs: an empty spec yields no
// source, a literal is rejected (never pass the key as a CLI argument), and the
// env/stdin sources resolve their passphrase.
func TestPassphraseFrom(t *testing.T) {
	empty, err := passphraseFrom("", nil)
	if err != nil {
		t.Fatalf("passphraseFrom(empty): %v", err)
	}
	if empty != nil {
		t.Fatal("passphraseFrom(empty) = non-nil source; want nil for plaintext")
	}

	if _, err := passphraseFrom("literal-secret", nil); err == nil {
		t.Fatal("passphraseFrom(literal) = nil error; a CLI literal must be rejected")
	}

	t.Run("env", func(t *testing.T) {
		t.Setenv("RANKE_TEST_KEYSRC", "sekret")
		src, err := passphraseFrom("env:RANKE_TEST_KEYSRC", nil)
		if err != nil {
			t.Fatalf("passphraseFrom: %v", err)
		}
		got, err := src()
		if err != nil {
			t.Fatalf("src: %v", err)
		}
		if got != "sekret" {
			t.Fatalf("env source = %q, want %q", got, "sekret")
		}
	})

	t.Run("stdin", func(t *testing.T) {
		src, err := passphraseFrom("stdin", strings.NewReader("piped\n"))
		if err != nil {
			t.Fatalf("passphraseFrom: %v", err)
		}
		got, err := src()
		if err != nil {
			t.Fatalf("src: %v", err)
		}
		if got != "piped" {
			t.Fatalf("stdin source = %q, want %q", got, "piped")
		}
	})
}
