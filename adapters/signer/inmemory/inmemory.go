// package: inmemory / crypto
// type:    adapter
// job:     load a config-provided ed25519 private key into a signer.Signer the server signs merges with
// limits:  never generates a key; the key is supplied by config (inline, env(), or vault()) -> signer.New
//
// Package inmemory is the in-process signer backend: it holds the server's
// ed25519 private key in memory and signs with it directly. The key is always
// PROVIDED — a freshly generated key would attest claims as an anonymous nobody
// and would change on every restart, fragmenting the archive's merge chain
// across identities — so New loads supplied key material and never mints its own.
// config resolves where the material comes from (inline in an encrypted config,
// env(), or vault()) and hands New the section to read it from.
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

// Signer holds an ed25519 private key in memory and signs with it directly; it
// implements signer.Signer.
type Signer struct {
	key ed25519.PrivateKey
}

// Sign returns the ed25519 signature over hash. ed25519 is deterministic, so the
// same hash yields the same signature — the port's Sign contract.
func (s *Signer) Sign(_ context.Context, hash []byte) ([]byte, error) {
	return ed25519.Sign(s.key, hash), nil
}

// Public returns the ed25519 public key the signatures bind to.
func (s *Signer) Public(_ context.Context) (crypto.PublicKey, error) {
	return s.key.Public(), nil
}

// PrepareKey satisfies the signer conformance suite's test view. This backend is
// given its key by config rather than minting one, so it ignores name and returns
// the held key's public half.
func (s *Signer) PrepareKey(_ context.Context, _ string) (crypto.PublicKey, error) {
	return s.key.Public(), nil
}

// New reads the "key" value from the instance section and parses it as an
// Ed25519 PKCS#8 PEM private key. The key is required: this backend cannot sign
// without a provided identity (it does not generate one). The section resolves
// any env()/vault() delegation lazily, so reading the key here is where a missing
// secret fails.
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
