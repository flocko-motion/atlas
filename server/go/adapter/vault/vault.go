// package: vault / contract
// type:    interface
// job:     define the secret-store seam — Vault (Secret + Signer) and Opener (-> backends in vault/{age,env,file})
// limits:  knows nothing of config; never holds a backend impl — those are sub-packages
//
// Package vault holds the secret-store seam for ranke-db: connection
// strings, private keys, anything a deployment must keep out of its
// (non-secret, version-controlled) config. config holds REFERENCES into a
// vault; the vault knows nothing of config — it is a pure leaf.
//
// Signer returns a crypto.Signer rather than raw key bytes, so a backend
// may keep the key inside its boundary (HSM/KMS) and never expose it;
// file/env backends load it into memory. Concrete backends live in
// sub-packages (vault/env, vault/file, …).
package vault

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

// Vault stores the secrets a ranke-db deployment needs, addressed by name.
type Vault interface {
	// Secret returns the secret bytes stored under name (e.g. a DB DSN or
	// an S3 access key).
	Secret(name string) ([]byte, error)
	// Signer returns a crypto.Signer for the key stored under name. The key
	// may never leave the vault (HSM/KMS); file/env load it into memory.
	Signer(name string) (crypto.Signer, error)
	// Close releases any resources the vault holds.
	Close() error
}

// ErrNotFound reports an absent secret or key.
var ErrNotFound = errors.New("vault: not found")

// SignerFromPEM parses a PKCS#8 PEM-encoded private key into a crypto.Signer
// (Ed25519, ECDSA, RSA — whatever PKCS#8 carries). Shared by the in-memory
// backends so they don't each reimplement key parsing.
func SignerFromPEM(data []byte) (crypto.Signer, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("vault: no PEM block in key data")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("vault: parse PKCS#8 key: %w", err)
	}
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("vault: key type %T is not a crypto.Signer", key)
	}
	return signer, nil
}

// Opener builds a Vault, possibly requiring secret-zero. The assembler asks
// NeedsSecret to decide whether to front the vault with a seal gate (env or
// /unlock): if true, it stays sealed until the master arrives; if false, the
// vault opens immediately at boot with no sealing. The backend declares its
// own requirement — the assembler doesn't hardcode which vaults need a key.
type Opener interface {
	NeedsSecret() bool
	Open(ctx context.Context, secret []byte) (Vault, error)
}

// NoSecret wraps a vault that opens with no secret-zero (e.g. file, env).
func NoSecret(open func(ctx context.Context) (Vault, error)) Opener {
	return opener{open: func(ctx context.Context, _ []byte) (Vault, error) { return open(ctx) }}
}

// WithSecret wraps a vault that requires secret-zero to open (e.g. age).
func WithSecret(open func(ctx context.Context, secret []byte) (Vault, error)) Opener {
	return opener{needsSecret: true, open: open}
}

type opener struct {
	needsSecret bool
	open        func(ctx context.Context, secret []byte) (Vault, error)
}

func (o opener) NeedsSecret() bool { return o.needsSecret }

func (o opener) Open(ctx context.Context, secret []byte) (Vault, error) {
	return o.open(ctx, secret)
}
