// package: scope / config
// type:    struct
// job:     a resolution-free Section over a flat map, for known values and tests
// limits:  flat leaves plus literal arrays; cfgSection resolves env()/vault() (-> config)
//
// This file provides Literal: a Section whose values are already-known literals, with
// no env()/vault() resolution. The config-driven handout (config.cfgSection) is
// unexported, so an adapter test — which per the architecture must drive its real
// counterpart through the port's factory, not construct it directly — uses Literal to
// build the section it hands its constructor. LiteralArray adds array-valued keys
// (a backend's "keys" list, an endpoint's "auth" backends) for the same reason: a
// factory a test cannot reach any other way.
package scope

import (
	"context"
	"fmt"
	"sort"
)

// Literal wraps a flat map of known values as a Section. Values resolve to
// themselves; there are no nested sections. Equivalent to LiteralArray(values, nil).
func Literal(values map[string]string) Section {
	return litSection{values: values}
}

// LiteralArray is Literal plus array-valued keys, each already built as Sections —
// typically nested Literal calls. Config arrays are always arrays of objects, matching
// GetArray's own contract.
func LiteralArray(values map[string]string, arrays map[string][]Section) Section {
	return litSection{values: values, arrays: arrays}
}

type litSection struct {
	values map[string]string
	arrays map[string][]Section
}

func (s litSection) GetSection(string) Section { return litSection{} }

func (s litSection) Get(_ context.Context, key string) (string, error) {
	v, ok := s.values[key]
	if !ok {
		return "", fmt.Errorf("scope: key %q is absent", key)
	}
	return v, nil
}

func (s litSection) GetArray(key string) []Section { return s.arrays[key] }

func (s litSection) HasValue(key string) bool { _, ok := s.values[key]; return ok }

func (s litSection) HasSection(string) bool { return false }

func (s litSection) HasArray(key string) bool { _, ok := s.arrays[key]; return ok }

func (s litSection) HasKey(key string) bool { return s.HasValue(key) || s.HasArray(key) }

func (s litSection) Keys() []string {
	keys := make([]string, 0, len(s.values)+len(s.arrays))
	for k := range s.values {
		keys = append(keys, k)
	}
	for k := range s.arrays {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
