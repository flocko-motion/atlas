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
	"github.com/flocko-motion/rankedb/internal/core"
	"github.com/flocko-motion/rankedb/internal/core/access"
)

// Config is the parsed launch artifact. The driven ports (signer, storage,
// sequencer, vault) and the account roster are shared by the whole instance —
// there is one archive and one signing identity. Authentication and admission are
// per endpoint: each endpoint block carries its own transport, its own auth
// backends, and the set of accounts it admits. Each adapter section is an open
// key/value map, so the schema fixes only which ports exist, not which settings
// each backend takes; values may be env()/vault() placeholders until read.
type Config struct {
	Signer    section                  `json:"signer"`
	Vault     section                  `json:"vault"`
	Storage   section                  `json:"storage"`
	Sequencer section                  `json:"sequencer"`
	Accounts  map[string]accountConfig `json:"accounts"`
	Endpoints []endpointConfig         `json:"endpoints"`

	// box is the shared secret-store holder, seeded from Vault at parse. It is
	// assembled state, not JSON (unexported), and the only place a section reaches
	// the vault — built lazily on first vault() use.
	box *vaultBox
}

// accountConfig is one system account: its CRUD grants over branch globs. It
// carries no credential material — how a request authenticates AS this account is
// an endpoint's auth backend (which maps a token to this name); accounts hold only
// authorization. Grants carry no env()/vault() delegation — pure policy, validated
// at parse.
type accountConfig struct {
	Grants []string `json:"grants"`
}

// endpointConfig is one endpoint: a transport, the auth backends it accepts, and
// the admit list of account names it serves. Admission is the isolation boundary —
// an account absent from admit is unreachable here, even if some backend minted
// its name, because this endpoint's access checker is built from the admitted
// subset only.
type endpointConfig struct {
	Transport section   `json:"transport"`
	Auth      []section `json:"auth"`
	Admit     []string  `json:"admit"`
}

// section is one adapter's raw config: keys to undecoded JSON values, resolved
// per value into a scope when the adapter reads them.
type section map[string]json.RawMessage

