// package: scope / config
// type:    struct
// job:     a resolution-free Section over a flat map, for known values and tests
// limits:  flat leaves only; cfgSection resolves env()/vault() (-> config)
//
// This file provides Literal: a Section whose values are already-known literals,
// with no env()/vault() resolution and no nested sections or arrays. The
// config-driven handout (config.cfgSection) is unexported, so an adapter test —
// which per the architecture must drive its real counterpart — uses Literal to
// build the section it hands its constructor.
package scope

import (
	"context"
	"fmt"
	"sort"
)

// Literal wraps a flat map of known values as a Section. Values resolve to
// themselves; there are no nested sections or arrays.
func Literal(values map[string]string) Section {
	return litSection(values)
}

type litSection map[string]string

func (s litSection) GetSection(string) Section { return litSection(nil) }

func (s litSection) Get(_ context.Context, key string) (string, error) {
	v, ok := s[key]
	if !ok {
		return "", fmt.Errorf("scope: key %q is absent", key)
	}
	return v, nil
}

func (s litSection) GetArray(string) []Section { return nil }

func (s litSection) HasValue(key string) bool { _, ok := s[key]; return ok }

func (s litSection) HasSection(string) bool { return false }

func (s litSection) HasArray(string) bool { return false }

func (s litSection) HasKey(key string) bool { _, ok := s[key]; return ok }

func (s litSection) Keys() []string {
	keys := make([]string, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
