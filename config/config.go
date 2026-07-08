// package: config / composition
// type:    struct
// job:     decrypt/parse the launch config and either check it (Verify) or assemble the adapter stack (Run)
// limits:  the only component that sees the whole config; adapters get scope.Section slices (-> Verify, Run)
//
// Package config is the composition root. Its public surface is two entry points
// that mirror the CLI verbs: Verify checks an (optionally age-encrypted) config
// to a chosen depth without assembling anything; Run decrypts, parses, and
// assembles the live adapter stack. Both take the config bytes and a
// PassphraseSource the frontend supplies. config hands each adapter its own slice
// as a scope.Section — that section's keys and nothing else; env(KEY)/vault(ref)
// delegations resolve lazily when the adapter reads them, so a rotating secret is
// fetched at use. config is the ONE component that sees the whole config; a
// backend can read neither another port's secrets nor a sibling instance's — the
// narrowing is by containment, not visibility.
package config

import (
	"bytes"
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
// until they are read.
type Config struct {
	Signer    section   `json:"signer"`
	Endpoints []section `json:"endpoints"`
	Auth      []section `json:"auth"`
	Vault     section   `json:"vault"`
	Storage   section   `json:"storage"`
	Sequencer section   `json:"sequencer"`
}

// section is one adapter's raw config: keys to undecoded JSON values, resolved
// per value into a scope when the adapter reads them.
type section map[string]json.RawMessage

// App is the assembled adapter stack Run produces, in bootstrap order.
type App struct {
	Storage   storage.Storage
	Signer    signer.Signer
	Auth      []auth.Auth
	Sequencer sequencer.Sequencer
	Endpoints []endpoints.Endpoints
	// Vault is omitted on purpose: it is consumed during assembly to resolve
	// secrets, and nobody downstream holds it.
}

// Level is the depth of a Verify check.
type Level int

const (
	// LevelSyntax parses and shape-checks the config offline — no environment,
	// no vault, no assembly.
	LevelSyntax Level = iota
	// LevelResolve additionally resolves every env()/vault() reference, catching
	// an unset variable or a missing vault key without assembling any adapter.
	LevelResolve
	// LevelConnect additionally assembles the adapters — dialing storage, vault,
	// and the sequencer — then discards them without serving, so a bad backend
	// surfaces before Run does.
	LevelConnect
)

// Verify checks an (optionally age-encrypted) config to the given level. It
// decrypts with pass when the bytes are encrypted, then: LevelSyntax parses and
// shape-checks; LevelResolve also builds the vault and resolves every reference;
// LevelConnect also assembles the adapters — reaching every backend — and
// discards them without serving.
func Verify(ctx context.Context, cfg io.Reader, pass PassphraseSource, level Level) error {
	c, err := decode(cfg, pass)
	if err != nil {
		return err
	}
	if level < LevelResolve {
		return nil
	}
	v, err := c.buildVault(ctx)
	if err != nil {
		return err
	}
	if err := c.resolveAll(ctx, v); err != nil {
		return err
	}
	if level < LevelConnect {
		return nil
	}
	// Assembling the stack reaches every backend; discard it without serving.
	_, err = c.build(ctx, v)
	return err
}

// Run decrypts with pass when needed, parses, builds the vault, and assembles the
// adapter stack, returning the live App.
func Run(ctx context.Context, cfg io.Reader, pass PassphraseSource) (*App, error) {
	c, err := decode(cfg, pass)
	if err != nil {
		return nil, err
	}
	v, err := c.buildVault(ctx)
	if err != nil {
		return nil, err
	}
	return c.build(ctx, v)
}

// decode reads, decrypts (when encrypted), and parses the config — the shared
// front of Verify and Run. pass is consulted only if the bytes are encrypted.
func decode(cfg io.Reader, pass PassphraseSource) (*Config, error) {
	data, err := io.ReadAll(cfg)
	if err != nil {
		return nil, fmt.Errorf("config: read: %w", err)
	}
	plaintext, err := decrypt(data, pass)
	if err != nil {
		return nil, err
	}
	return load(bytes.NewReader(plaintext))
}

// load parses the JSON launch artifact without resolving delegations — the
// offline shape check. Unknown top-level sections are rejected.
func load(r io.Reader) (*Config, error) {
	var c Config
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("config: parse: %w", err)
	}
	return &c, nil
}

// build assembles the adapter stack bottom-up in dependency order: the leaf
// ports (storage, signer, auth) first, then the sequencer (which needs storage
// and the signing identity), then the endpoints (each of which needs the
// sequencer and the full set of authenticators). Each adapter is handed only its
// own section as a scope.Section, whose env()/vault() delegations resolve against
// v when the adapter reads them. An absent section yields a nil/empty adapter, so
// a partial config (storage-only) assembles just what it carries.
func (c *Config) build(ctx context.Context, v vault.Vault) (*App, error) {
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

// buildVault constructs the secret store from the vault section, resolved
// env-only — it cannot resolve vault() before the vault exists, so secret-zero
// collapses to an inline (encrypted) literal or an env() reference. It returns a
// nil Vault when no vault section is configured, in which case any vault()
// reference elsewhere fails loud when it is read.
func (c *Config) buildVault(ctx context.Context) (vault.Vault, error) {
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

// resolveAll resolves every env()/vault() reference in the config against v,
// assembling no adapter — the LevelResolve verification pass. It fails on the
// first unresolvable reference (an unset env var, a missing vault key).
func (c *Config) resolveAll(ctx context.Context, v vault.Vault) error {
	singles := []struct {
		name string
		sec  section
	}{
		{"signer", c.Signer},
		{"vault", c.Vault},
		{"storage", c.Storage},
		{"sequencer", c.Sequencer},
	}
	for _, s := range singles {
		if len(s.sec) == 0 {
			continue
		}
		if err := resolveSection(ctx, newSection(s.sec, v)); err != nil {
			return fmt.Errorf("config: %s: %w", s.name, err)
		}
	}
	lists := []struct {
		name string
		secs []section
	}{
		{"auth", c.Auth},
		{"endpoints", c.Endpoints},
	}
	for _, l := range lists {
		for i, sec := range l.secs {
			if err := resolveSection(ctx, newSection(sec, v)); err != nil {
				return fmt.Errorf("config: %s[%d]: %w", l.name, i, err)
			}
		}
	}
	return nil
}