// App is the assembled stack Run produces: the shared driven ports and the
// endpoints, each of which internally holds its own core (auth + admitted-account
// access + the shared driven ports).
type App struct {
	Storage   storage.Storage
	Signer    signer.Signer
	Sequencer sequencer.Sequencer
	Endpoints []endpoints.Endpoints
	// The secret store is omitted on purpose: it lives in the config's section box,
	// built lazily to resolve vault() references, and nobody downstream holds it.
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
// shape-checks; LevelResolve resolves every env()/vault() reference (building the
// vault lazily, only if referenced); LevelConnect also assembles the adapters —
// reaching every backend actually used — and discards them without serving.
func Verify(ctx context.Context, cfg io.Reader, pass PassphraseSource, level Level) error {
	c, err := decode(cfg, pass)
	if err != nil {
		return err
	}
	if level < LevelResolve {
		return nil
	}
	if err := c.resolveAll(ctx); err != nil {
		return err
	}
	if level < LevelConnect {
		return nil
	}
	// Assembling the stack reaches every backend it uses; discard it without serving.
	_, err = c.build(ctx)
	return err
}

// Run decrypts with pass when needed, parses, and assembles the adapter stack,
// returning the live App. The vault builds lazily as sections resolve vault()
// references during assembly.
func Run(ctx context.Context, cfg io.Reader, pass PassphraseSource) (*App, error) {
	c, err := decode(cfg, pass)
	if err != nil {
		return nil, err
	}
	return c.build(ctx)
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
	// Accounts are pure policy: validate every account's grants now, offline, so a
	// malformed grant fails the syntax check rather than waiting for assembly.
	if _, err := access.New(c.accountSpecs()); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	// Referential integrity, also offline: every admitted name must be a defined
	// account, so a typo in an admit list fails syntax rather than silently
	// admitting no one.
	for i, ec := range c.Endpoints {
		for _, name := range ec.Admit {
			if _, ok := c.Accounts[name]; !ok {
				return nil, fmt.Errorf("config: endpoints[%d]: admit names undefined account %q", i, name)
			}
		}
	}
	// Seed the shared secret-store holder from the vault section; it builds the
	// store lazily, only if some section resolves a vault() reference.
	c.box = &vaultBox{cfg: c.Vault}
	return &c, nil
}

// accountSpecs flattens the account roster to the name→grant-specs map the access
// checker consumes.
func (c *Config) accountSpecs() map[string][]string {
	specs := make(map[string][]string, len(c.Accounts))
	for name, ac := range c.Accounts {
		specs[name] = ac.Grants
	}
	return specs
}

// build assembles the stack in dependency order: the shared driven ports first
// (storage, signer, then the sequencer that needs both), then each endpoint —
// which gets its own core composing its auth backends and its admitted-account
// access checker over the shared driven ports. Each adapter is handed only its
// own section as a scope.Section, whose env()/vault() delegations resolve lazily
// when read (the vault builds on the first vault() reference). An absent section
// yields a nil/empty adapter, so a partial config (storage-only) assembles just
// what it carries.
func (c *Config) build(ctx context.Context) (*App, error) {
	var app App

	if len(c.Storage) > 0 {
		st, err := storage.New(ctx, c.section(c.Storage))
		if err != nil {
			return nil, err
		}
		app.Storage = st
	}

	if len(c.Signer) > 0 {
		sg, err := signer.New(ctx, c.section(c.Signer))
		if err != nil {
			return nil, err
		}
		app.Signer = sg
	}

	if len(c.Sequencer) > 0 {
		seq, err := sequencer.New(ctx, c.section(c.Sequencer), app.Storage, app.Signer)
		if err != nil {
			return nil, err
		}
		app.Sequencer = seq
	}

	for i, ec := range c.Endpoints {
		cr, err := c.buildEndpoint(ctx, ec, app.Storage, app.Sequencer)
		if err != nil {
			return nil, fmt.Errorf("config: endpoints[%d]: %w", i, err)
		}
		ep, err := endpoints.New(ctx, c.section(ec.Transport), cr)
		if err != nil {
			return nil, fmt.Errorf("config: endpoints[%d]: %w", i, err)
		}
		app.Endpoints = append(app.Endpoints, ep)
	}

	return &app, nil
}

// buildEndpoint composes one endpoint's core: its auth backends indexed for scheme
// dispatch, and an access checker built from only the accounts it admits (so an
// un-admitted account is simply absent and denied), over the shared driven ports.
func (c *Config) buildEndpoint(ctx context.Context, ec endpointConfig, store storage.Storage, seq sequencer.Sequencer) (*core.Core, error) {
	var auths []auth.Auth
	for j, a := range ec.Auth {
		au, err := auth.New(ctx, c.section(a))
		if err != nil {
			return nil, fmt.Errorf("auth[%d]: %w", j, err)
		}
		auths = append(auths, au)
	}
	set, err := auth.NewSet(auths)
	if err != nil {
		return nil, err
	}

	admitted := make(map[string][]string, len(ec.Admit))
	for _, name := range ec.Admit {
		admitted[name] = c.Accounts[name].Grants // presence guaranteed by load's admit check
	}
	chk, err := access.New(admitted)
	if err != nil {
		return nil, err
	}

	return core.New(set, chk, seq, store), nil
}

// resolveAll resolves every env()/vault() reference in the config, assembling no
// adapter — the LevelResolve verification pass. It fails on the first unresolvable
// reference (an unset env var, a missing vault key). The vault section is not
// resolved on its own: it is dialed only if some other section references vault(),
// so resolve checks only what is actually used.
func (c *Config) resolveAll(ctx context.Context) error {
	singles := []struct {
		name string
		sec  section
	}{
		{"signer", c.Signer},
		{"storage", c.Storage},
		{"sequencer", c.Sequencer},
	}
	for _, s := range singles {
		if len(s.sec) == 0 {
			continue
		}
		if err := resolveSection(ctx, c.section(s.sec)); err != nil {
			return fmt.Errorf("config: %s: %w", s.name, err)
		}
	}
	// Each endpoint's transport and auth backends carry delegations; the account
	// roster and admit lists are pure policy with none.
	for i, ec := range c.Endpoints {
		if len(ec.Transport) > 0 {
			if err := resolveSection(ctx, c.section(ec.Transport)); err != nil {
				return fmt.Errorf("config: endpoints[%d].transport: %w", i, err)
			}
		}
		for j, a := range ec.Auth {
			if err := resolveSection(ctx, c.section(a)); err != nil {
				return fmt.Errorf("config: endpoints[%d].auth[%d]: %w", i, j, err)
			}
		}
	}
	return nil
}
