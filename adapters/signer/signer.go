// package: signer / crypto
// type:    interface + factory
// job:     the Signer port — the server's signing identity — plus its factory
// limits:  contract + dispatch; keys live in the backends (-> inmemory, openbao, azure)
//
// Package signer builds the server's merge-signing identity from config. It signs the
// branch-table claims; the key signing the CLAIMS is the application's.
//
// Narrower than crypto.Signer (no rand/opts) and ctx-aware, so an in-process key and one
// that never leaves an HSM present identically. Core adapts it at ranke-go's boundary.
package signer

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"

	"github.com/rankegraph/ranke-db/adapters/signer/inmemory"
	"github.com/rankegraph/ranke-db/adapters/signer/openbao"
	"github.com/rankegraph/ranke-db/config/scope"
)

// Signer is the server's signing identity: Sign is the paper's Sign(H(S(v))), Public the
// key it binds to. Backends: inmemory, OpenBao Transit, Azure Key Vault.
type Signer interface {
	Sign(ctx context.Context, hash []byte) ([]byte, error)
	Public(ctx context.Context) (crypto.PublicKey, error)
}

// testSigner is the conformance suite's view: a Signer that can also prepare the key the
// suite signs with. Private, so production never sees PrepareKey.
type testSigner interface {
	Signer
	PrepareKey(ctx context.Context, name string) (crypto.PublicKey, error)
}

// New builds the backend named by the section's "type" as the narrow production Signer.
func New(ctx context.Context, cfg scope.Section) (Signer, error) {
	return newTestSigner(ctx, cfg)
}

// Identity renders the signer as "<algorithm>:<base64 key>", the one form the launch log
// and health both use. An unreadable key yields the reason rather than hiding it.
func Identity(ctx context.Context, s Signer) string {
	if s == nil {
		return ""
	}
	pub, err := s.Public(ctx)
	if err != nil {
		return fmt.Sprintf("unavailable: %v", err)
	}
	if ed, ok := pub.(ed25519.PublicKey); ok {
		return "ed25519:" + base64.RawStdEncoding.EncodeToString(ed)
	}
	return fmt.Sprintf("%T", pub)
}

// newTestSigner builds the backend as a testSigner, which New downcasts to Signer.
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
