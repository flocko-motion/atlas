// package: inmemory / crypto
// type:    adapter
// job:     load a config-provided ed25519 private key into a signer.Signer the server signs merges with
// limits:  never generates a key; the key is supplied by config (inline, env(), or vault()) -> signer.New
//
// Package inmemory holds the server's ed25519 key in memory and signs with it directly.
// The key is always PROVIDED: a generated one would change every restart, fragmenting the
// merge chain across identities. config resolves where the material comes from.
package inmemory

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/flocko-motion/rankedb/config/scope"
)

// Signer holds an ed25519 private key in memory; it implements signer.Signer.
type Signer struct {
	key ed25519.PrivateKey
}

// Sign returns the ed25519 signature over hash, deterministic as the port requires.
func (s *Signer) Sign(_ context.Context, hash []byte) ([]byte, error) {
	return ed25519.Sign(s.key, hash), nil
}

// Public returns the ed25519 public key the signatures bind to.
func (s *Signer) Public(_ context.Context) (crypto.PublicKey, error) {
	return s.key.Public(), nil
}

// PrepareKey satisfies the conformance suite. This backend is given its key, so name is
// ignored and the held key's public half comes back.
func (s *Signer) PrepareKey(_ context.Context, _ string) (crypto.PublicKey, error) {
	return s.key.Public(), nil
}

// New parses the section's "key" as an Ed25519 PKCS#8 PEM private key, required because
// this backend mints none. Reading it here is where a missing secret fails.
func New(ctx context.Context, cfg scope.Section) (*Signer, error) {
	keyPEM, err := cfg.Get(ctx, "key")
	if err != nil {
		return nil, fmt.Errorf("signer/inmemory: %w (supply an ed25519 private key inline, via env(), or via vault())", err)
	}
	if keyPEM == "" {
		return nil, errors.New("signer/inmemory: key is required (supply an ed25519 private key inline, via env(), or via vault())")
	}
	block, _ := pem.Decode([]byte(keyPEM))
	if block == nil {
		return nil, errors.New("signer/inmemory: key material is not valid PEM")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("signer/inmemory: parse PKCS#8 private key: %w", err)
	}
	ed, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("signer/inmemory: key is %T, want ed25519.PrivateKey", key)
	}
	return &Signer{key: ed}, nil
}
