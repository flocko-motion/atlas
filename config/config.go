// package: config / composition
// type:    struct
// job:     parse the launch config, slice each section into a scope.Section, and assemble the adapter stack
// limits:  the only component that sees the whole config; adapters get scope.Section slices (-> Build)
//
// Package config is the composition root. It parses the JSON launch artifact and
// hands each adapter instance its own slice as a scope.Section — that section's
// keys and nothing else. env(KEY)/vault(ref) delegations are not expanded
// eagerly; a section's leaves resolve lazily when the adapter reads them, so a
// rotating secret is fetched at use. config is the ONE component that sees the
// whole config; a backend can read neither another port's secrets nor a sibling
// instance's — the narrowing is by containment, not visibility.
package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/flocko-motion/rankedb/adapters/auth"
	"github.com/flocko-motion/rankedb/adapters/endpoints"
	"github.com/flocko-motion/rankedb/adapters/sequencer"
	"github.com/flocko-motion/rankedb/adapters/signer"
	"github.com/flocko-motion/rankedb/adapters/storage"
	"github.com/flocko-motion/rankedb/adapters/vault"
)

// Config is the parsed launch artifact. Each section is an open key/value map
// (backend-specific keys), so the schema fixes only which ports exist, not
// which settings each backend takes. Values may be env()/vault() placeholders
// until Build resolves them.
type Config struct {
	Signer    section   `json:"signer"`
	Endpoints []section `json:"endpoints"`
	Auth      []section `json:"auth"`
	Vault     section   `json:"vault"`
	Storage   section   `json:"storage"`
	Sequencer section   `json:"sequencer"`
}

// section is one adapter's raw config: keys to undecoded JSON values, resolved
// per value into a scope when the adapter is built.
type section map[string]json.RawMessage

// App is the assembled adapter stack Build produces, in bootstrap order.
type App struct {
	Storage   storage.Storage
	Signer    signer.Signer
	Auth      []auth.Auth
	Sequencer sequencer.Sequencer
	Endpoints []endpoints.Endpoints
	// Vault is omitted on purpose: it is consumed during assembly to resolve
	// secrets, and nobody downstream holds it.
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

// Build assembles the adapter stack bottom-up in dependency order: the leaf
// ports (storage, signer, auth) first, then the sequencer (which needs storage
// and the signing identity), then the endpoints (each of which needs the
// sequencer and the full set of authenticators). Each adapter is handed only its
// own section as a scope.Section, whose env()/vault() delegations resolve against
// v when the adapter reads them. An absent section yields a nil/empty adapter, so
// a partial config (storage-only, verify-only) assembles just what it carries.
func (c *Config) Build(ctx context.Context, v vault.Vault) (*App, error) {
	var app App

	if len(c.Storage) > 0 {
		st, err := storage.New(ctx, newSection(c.Storage, v))
		if err != nil {
			return nil, err
		}
		app.Storage = st
	}

	if len(c.Signer) > 0 {
		sg, err := signer.New(ctx, newSection(c.Signer, v))
		if err != nil {
			return nil, err
		}
		app.Signer = sg
	}

	for i, a := range c.Auth {
		au, err := auth.New(ctx, newSection(a, v))
		if err != nil {
			return nil, fmt.Errorf("config: auth[%d]: %w", i, err)
		}
		app.Auth = append(app.Auth, au)
	}

	if len(c.Sequencer) > 0 {
		seq, err := sequencer.New(ctx, newSection(c.Sequencer, v), app.Storage, app.Signer)
		if err != nil {
			return nil, err
		}
		app.Sequencer = seq
	}

	for i, e := range c.Endpoints {
		ep, err := endpoints.New(ctx, newSection(e, v), app.Sequencer, app.Auth)
		if err != nil {
			return nil, fmt.Errorf("config: endpoints[%d]: %w", i, err)
		}
		app.Endpoints = append(app.Endpoints, ep)
	}

	return &app, nil
}

// BuildVault constructs the secret store from the vault section, resolved
// env-only — it cannot resolve vault() before the vault exists, so secret-zero
// collapses to an inline (encrypted) literal or an env() reference. It returns a
// nil Vault when no vault section is configured, in which case any vault()
// reference elsewhere fails loud when it is read.
func (c *Config) BuildVault(ctx context.Context) (vault.Vault, error) {
	if len(c.Vault) == 0 {
		return nil, nil
	}
	sec := newSection(c.Vault, nil)
	if !sec.HasValue("type") {
		return nil, fmt.Errorf("config: vault section present but no type set")
	}
	t, err := sec.GetValue("type").Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("config: vault: %w", err)
	}
	return nil, fmt.Errorf("config: vault backend %q not yet implemented", t)
}
