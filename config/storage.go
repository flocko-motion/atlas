// package: config / composition
// type:    func
// job:     parse the storage section's layer list and resolve each leaf into a storage.Spec
// limits:  shape + resolution only; composition into a Universe is storage's (-> adapters/storage)
//
// This file handles the multi-instance storage port. Unlike the single-instance
// ports (one signer, one auth), storage is an ordered list of layers, each its
// own adapter instance with its own connection details. The composition root
// walks the list, slices each layer's settings into that instance's scope,
// resolves env()/vault() delegations, parses the human-readable size cap, and
// recurses into partition shards — handing storage.New a tree of resolved specs
// in which no leaf can see another's secrets.
package config

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/flocko-motion/rankedb/adapters/storage"
	"github.com/flocko-motion/rankedb/adapters/vault"
)

// layerKeys are the structural keys handled here; everything else in a layer
// object is a backend setting resolved into the instance scope.
var layerKeys = map[string]bool{
	"mode": true, "type": true, "maxContentSize": true, "noReadFill": true, "shards": true,
}

// rawLayer is a storage layer as parsed: its structural fields plus the open
// remainder, which becomes the instance scope.
type rawLayer struct {
	Mode           string
	Type           string
	MaxContentSize string
	NoReadFill     bool
	Shards         []rawLayer
	Rest           json.RawMessage // backend settings: every non-structural key
}

// UnmarshalJSON splits a layer object into its structural fields and the
// backend-setting remainder. The structural keys (mode, type, maxContentSize,
// noReadFill, shards) are decoded into fields; every other key is collected
// into Rest as an object, which becomes the layer's instance scope.
func (l *rawLayer) UnmarshalJSON(data []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	for key, raw := range m {
		var err error
		switch key {
		case "mode":
			err = json.Unmarshal(raw, &l.Mode)
		case "type":
			err = json.Unmarshal(raw, &l.Type)
		case "maxContentSize":
			// Accept a string ("8kb") or a bare JSON number (8192).
			if e := json.Unmarshal(raw, &l.MaxContentSize); e != nil {
				l.MaxContentSize = string(raw)
			}
		case "noReadFill":
			err = json.Unmarshal(raw, &l.NoReadFill)
		case "shards":
			err = json.Unmarshal(raw, &l.Shards)
		}
		if err != nil {
			return fmt.Errorf("storage layer key %q: %w", key, err)
		}
	}
	rest := make(map[string]json.RawMessage, len(m))
	for key, raw := range m {
		if !layerKeys[key] {
			rest[key] = raw
		}
	}
	if len(rest) > 0 {
		b, err := json.Marshal(rest)
		if err != nil {
			return err
		}
		l.Rest = b
	}
	return nil
}

// buildStorageSpecs resolves the storage section into storage.Spec specs ready
// for composition. Returns nil when no storage is configured (the caller then
// reports the stack has no storage).
func (c *Config) buildStorageSpecs(ctx context.Context, v vault.Vault) ([]storage.Spec, error) {
	if len(c.Storage) == 0 {
		return nil, nil
	}
	return c.resolveLayers(ctx, c.Storage, v)
}

// resolveLayers turns parsed layers into resolved specs, recursing into shards.
func (c *Config) resolveLayers(ctx context.Context, layers []rawLayer, v vault.Vault) ([]storage.Spec, error) {
	out := make([]storage.Spec, 0, len(layers))
	for i, l := range layers {
		sc, err := c.scopeFromRaw(ctx, fmt.Sprintf("storage[%d]", i), l.Rest, v)
		if err != nil {
			return nil, err
		}
		size, err := parseSize(l.MaxContentSize)
		if err != nil {
			return nil, fmt.Errorf("config: storage[%d].maxContentSize: %w", i, err)
		}
		spec := storage.Spec{
			Mode:           l.Mode,
			Type:           l.Type,
			Scope:          sc,
			MaxContentSize: size,
			NoReadFill:     l.NoReadFill,
		}
		if len(l.Shards) > 0 {
			spec.Shards, err = c.resolveLayers(ctx, l.Shards, v)
			if err != nil {
				return nil, err
			}
		}
		out = append(out, spec)
	}
	return out, nil
}

// parseSize parses a human-readable byte size: a bare number is bytes, or a
// number with a kb/mb/gb/tb suffix (decimal, case-insensitive, optional "b").
// An empty string is 0 (uncapped).
func parseSize(s string) (uint64, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, nil
	}
	mult := uint64(1)
	matched := false
	for suffix, m := range map[string]uint64{"tb": 1 << 40, "gb": 1 << 30, "mb": 1 << 20, "kb": 1 << 10} {
		if strings.HasSuffix(s, suffix) {
			mult = m
			s = strings.TrimSpace(strings.TrimSuffix(s, suffix))
			matched = true
			break
		}
	}
	if !matched {
		s = strings.TrimSpace(strings.TrimSuffix(s, "b")) // bare bytes, e.g. "4096b"
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size %q", s)
	}
	return n * mult, nil
}
