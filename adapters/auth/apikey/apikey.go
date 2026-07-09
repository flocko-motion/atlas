// package: apikey / authn
// type:    adapter
// job:     authenticate a request by matching its API key against configured account keys
// limits:  recognises keys, does not mint them; holds only digests, never the raw keys (-> auth.New, internal/core/access)
//
// Package apikey is the API-key authentication backend: it maps a presented key to
// the system account it authenticates as. The config carries the SHA-256 digest of
// each key, not the key itself — the server needs only to recognise a key, never to
// reproduce it, so a compromise of the running config or process never yields the
// keys. The client sends the raw key (over TLS); this backend hashes it and looks
// up the account. Like the other auth backends it stays independent of the auth
// package (returns access.Principal and its own error, reports its scheme as a
// literal), so auth.New can dispatch to it without an import cycle.
package apikey

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/flocko-motion/rankedb/config/scope"
	"github.com/flocko-motion/rankedb/internal/core/access"
)

// minKeyLength is the shortest key this backend will honour. It is a guardrail
// against trivially short keys (a mistakenly configured "1234"), not an entropy
// guarantee — a key's real strength is the operator's responsibility at
// generation, and length does not imply entropy across encodings.
const minKeyLength = 16

// errUnauthenticated is returned for any credential this backend does not accept.
// It is package-local (not auth.ErrUnauthenticated) to keep the backend free of an
// import cycle with the dispatching auth package; the endpoint maps it to 401.
var errUnauthenticated = errors.New("apikey: unauthenticated")

// Auth is the API-key backend: a set of key digests, each mapped to the account it
// authenticates as.
type Auth struct {
	byDigest map[string]string // hex sha256(key) -> account
}

// New builds the backend from the section's "keys" array, each entry an object
// with an "account" and the "sha256" hex digest of that account's key. It rejects
// a malformed digest, an empty account, a duplicate digest, and an empty key set
// (a backend that authenticates nobody is a misconfiguration).
func New(ctx context.Context, cfg scope.Section) (*Auth, error) {
	entries := cfg.GetArray("keys")
	if len(entries) == 0 {
		return nil, errors.New("apikey: no keys configured")
	}
	byDigest := make(map[string]string, len(entries))
	for i, e := range entries {
		account, err := e.Get(ctx, "account")
		if err != nil {
			return nil, fmt.Errorf("apikey: keys[%d]: account: %w", i, err)
		}
		if account == "" {
			return nil, fmt.Errorf("apikey: keys[%d]: empty account", i)
		}
		digest, err := e.Get(ctx, "sha256")
		if err != nil {
			return nil, fmt.Errorf("apikey: keys[%d]: sha256: %w", i, err)
		}
		digest = strings.ToLower(digest)
		if raw, err := hex.DecodeString(digest); err != nil || len(raw) != sha256.Size {
			return nil, fmt.Errorf("apikey: keys[%d]: sha256 must be %d-byte hex", i, sha256.Size)
		}
		if _, dup := byDigest[digest]; dup {
			return nil, fmt.Errorf("apikey: keys[%d]: duplicate sha256", i)
		}
		byDigest[digest] = account
	}
	return &Auth{byDigest: byDigest}, nil
}

// Authenticate hashes the presented key and returns the account it maps to. A key
// shorter than minKeyLength, or one whose digest is not configured, is rejected.
func (a *Auth) Authenticate(_ context.Context, token string) (access.Principal, error) {
	if len(token) < minKeyLength {
		return access.Principal{}, errUnauthenticated
	}
	sum := sha256.Sum256([]byte(token))
	account, ok := a.byDigest[hex.EncodeToString(sum[:])]
	if !ok {
		return access.Principal{}, errUnauthenticated
	}
	return access.Principal{Account: account}, nil
}

// Scheme reports the credential scheme this backend consumes. Returned as a literal
// so apikey need not import the auth package that dispatches to it; it equals
// auth.SchemeAPIKey.
func (a *Auth) Scheme() string { return "apikey" }
