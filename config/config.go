// package: config / composition
// type:    struct
// job:     parse the launch config, slice each section into a resolved instance scope, and build the adapter stack
// limits:  the only component that sees the whole config; adapters get scope.Config slices (-> Build)
//
// Package config is the composition root. It parses the JSON launch artifact,
// and for each adapter instance slices out that instance's section, resolves
// every env(KEY)/vault(ref) delegation to plaintext, and hands the adapter a
// scope.Config holding only those values. It is the ONE component that sees the
// whole config; each adapter receives only its slice, so a backend can read
// neither another port's secrets nor a sibling instance's — the narrowing is by
// containment, not visibility.
package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/flocko-motion/ranke-go"

	"github.com/flocko-motion/rankedb/adapters/auth"
	"github.com/flocko-motion/rankedb/adapters/signer"
	"github.com/flocko-motion/rankedb/adapters/storage"
	"github.com/flocko-motion/rankedb/adapters/vault"
	"github.com/flocko-motion/rankedb/scope"
)

// Config is the parsed launch artifact. Each section is an open key/value map
// (backend-specific keys), so the schema fixes only which ports exist, not
// which settings each backend takes. Values may be env()/vault() placeholders
// until Build resolves them.
type Config struct {
	Signer  section    `json:"signer"`
	Auth    section    `json:"auth"`
	Vault   section    `json:"vault"`
	Storage []rawLayer `json:"storage"`
}

// section is one adapter's raw config: keys to undecoded JSON values, resolved
// per value into a scope when the adapter is built.
type section map[string]json.RawMessage

// App is the assembled adapter stack Build produces.
type App struct {
	Signer  signer.Signer
	Auth    auth.Auth
	Storage ranke.Universe
}

// Load parses the JSON launch artifact from r without resolving delegations —
// the offline shape check used by the verify command. Unknown top-level
// sections are rejected. Call Build to resolve secrets and assemble adapters.
func Load(r io.Reader) (*Config, error) {
	var c Config
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("config: parse: %w", err)
	}
	return &c, nil
}

// Build resolves each section's delegations against v (nil = env-only; a
// vault() reference then errors) and assembles the adapter stack, handing each
// adapter only its own resolved scope. It assembles only the ports a config
// actually carries; an absent section yields a nil adapter. Policy — which
// ports a serving instance requires — is the caller's (see cmd/ranke-db run),
// so a storage-only or verify-only tool can assemble just what it needs.
func (c *Config) Build(ctx context.Context, v vault.Vault) (*App, error) {
	var app App
	if len(c.Signer) > 0 {
		sc, err := c.scopeOf(ctx, "signer", c.Signer, v)
		if err != nil {
			return nil, err
		}
		if app.Signer, err = signer.New(sc); err != nil {
			return nil, err
		}
	}
	if len(c.Auth) > 0 {
		sc, err := c.scopeOf(ctx, "auth", c.Auth, v)
		if err != nil {
			return nil, err
		}
		if app.Auth, err = auth.New(sc); err != nil {
			return nil, err
		}
	}
	specs, err := c.buildStorageSpecs(ctx, v)
	if err != nil {
		return nil, err
	}
	if specs != nil {
		if app.Storage, err = storage.New(specs); err != nil {
			return nil, err
		}
	}
	return &app, nil
}

// BuildVault constructs the secret store from the vault section. That section
// is resolved env-only — it cannot resolve vault() before the vault exists, so
// secret-zero collapses to an inline (encrypted) literal or an env() reference.
// It returns a nil Vault when no vault section is configured, in which case any
// vault() reference elsewhere fails loud at resolution.
func (c *Config) BuildVault(ctx context.Context) (vault.Vault, error) {
	if len(c.Vault) == 0 {
		return nil, nil
	}
	vs, err := c.scopeOf(ctx, "vault", c.Vault, nil)
	if err != nil {
		return nil, err
	}
	switch t := vs.String("type"); t {
	case "":
		return nil, fmt.Errorf("config: vault section present but no type set")
	default:
		return nil, fmt.Errorf("config: vault backend %q not yet implemented", t)
	}
}

// scopeOf resolves one section into an instance scope.
func (c *Config) scopeOf(ctx context.Context, name string, sec section, v vault.Vault) (scope.Config, error) {
	return c.resolveMap(ctx, name, sec, v)
}

// scopeFromRaw resolves a layer's remaining (non-structural) keys, given as a
// JSON object, into an instance scope. An empty object yields an empty scope.
func (c *Config) scopeFromRaw(ctx context.Context, name string, raw json.RawMessage, v vault.Vault) (scope.Config, error) {
	if len(raw) == 0 {
		return scope.New(map[string]string{}), nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return scope.Config{}, fmt.Errorf("config: %s: %w", name, err)
	}
	return c.resolveMap(ctx, name, m, v)
}

// resolveMap resolves a key/value section into an instance scope. A JSON string
// value is run through env()/vault() expansion; a non-string (bool, number) is
// carried as its literal text so the adapter can parse it. name labels
// resolution errors with the owning section.
func (c *Config) resolveMap(ctx context.Context, name string, m map[string]json.RawMessage, v vault.Vault) (scope.Config, error) {
	out := make(map[string]string, len(m))
	for key, raw := range m {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			out[key] = string(raw)
			continue
		}
		resolved, err := resolveValue(ctx, s, v)
		if err != nil {
			return scope.Config{}, fmt.Errorf("config: %s.%s: %w", name, key, err)
		}
		out[key] = resolved
	}
	return scope.New(out), nil
}
