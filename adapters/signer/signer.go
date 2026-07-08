// package: signer / crypto
// type:    interface + factory
// job:     the Signer port — the server's signing identity — plus the factory that builds a backend from config
// limits:  contract + dispatch; key handling lives in the backends (-> adapters/signer/inmemory, openbao, azure)
//
// Package signer defines the server's merge-signing identity and builds it from
// config. The server signs the branch-table claims (the hard timestamp) with it;
// the contributor key that signs the CLAIMS themselves is the application's.
//
// Per the foundation paper, identity is id(v) = Sign(H(S(v))): Sign takes a hash
// and returns a deterministic, self-describing signature bound to the signer's
// public key. The port is deliberately narrower than crypto.Signer — it drops the
// rand/opts we never use — and is ctx-aware, so an in-process key (inmemory) and a
// key that never leaves an HSM/KMS (OpenBao Transit, Azure Key Vault) present
// identically. When core attests a merge through ranke-go (which consumes a
// crypto.Signer), it adapts a Signer at the boundary.
package signer

import (
	"context"
	"crypto"
	"fmt"

	"github.com/flocko-motion/rankedb/adapters/signer/inmemory"
	"github.com/flocko-motion/rankedb/adapters/signer/openbao"
	"github.com/flocko-motion/rankedb/config/scope"
)

// Signer is the server's signing identity. Sign returns a deterministic,
// self-describing signature over hash (the paper's Sign(H(S(v)))); Public returns
// the public key the signature binds to. Backends: in-memory, OpenBao Transit,
// Azure Key Vault (-> sub-packages).
type Signer interface {
	Sign(ctx context.Context, hash []byte) ([]byte, error)
	Public(ctx context.Context) (crypto.PublicKey, error)
}

// testSigner is the conformance suite's view of a signer: a Signer that can also
// prepare the key the suite signs with — inmemory returns the key it was given,
// OpenBao mints one in Transit. It is private, so production (which only ever
// holds a Signer) never sees PrepareKey; the method is public on the backend
// types only because Go requires exported methods for a type in another package
// to satisfy the interface.
type testSigner interface {
	Signer
	PrepareKey(ctx context.Context, name string) (crypto.PublicKey, error)
}

// New builds the signer backend named by the section's "type", returning it as
// the narrow Signer for production. An empty or unknown type is an error.
func New(ctx context.Context, cfg scope.Section) (Signer, error) {
	return newTestSigner(ctx, cfg)
}

// newTestSigner builds the backend as a testSigner — the conformance suite's
// view. New downcasts the result to Signer. Delegated backends (openbao, azure)
// land in this switch as they are added.
func newTestSigner(ctx context.Context, cfg scope.Section) (testSigner, error) {
	var t string
	if cfg.HasValue("type") {
		var err error
		if t, err = cfg.Get(ctx, "type"); err != nil {
			return nil, fmt.Errorf("signer: type: %w", err)
		}
	}
	switch t {
	case "inmemory":
		return inmemory.New(ctx, cfg)
	case "openbao":
		return openbao.New(ctx, cfg)
	case "":
		return nil, fmt.Errorf("signer: no backend type configured")
	default:
		return nil, fmt.Errorf("signer: unknown backend type %q", t)
	}
}
