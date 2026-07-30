// package: config / composition
// type:    func
// job:     decrypt an age-encrypted config with a caller-supplied passphrase source
// limits:  passphrase (scrypt) only; the source is built by the frontend (-> cmd/ranke-db)
//
// A reference-only config is plaintext; one carrying an inline secret is age-encrypted.
// decrypt sniffs the header and passes plaintext through, so Verify and Run can route
// every config through it and ask for a passphrase only when there is something to open.
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

// PassphraseSource yields the passphrase for an age-encrypted config, called at most
// once. The frontend supplies it — the CLI from --age-key, a TUI from a modal.
type PassphraseSource func() (string, error)

// decrypt returns plaintext config bytes, passing plaintext input through. A nil src on
// encrypted input errors, so a missing key never fails silently.
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

// looksEncrypted sniffs the age binary or armor header, so plaintext needs no key.
func looksEncrypted(data []byte) bool {
	return bytes.HasPrefix(data, []byte(ageBinaryMagic)) ||
		bytes.HasPrefix(data, []byte(ageArmorMagic))
}
