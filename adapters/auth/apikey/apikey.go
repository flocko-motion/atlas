// package: apikey / authn
// type:    adapter
// job:     authenticate a request by matching its API key against configured account keys
// limits:  recognises keys, never mints them; holds digests only (-> auth.New, internal/core/access)
//
// Package apikey maps a presented key to the account it authenticates as. The config
// carries each key's SHA-256 digest, never the key: recognising one needs no ability to
// reproduce it, so a leaked config yields no keys. Independent of the auth package that
// dispatches to it, to avoid an import cycle.
package apikey

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/rankegraph/ranke-db/adapters/auth/autherr"
	"github.com/rankegraph/ranke-db/config/scope"
	"github.com/rankegraph/ranke-db/internal/core/access"
)

// minKeyLength guards against a mistakenly configured "1234". Not an entropy claim:
// strength is the operator's to get right at generation.
const minKeyLength = 16

// Auth is the API-key backend: key digests mapped to accounts.
type Auth struct {
	byDigest map[string]string // hex sha256(key) -> account
}

// New builds the backend from the section's "keys" array, each an "account" with the
// "sha256" of its key. An empty set is a misconfiguration, not an empty allowlist.
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

// Authenticate hashes the presented key and returns the account it maps to.
func (a *Auth) Authenticate(_ context.Context, token string) (access.Principal, error) {
	if len(token) < minKeyLength {
		return access.Principal{}, autherr.ErrUnauthenticated
	}
	sum := sha256.Sum256([]byte(token))
	account, ok := a.byDigest[hex.EncodeToString(sum[:])]
	if !ok {
		return access.Principal{}, autherr.ErrUnauthenticated
	}
	return access.Principal{Account: account}, nil
}

// Scheme is the credential scheme consumed — a literal equal to auth.SchemeAPIKey.
func (a *Auth) Scheme() string { return "apikey" }
