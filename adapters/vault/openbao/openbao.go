// package: openbao / secrets
// type:    adapter
// job:     resolve vault(ref) secrets from an OpenBao KV v2 engine
// limits:  KV v2 reads only; the mount + credentials come from the vault section (-> adapters/vault)
//
// Package openbao is the OpenBao secret backend. It reads secrets from a KV v2
// engine: a ref is "path#field" — the KV v2 secret at path (under the configured
// mount), returning the named field. Address, token, and mount come from the
// vault section, resolved like any other config value.
package openbao

import (
	"context"
	"fmt"
	"strings"

	openbao "github.com/openbao/openbao/api/v2"

	"github.com/flocko-motion/rankedb/config/scope"
)

// Vault reads secrets from an OpenBao KV v2 engine.
type Vault struct {
	client *openbao.Client
	mount  string
}

// New builds the OpenBao backend from the vault section: "address" (server URL,
// required), "token" (auth token, required), and "mount" (the KV v2 mount path,
// default "secret").
func New(ctx context.Context, cfg scope.Section) (*Vault, error) {
	address, err := cfg.GetValue("address").Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("vault/openbao: address: %w", err)
	}
	if address == "" {
		return nil, fmt.Errorf("vault/openbao: address is required")
	}
	token, err := cfg.GetValue("token").Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("vault/openbao: token: %w", err)
	}
	if token == "" {
		return nil, fmt.Errorf("vault/openbao: token is required")
	}
	mount := "secret"
	if cfg.HasValue("mount") {
		if mount, err = cfg.GetValue("mount").Get(ctx); err != nil {
			return nil, fmt.Errorf("vault/openbao: mount: %w", err)
		}
	}

	conf := openbao.DefaultConfig()
	conf.Address = address
	client, err := openbao.NewClient(conf)
	if err != nil {
		return nil, fmt.Errorf("vault/openbao: client: %w", err)
	}
	client.SetToken(token)
	return &Vault{client: client, mount: mount}, nil
}

// Secret reads ref ("path#field") from the KV v2 engine and returns the field's
// string value.
func (v *Vault) Secret(ctx context.Context, ref string) (string, error) {
	path, field, ok := splitRef(ref)
	if !ok {
		return "", fmt.Errorf("vault/openbao: ref %q is not %q", ref, "path#field")
	}
	secret, err := v.client.KVv2(v.mount).Get(ctx, path)
	if err != nil {
		return "", fmt.Errorf("vault/openbao: read %q: %w", path, err)
	}
	raw, ok := secret.Data[field]
	if !ok {
		return "", fmt.Errorf("vault/openbao: secret %q has no field %q", path, field)
	}
	s, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("vault/openbao: field %q of %q is %T, want string", field, path, raw)
	}
	return s, nil
}

// splitRef splits "path#field" on the last '#'. Both sides must be non-empty.
func splitRef(ref string) (path, field string, ok bool) {
	i := strings.LastIndex(ref, "#")
	if i <= 0 || i == len(ref)-1 {
		return "", "", false
	}
	return ref[:i], ref[i+1:], true
}
