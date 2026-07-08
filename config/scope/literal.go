// package: scope / config
// type:    struct
// job:     a literal, resolution-free Section over a flat map — for adapters holding known values and their tests
// limits:  flat leaves only (no nesting, no env()/vault()); the config-driven cfgSection does resolution (-> config)
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

func (s litSection) GetValue(key string) Value {
	v, ok := s[key]
	return litValue{key: key, val: v, present: ok}
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

type litValue struct {
	key     string
	val     string
	present bool
}

func (v litValue) Get(context.Context) (string, error) {
	if !v.present {
		return "", fmt.Errorf("scope: key %q is absent", v.key)
	}
	return v.val, nil
}
