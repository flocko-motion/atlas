// package: config / composition
// type:    func
// job:     decrypt an age-encrypted config and resolve the operator-chosen passphrase source
// limits:  passphrase (scrypt) only; never a command-line literal (-> cmd/ranke-db run)
//
// This file is the config library's age layer. A reference-only config holds no
// secret and is plaintext; a config carrying an inline literal secret is
// age-encrypted, and its passphrase is supplied at launch from a source the
// operator chooses — prompt, stdin, env:VAR, or file:path — never as a literal
// on the command line. Decrypt sniffs the age header and passes plaintext
// through untouched, so callers always route config bytes through it.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"filippo.io/age"
	"filippo.io/age/armor"
	"golang.org/x/term"
)

const (
	ageBinaryMagic = "age-encryption.org/v1"
	ageArmorMagic  = "-----BEGIN AGE ENCRYPTED FILE-----"
)

// PassphraseSource yields the passphrase that decrypts an age-encrypted config.
// It is called at most once, only when the config is actually encrypted.
type PassphraseSource func() (string, error)

// PassphraseFrom builds a PassphraseSource from an operator-chosen spec:
// "prompt" reads from the terminal without echo, "stdin" reads a line from in,
// "env:VAR" reads environment variable VAR, and "file:path" reads a file's
// contents. An empty spec yields a nil source (config assumed plaintext). A
// literal passphrase as the spec is intentionally unsupported — the paper
// forbids passing the key as a command-line literal.
func PassphraseFrom(spec string, in io.Reader) (PassphraseSource, error) {
	switch {
	case spec == "":
		return nil, nil
	case spec == "prompt":
		return promptPassphrase, nil
	case spec == "stdin":
		return func() (string, error) {
			b, err := io.ReadAll(in)
			if err != nil {
				return "", fmt.Errorf("read age passphrase from stdin: %w", err)
			}
			return strings.TrimRight(string(b), "\r\n"), nil
		}, nil
	case strings.HasPrefix(spec, "env:"):
		name := strings.TrimPrefix(spec, "env:")
		return func() (string, error) {
			v, ok := os.LookupEnv(name)
			if !ok {
				return "", fmt.Errorf("age key env(%s) is not set", name)
			}
			return v, nil
		}, nil
	case strings.HasPrefix(spec, "file:"):
		path := strings.TrimPrefix(spec, "file:")
		return func() (string, error) {
			b, err := os.ReadFile(path)
			if err != nil {
				return "", fmt.Errorf("read age key file: %w", err)
			}
			return strings.TrimRight(string(b), "\r\n"), nil
		}, nil
	default:
		return nil, fmt.Errorf("unknown age key source %q; use prompt|stdin|env:VAR|file:path", spec)
	}
}

// promptPassphrase reads a passphrase from the controlling terminal without
// echo. It opens /dev/tty directly so it works even when stdin is the config
// pipe.
func promptPassphrase() (string, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", fmt.Errorf("open terminal for passphrase prompt: %w", err)
	}
	defer func() { _ = tty.Close() }()
	fmt.Fprint(tty, "age passphrase: ")
	b, err := term.ReadPassword(int(tty.Fd()))
	fmt.Fprintln(tty)
	if err != nil {
		return "", fmt.Errorf("read passphrase: %w", err)
	}
	return string(b), nil
}

// Decrypt returns plaintext config bytes. Plaintext input is returned
// unchanged. age-encrypted input (binary or armored) is decrypted with the
// passphrase from src; a nil src then errors, so an encrypted config never
// fails silently for want of a key.
func Decrypt(data []byte, src PassphraseSource) ([]byte, error) {
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
