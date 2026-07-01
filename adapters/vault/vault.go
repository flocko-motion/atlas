// package: vault / secrets
// type:    interface
// job:     the Vault port — resolve a named secret reference to its value (config vault(ref) expansion + adapter credentials)
// limits:  contract only; backends live in sub-packages (-> adapters/vault/openbao, adapters/vault/azure)
//
// Package vault defines the secret-resolution port. The config loader calls it
// to expand vault(ref) placeholders, and adapters draw connection credentials
// from it. It is optional: a stack whose every adapter needs no secret (mem/fs
// storage, in-memory signer, no auth) configures no vault at all.
package vault

import "context"

// Vault resolves secret references to their plaintext values. Backends: OpenBao
// KV and Azure Key Vault (-> sub-packages).
type Vault interface {
	// Secret returns the value stored under ref, or an error if it is absent
	// or the vault is unreachable.
	Secret(ctx context.Context, ref string) (string, error)
}
