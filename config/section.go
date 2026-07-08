// package: config / composition
// type:    struct
// job:     the concrete scope.Section / scope.Value handed to each adapter — a lazily-resolved slice of the parsed config
// limits:  navigates one parsed JSON object and resolves its leaves on demand; holds no path to the rest of the config (-> config, config/scope)
//
// This file is the handout itself: the concrete types that fulfil the scope
// contract. cfgSection wraps one parsed JSON object plus the vault its leaves may
// reference; it navigates into nested sections and yields leaf Values, and holds
// no back-reference to the enclosing config — the containment that keeps one
// adapter from reading another's slice. cfgValue resolves a single leaf lazily on
// Get: a literal passes through, an env()/vault() delegation is expanded only
// then, so a rotating secret is fetched at use and the source stays opaque to the
// adapter.
package config

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/flocko-motion/rankedb/adapters/vault"
	"github.com/flocko-motion/rankedb/config/scope"
)

// cfgSection is the concrete scope.Section over one parsed JSON object.
type cfgSection struct {
	raw   map[string]json.RawMessage
	vault vault.Vault
}

// newSection wraps a parsed JSON object as a scope.Section whose leaves resolve
// their env()/vault() delegations against v.
func newSection(raw map[string]json.RawMessage, v vault.Vault) scope.Section {
	return cfgSection{raw: raw, vault: v}
}

// GetSection descends into a nested object. A missing or non-object key yields an
// empty section, so navigation never returns nil.
func (s cfgSection) GetSection(key string) scope.Section {
	empty := cfgSection{raw: map[string]json.RawMessage{}, vault: s.vault}
	raw, ok := s.raw[key]
	if !ok || !isObject(raw) {
		return empty
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return empty
	}
	return cfgSection{raw: m, vault: s.vault}
}

// GetValue returns the leaf at key. An absent key yields a Value that reports the
// absence when resolved.
func (s cfgSection) GetValue(key string) scope.Value {
	raw, ok := s.raw[key]
	return cfgValue{key: key, raw: raw, present: ok, vault: s.vault}
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
		out = append(out, cfgSection{raw: m, vault: s.vault})
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

// cfgValue is the concrete scope.Value: one leaf resolved lazily on Get.
type cfgValue struct {
	key     string
	raw     json.RawMessage
	present bool
	vault   vault.Vault
}

// Get resolves the leaf. An absent key errors; a non-string leaf (number, bool)
// carries its literal JSON text; a string leaf is run through env()/vault()
// expansion, which reaches the environment or vault only now, at use.
func (v cfgValue) Get(ctx context.Context) (string, error) {
	if !v.present {
		return "", fmt.Errorf("config: key %q is absent", v.key)
	}
	var s string
	if err := json.Unmarshal(v.raw, &s); err != nil {
		return string(v.raw), nil
	}
	return resolveValue(ctx, s, v.vault)
}

// resolveSection resolves every leaf under sec (recursing through nested
// sections and arrays), discarding the values — a pass that surfaces any
// unresolvable env()/vault() reference. Verify uses it at LevelResolve.
func resolveSection(ctx context.Context, sec scope.Section) error {
	for _, k := range sec.Keys() {
		switch {
		case sec.HasValue(k):
			if _, err := sec.GetValue(k).Get(ctx); err != nil {
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
