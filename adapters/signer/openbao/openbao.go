// package: openbao / crypto
// type:    adapter
// job:     sign via an OpenBao Transit key that never leaves the server
// limits:  ed25519 Transit keys, which stay in OpenBao (-> adapters/signer)
//
// Package openbao is the Transit signer backend. The private key is generated in
// and never leaves OpenBao; this holds only a client, the Transit mount, and the
// key name, and signs by calling transit/sign. Public reads the key's public half
// from transit/keys, and PrepareKey (the conformance-test hook) mints an ed25519
// key in Transit and pins the signer to it.
package openbao

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"

	openbao "github.com/openbao/openbao/api/v2"

	"github.com/flocko-motion/rankedb/config/scope"
)

// Signer signs via an OpenBao Transit key. It implements signer.Signer (and the
// private signer test view via PrepareKey).
type Signer struct {
	client *openbao.Client
	mount  string
	key    string
}

// New builds the Transit signer from the section: "address" (server URL) and
// "token" (auth) are required; "mount" defaults to "transit"; "key" is the Transit
// key name (optional — a test provisions it via PrepareKey). New is lenient: it
// does not read the key, so PrepareKey can create it afterwards.
func New(ctx context.Context, cfg scope.Section) (*Signer, error) {
	address, err := cfg.Get(ctx, "address")
	if err != nil {
		return nil, fmt.Errorf("signer/openbao: address: %w", err)
	}
	if address == "" {
		return nil, fmt.Errorf("signer/openbao: address is required")
	}
	token, err := cfg.Get(ctx, "token")
	if err != nil {
		return nil, fmt.Errorf("signer/openbao: token: %w", err)
	}
	if token == "" {
		return nil, fmt.Errorf("signer/openbao: token is required")
	}
	mount := "transit"
	if cfg.HasValue("mount") {
		if mount, err = cfg.Get(ctx, "mount"); err != nil {
			return nil, fmt.Errorf("signer/openbao: mount: %w", err)
		}
	}
	var key string
	if cfg.HasValue("key") {
		if key, err = cfg.Get(ctx, "key"); err != nil {
			return nil, fmt.Errorf("signer/openbao: key: %w", err)
		}
	}

	conf := openbao.DefaultConfig()
	conf.Address = address
	client, err := openbao.NewClient(conf)
	if err != nil {
		return nil, fmt.Errorf("signer/openbao: client: %w", err)
	}
	client.SetToken(token)
	return &Signer{client: client, mount: mount, key: key}, nil
}

// Sign signs hash with the Transit key and returns the raw signature bytes.
func (s *Signer) Sign(ctx context.Context, hash []byte) ([]byte, error) {
	if s.key == "" {
		return nil, fmt.Errorf("signer/openbao: no key configured")
	}
	resp, err := s.client.Logical().WriteWithContext(ctx, s.mount+"/sign/"+s.key, map[string]any{
		"input": base64.StdEncoding.EncodeToString(hash),
	})
	if err != nil {
		return nil, fmt.Errorf("signer/openbao: sign: %w", err)
	}
	raw, ok := resp.Data["signature"].(string)
	if !ok {
		return nil, fmt.Errorf("signer/openbao: sign: no signature in response")
	}
	// Transit returns "vault:v<version>:<base64(signature)>".
	i := strings.LastIndex(raw, ":")
	if i < 0 {
		return nil, fmt.Errorf("signer/openbao: sign: malformed signature %q", raw)
	}
	sig, err := base64.StdEncoding.DecodeString(raw[i+1:])
	if err != nil {
		return nil, fmt.Errorf("signer/openbao: sign: decode signature: %w", err)
	}
	return sig, nil
}

// Public reads the Transit key's ed25519 public key.
func (s *Signer) Public(ctx context.Context) (crypto.PublicKey, error) {
	if s.key == "" {
		return nil, fmt.Errorf("signer/openbao: no key configured")
	}
	return s.publicKey(ctx, s.key)
}

// PrepareKey mints the named ed25519 Transit key (if absent), pins this signer to
// it, and returns its public key — the conformance suite's setup hook.
func (s *Signer) PrepareKey(ctx context.Context, name string) (crypto.PublicKey, error) {
	if _, err := s.client.Logical().WriteWithContext(ctx, s.mount+"/keys/"+name, map[string]any{
		"type": "ed25519",
	}); err != nil {
		return nil, fmt.Errorf("signer/openbao: create key %q: %w", name, err)
	}
	s.key = name
	return s.publicKey(ctx, name)
}

// publicKey reads a Transit key's latest ed25519 public key.
func (s *Signer) publicKey(ctx context.Context, name string) (ed25519.PublicKey, error) {
	resp, err := s.client.Logical().ReadWithContext(ctx, s.mount+"/keys/"+name)
	if err != nil {
		return nil, fmt.Errorf("signer/openbao: read key %q: %w", name, err)
	}
	if resp == nil {
		return nil, fmt.Errorf("signer/openbao: key %q not found", name)
	}
	keys, ok := resp.Data["keys"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("signer/openbao: key %q: no versions in response", name)
	}
	latest := fmt.Sprintf("%v", resp.Data["latest_version"])
	entry, ok := keys[latest].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("signer/openbao: key %q: version %s missing", name, latest)
	}
	b64, ok := entry["public_key"].(string)
	if !ok {
		return nil, fmt.Errorf("signer/openbao: key %q: no public_key", name)
	}
	pub, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("signer/openbao: key %q: decode public_key: %w", name, err)
	}
	if len(pub) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("signer/openbao: key %q: public key is %d bytes, want %d", name, len(pub), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(pub), nil
}
