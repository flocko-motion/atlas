// package: config / composition
// type:    func
// job:     decrypt an age-encrypted config with a caller-supplied passphrase source
// limits:  passphrase (scrypt) only; the source is built by the frontend (-> cmd/ranke-db)
//
// This file is the config library's age layer. A reference-only config is
// plaintext; a config carrying an inline literal secret is age-encrypted, its
// passphrase supplied by a PassphraseSource the frontend builds (the CLI wraps
// prompt/stdin/env/file; a TUI wraps a modal). decrypt sniffs the age header and
// passes plaintext through untouched, so Verify and Run route every config
// through it and call the source only when the bytes are actually encrypted.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"filippo.io/age"
	"filippo.io/age/armor"
)

const (
	ageBinaryMagic = "age-encryption.org/v1"
	ageArmorMagic  = "-----BEGIN AGE ENCRYPTED FILE-----"
)

// PassphraseSource yields the passphrase that decrypts an age-encrypted config.
// It is called at most once, only when the config is actually encrypted. The
// frontend supplies it (the CLI builds one from --age-key; a TUI from a modal).
type PassphraseSource func() (string, error)

// decrypt returns plaintext config bytes. Plaintext input is returned unchanged.
// age-encrypted input (binary or armored) is decrypted with the passphrase from
// src; a nil src then errors, so an encrypted config never fails silently for
// want of a key.
func decrypt(data []byte, src PassphraseSource) ([]byte, error) {
	if !looksEncrypted(data) {
		return data, nil
	}
	if src == nil {
		return nil, errors.New("config is age-encrypted; supply a key source with --age-key (prompt|stdin|env:VAR|file:path)")
	}
	pass, err := src()
	if err != nil {
		return nil, err
	}
	id, err := age.NewScryptIdentity(pass)
	if err != nil {
		return nil, fmt.Errorf("age identity: %w", err)
	}
	var in io.Reader = bytes.NewReader(data)
	if bytes.HasPrefix(data, []byte(ageArmorMagic)) {
		in = armor.NewReader(in)
	}
	r, err := age.Decrypt(in, id)
	if err != nil {
		return nil, fmt.Errorf("age decrypt: %w", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read decrypted config: %w", err)
	}
	return out, nil
}

// looksEncrypted sniffs the age binary or armor header so plaintext config is
// recognized without a key and routed straight to the parser.
func looksEncrypted(data []byte) bool {
	return bytes.HasPrefix(data, []byte(ageBinaryMagic)) ||
		bytes.HasPrefix(data, []byte(ageArmorMagic))
}
