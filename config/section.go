// package: config / composition
// type:    struct
// job:     the concrete scope.Section handed to each adapter — a lazily-resolved slice of the parsed config
// limits:  navigates one parsed JSON object and resolves its leaves on demand; holds no path to the rest of the config (-> config, config/scope)
//
// This file is the handout itself: the concrete type that fulfils the scope
// contract. cfgSection wraps one parsed JSON object plus the vault box its leaves
// may reference; it navigates into nested sections and resolves leaves via Get,
// and holds no back-reference to the enclosing config — the containment that keeps
// one adapter from reading another's slice. A leaf is resolved lazily on Get: a
// literal passes through, an env()/vault() delegation is expanded only then, so a
// rotating secret is fetched at use and the source stays opaque to the adapter.
package config

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/flocko-motion/rankedb/adapters/vault"
	"github.com/flocko-motion/rankedb/config/scope"
)

// vaultBox owns the secret store's lifecycle for one config. It builds the store
// on the first vault() reference resolved and caches it, so a config that never
// references vault() never dials one — and a LevelConnect check reaches only the
// backends actually used. It is shared by every section derived from the config
// (propagated on navigation), so the store is built at most once. The vault
// section itself gets a nil box, since a vault cannot resolve its own secrets.
type vaultBox struct {
	cfg  map[string]json.RawMessage
	once sync.Once
	v    vault.Vault
	err  error
}

// get returns the secret store, building it once from the vault section (resolved
// env-only, via a boxless section). A nil box — the vault section's own leaves —
// rejects vault() outright.
func (b *vaultBox) get(ctx context.Context) (vault.Vault, error) {
	if b == nil {
		return nil, errors.New("vault() is not resolvable here")
	}
	b.once.Do(func() {
		if len(b.cfg) == 0 {
			b.err = errors.New("referenced but no vault section is configured")
			return
		}
		b.v, b.err = vault.New(ctx, cfgSection{raw: b.cfg})
	})
	return b.v, b.err
}

// section wraps one of the config's raw objects as a scope.Section bound to the
// config's shared vault box — the single place a section acquires its box, so no
// builder threads the vault around.
func (c *Config) section(raw section) scope.Section {
	return newSection(raw, c.box)
}

// cfgSection is the concrete scope.Section over one parsed JSON object.
type cfgSection struct {
	raw map[string]json.RawMessage
	box *vaultBox
}

// newSection wraps a parsed JSON object as a scope.Section whose leaves resolve
// their vault() delegations through box (nil box → vault() is unresolvable).
func newSection(raw map[string]json.RawMessage, box *vaultBox) scope.Section {
	return cfgSection{raw: raw, box: box}
}

// GetSection descends into a nested object. A missing or non-object key yields an
// empty section, so navigation never returns nil.
func (s cfgSection) GetSection(key string) scope.Section {
	empty := cfgSection{raw: map[string]json.RawMessage{}, box: s.box}
	raw, ok := s.raw[key]
	if !ok || !isObject(raw) {
		return empty
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return empty
	}
	return cfgSection{raw: m, box: s.box}
}

// Get resolves the leaf at key. An absent key errors; a non-string leaf (number,
// bool) yields its literal JSON text; a string leaf is run through env()/vault()
// expansion, which reaches the environment or vault only now, at use.
func (s cfgSection) Get(ctx context.Context, key string) (string, error) {
	raw, ok := s.raw[key]
	if !ok {
		return "", fmt.Errorf("config: key %q is absent", key)
	}
	var str string
	if err := json.Unmarshal(raw, &str); err != nil {
		return string(raw), nil
	}
	return resolveValue(ctx, str, s.box)
}

// GetArray returns the elements of an array-valued key, each wrapped as a
// Section (config arrays are arrays of objects). A missing key or a non-array
// value yields nil.
func (s cfgSection) GetArray(key string) []scope.Section {
	raw, ok := s.raw[key]
	if !ok || !isArray(raw) {
		return nil
	}
	var elems []json.RawMessage
	if err := json.Unmarshal(raw, &elems); err != nil {
		return nil
	}
	out := make([]scope.Section, 0, len(elems))
	for _, e := range elems {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(e, &m); err != nil {
			m = map[string]json.RawMessage{}
		}
		out = append(out, cfgSection{raw: m, box: s.box})
	}
	return out
}

// HasValue reports whether key is present as a leaf value — not a nested object
// and not an array.
func (s cfgSection) HasValue(key string) bool {
	raw, ok := s.raw[key]
	return ok && !isObject(raw) && !isArray(raw)
}

// HasSection reports whether key is present as a nested object.
func (s cfgSection) HasSection(key string) bool {
	raw, ok := s.raw[key]
	return ok && isObject(raw)
}

// HasArray reports whether key is present as an array.
func (s cfgSection) HasArray(key string) bool {
	raw, ok := s.raw[key]
	return ok && isArray(raw)
}

// HasKey reports whether key is present at all.
func (s cfgSection) HasKey(key string) bool {
	_, ok := s.raw[key]
	return ok
}

// Keys lists the present keys, sorted for deterministic iteration.
func (s cfgSection) Keys() []string {
	keys := make([]string, 0, len(s.raw))
	for k := range s.raw {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// resolveSection resolves every leaf under sec (recursing through nested
// sections and arrays), discarding the values — a pass that surfaces any
// unresolvable env()/vault() reference. Verify uses it at LevelResolve.
func resolveSection(ctx context.Context, sec scope.Section) error {
	for _, k := range sec.Keys() {
		switch {
		case sec.HasValue(k):
			if _, err := sec.Get(ctx, k); err != nil {
				return err
			}
		case sec.HasSection(k):
			if err := resolveSection(ctx, sec.GetSection(k)); err != nil {
				return err
			}
		case sec.HasArray(k):
			for _, e := range sec.GetArray(k) {
				if err := resolveSection(ctx, e); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// isObject reports whether raw is a JSON object, distinguishing a nested section
// from a leaf value.
func isObject(raw json.RawMessage) bool {
	t := bytes.TrimSpace(raw)
	return len(t) > 0 && t[0] == '{'
}

// isArray reports whether raw is a JSON array.
func isArray(raw json.RawMessage) bool {
	t := bytes.TrimSpace(raw)
	return len(t) > 0 && t[0] == '['
}
