// package: file / vault
// type:    adapter
// job:     filesystem vault — each secret/key is a plaintext file in a directory, addressed by name
// limits:  no security (plaintext at rest); needs no secret-zero, so it opens unsealed (-> vault.NoSecret)
//
// Package file is a filesystem vault: each secret/key is a file in a
// directory, addressed by name (key files hold a PKCS#8 PEM private key).
package file

import (
	"context"
	"crypto"
	"errors"
	"os"
	"path/filepath"

	"rankedb/adapter/vault"
)

// New returns a vault backed by the files in dir.
func New(dir string) (vault.Vault, error) {
	if dir == "" {
		return nil, errors.New("adapter/vault/file.New: empty dir")
	}
	return &v{dir: dir}, nil
}

// Opener exposes the file vault as a vault.Opener that needs NO secret-zero
// (plaintext files) — so the assembler opens it immediately, unsealed.
func Opener(dir string) vault.Opener {
	return vault.NoSecret(func(context.Context) (vault.Vault, error) { return New(dir) })
}

type v struct{ dir string }

func (f *v) read(name string) ([]byte, error) {
	b, err := os.ReadFile(filepath.Join(f.dir, name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, vault.ErrNotFound
		}
		return nil, err
	}
	return b, nil
}

func (f *v) Secret(name string) ([]byte, error) { return f.read(name) }

func (f *v) Signer(name string) (crypto.Signer, error) {
	b, err := f.read(name)
	if err != nil {
		return nil, err
	}
	return vault.SignerFromPEM(b)
}

func (f *v) Close() error { return nil }
