// package: age / vault
// type:    adapter
// job:     production vault — secrets as a master-encrypted age blob (-> vault.Opener WithSecret)
// limits:  no security policy of its own; needs secret-zero, so it sits behind a seal gate (-> seal)
//
// Package age is the production vault: secrets live as a single
// age-encrypted blob (a passphrase-protected JSON map), held in a config
// Cell. New decrypts the blob with the master passphrase into a private
// in-memory map; Signer hands out a crypto.Signer whose key stays in RAM.
//
// The ciphertext is non-secret (the master is the only secret), so it rides
// in config like any other value — the Cell narrows the vault's authority to
// exactly that one config key.
package age

import (
	"bytes"
	"context"
	"crypto"
	"encoding/json"
	"fmt"
	"io"

	"filippo.io/age"

	"rankedb/adapter/config"
	"rankedb/adapter/vault"
)

// New opens an age vault: read the ciphertext from cell, decrypt with the
// master passphrase, parse the secret map. A wrong master fails decryption
// and returns an error (so a seal gate stays sealed).
func New(ctx context.Context, master string, cell config.Cell) (vault.Vault, error) {
	ct, err := cell.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("adapter/vault/age: read blob: %w", err)
	}
	id, err := age.NewScryptIdentity(master)
	if err != nil {
		return nil, err
	}
	r, err := age.Decrypt(bytes.NewReader(ct), id)
	if err != nil {
		return nil, fmt.Errorf("adapter/vault/age: decrypt (wrong master?): %w", err)
	}
	plain, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var m map[string]string
	if err := json.Unmarshal(plain, &m); err != nil {
		return nil, fmt.Errorf("adapter/vault/age: parse secrets: %w", err)
	}
	return &v{secrets: m}, nil
}

// Opener exposes the age vault as a vault.Opener that REQUIRES secret-zero
// (the master passphrase). The assembler fronts it with a seal gate.
func Opener(cell config.Cell) vault.Opener {
	return vault.WithSecret(func(ctx context.Context, secret []byte) (vault.Vault, error) {
		return New(ctx, string(secret), cell)
	})
}

// Encrypt produces the age ciphertext for a secret map, to be stored via a
// config.Cell — the admin/rotation counterpart to New.
func Encrypt(master string, secrets map[string]string) ([]byte, error) {
	rcp, err := age.NewScryptRecipient(master)
	if err != nil {
		return nil, err
	}
	plain, err := json.Marshal(secrets)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, rcp)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(plain); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type v struct{ secrets map[string]string }

func (x *v) Secret(name string) ([]byte, error) {
	s, ok := x.secrets[name]
	if !ok {
		return nil, vault.ErrNotFound
	}
	return []byte(s), nil
}

func (x *v) Signer(name string) (crypto.Signer, error) {
	s, ok := x.secrets[name]
	if !ok {
		return nil, vault.ErrNotFound
	}
	return vault.SignerFromPEM([]byte(s))
}

func (x *v) Close() error {
	for k := range x.secrets {
		delete(x.secrets, k)
	}
	x.secrets = nil
	return nil
}
