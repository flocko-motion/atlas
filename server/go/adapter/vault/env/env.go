// package: env / vault
// type:    adapter
// job:     12-factor vault — read secrets and PEM keys from env vars named prefix+name
// limits:  no encryption or storage; needs no secret-zero, so it opens unsealed (-> vault.NoSecret)
//
// Package env is an environment-variable vault (12-factor style): secrets
// and PEM-encoded keys are read from env vars named prefix+name.
package env

import (
	"context"
	"crypto"
	"os"

	"rankedb/adapter/vault"
)

// New returns a vault reading from env vars named prefix+name.
func New(prefix string) vault.Vault { return &v{prefix: prefix} }

// Opener exposes the env vault as a vault.Opener that needs NO secret-zero —
// so the assembler opens it immediately, unsealed.
func Opener(prefix string) vault.Opener {
	return vault.NoSecret(func(context.Context) (vault.Vault, error) { return New(prefix), nil })
}

type v struct{ prefix string }

func (e *v) lookup(name string) (string, bool) {
	return os.LookupEnv(e.prefix + name)
}

func (e *v) Secret(name string) ([]byte, error) {
	s, ok := e.lookup(name)
	if !ok {
		return nil, vault.ErrNotFound
	}
	return []byte(s), nil
}

func (e *v) Signer(name string) (crypto.Signer, error) {
	s, ok := e.lookup(name)
	if !ok {
		return nil, vault.ErrNotFound
	}
	return vault.SignerFromPEM([]byte(s))
}

func (e *v) Close() error { return nil }
