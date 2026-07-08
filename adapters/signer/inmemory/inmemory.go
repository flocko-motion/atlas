// package: inmemory / crypto
// type:    adapter
// job:     load a config-provided ed25519 private key into a crypto.Signer the server signs merges with
// limits:  never generates a key; the key is supplied by config (inline, env(), or vault()) -> signer.New
//
// Package inmemory is the in-process signer backend: it holds the server's
// ed25519 private key in memory and signs with it directly. The key is always
// PROVIDED — a freshly generated key would attest claims as an anonymous nobody
// and would change on every restart, fragmenting the archive's merge chain
// across identities — so New loads supplied key material and never mints its
// own. config resolves where the material comes from (inline in an encrypted
// config, env(), or vault()) and hands New the resolved PEM.
package inmemory

import (
	"crypto"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/flocko-motion/rankedb/config/scope"
)

// New reads the resolved "key" value from the instance scope and parses it as
// an Ed25519 PKCS#8 PEM private key, returning it as a crypto.Signer. The key
// is required: this backend cannot sign without a provided identity (it does
// not generate one). config has already resolved any env()/vault() delegation,
// so the scope carries the literal PEM.
func New(cfg scope.Config) (crypto.Signer, error) {
	keyPEM, err := cfg.Require("key")
	if err != nil {
		return nil, fmt.Errorf("signer/inmemory: %w (supply an ed25519 private key inline, via env(), or via vault())", err)
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
	return ed, nil
}
