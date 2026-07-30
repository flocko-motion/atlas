// package: storage / composition
// type:    factory
// job:     build one ranke.Universe from a storage section — a leaf, or a composite
// limits:  wiring only; the persistence logic is ranke-go's adapters (-> github.com/flocko-motion/ranke-go)
//
// Package storage is ranke-db's storage port. A composed stack or partition is
// itself a Universe (the paper's composable-universe property), so the whole
// store — however deeply layered — is one recursive descriptor and presents as
// one type. New reads that descriptor from its scope.Section: a "type" selects a
// leaf backend (mem, fs, sqlite, minimal, s3) or a composite ("stack" over its
// "layers", "partition" over its "shards"), and composites recurse, so a layer
// or a shard may itself be a stack or a partition. Each leaf reads its own
// settings from its section, resolving env()/vault() delegations as it does.
//
// A layer's write role and content cap are not ours to set: ranke-go reports
// them per universe as ranke.Capabilities, fixed by the backend's own adapter
// options, so this package only holds a layer's declared "mode"/"maxContentSize"
// against what the built universe reports and fails when they disagree.
package storage

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/flocko-motion/ranke-go"
	"github.com/flocko-motion/ranke-go/adapter/storage/fs"
	"github.com/flocko-motion/ranke-go/adapter/storage/mem"
	"github.com/flocko-motion/ranke-go/adapter/storage/minimal"
	"github.com/flocko-motion/ranke-go/adapter/storage/partition"
	"github.com/flocko-motion/ranke-go/adapter/storage/sqlite"
	"github.com/flocko-motion/ranke-go/adapter/storage/stack"

	"github.com/flocko-motion/rankedb/config/scope"
)

// Storage is the storage port's product: a ranke.Universe. A composed stack or partition
// implements Universe too, so the whole store presents as this one type.
type Storage = ranke.Universe

// New builds the storage universe described by sec (see the package doc for the
// descriptor shape).
func New(ctx context.Context, sec scope.Section) (Storage, error) {
	return build(ctx, sec)
}

// Layer is what introspection may report of a configured layer: a name and a type.
type Layer struct {
	Name string
	Type string
}

// Describe reports the configured layers top to bottom, touching no backend. A layer
// takes its optional "name" key, else its type.
func Describe(ctx context.Context, sec scope.Section) ([]Layer, error) {
	t, err := typeOf(ctx, sec)
	if err != nil {
		return nil, err
	}
	descriptors := []scope.Section{sec}
	switch t {
	case "stack":
		descriptors = sec.GetArray("layers")
	case "partition":
		descriptors = sec.GetArray("shards")
	}

	layers := make([]Layer, 0, len(descriptors))
	for _, d := range descriptors {
		lt, err := typeOf(ctx, d)
		if err != nil {
			return nil, err
		}
		name := lt
		if d.HasValue("name") {
			if name, err = d.Get(ctx, "name"); err != nil {
				return nil, fmt.Errorf("storage: layer name: %w", err)
			}
		}
		layers = append(layers, Layer{Name: name, Type: lt})
	}
	return distinct(layers), nil
}

// distinct suffixes a repeated name with its position, so every name identifies one layer.
func distinct(layers []Layer) []Layer {
	seen := make(map[string]int, len(layers))
	for _, l := range layers {
		seen[l.Name]++
	}
	for i, l := range layers {
		if seen[l.Name] > 1 {
			layers[i].Name = fmt.Sprintf("%s-%d", l.Name, i)
		}
	}
	return layers
}

// build dispatches one descriptor to a leaf backend or a composite, recursing.
func build(ctx context.Context, sec scope.Section) (ranke.Universe, error) {
	t, err := typeOf(ctx, sec)
	if err != nil {
		return nil, err
	}
	switch t {
	case "stack":
		return buildStack(ctx, sec.GetArray("layers"))
	case "partition":
		return buildPartition(ctx, sec.GetArray("shards"))
	case "fs":
		dir, err := sec.Get(ctx, "dir")
		if err != nil {
			return nil, err
		}
		return fs.New(dir)
	case "sqlite":
		dsn, err := sec.Get(ctx, "dsn")
		if err != nil {
			return nil, err
		}
		return sqlite.New(dsn)
	case "mem":
		return mem.New(), nil
	case "minimal":
		return minimal.New(), nil
	case "s3":
		// ranke-go's s3 adapter takes a constructed *s3.Client; wiring the AWS
		// SDK (region/endpoint/credentials from the section) is a dedicated pass.
		return nil, fmt.Errorf("storage: s3 backend not yet wired")
	case "":
		return nil, fmt.Errorf("storage: missing type")
	default:
		return nil, fmt.Errorf("storage: unknown type %q", t)
	}
}

// buildStack composes ordered layer descriptors into a stack, read top-down in config
// order. Since ranke-go v0.3.0 the write role and content cap are what the universe
// reports through ranke.Capabilities, not options of the composition.
func buildStack(ctx context.Context, layers []scope.Section) (ranke.Universe, error) {
	if len(layers) == 0 {
		return nil, fmt.Errorf("storage: stack has no layers")
	}
	built := make([]ranke.Universe, 0, len(layers))
	for i, l := range layers {
		u, err := build(ctx, l)
		if err != nil {
			return nil, fmt.Errorf("storage: stack layer %d: %w", i, err)
		}
		if err := checkLayer(ctx, l, u); err != nil {
			return nil, fmt.Errorf("storage: stack layer %d: %w", i, err)
		}
		built = append(built, u)
	}
	return stack.NewStack(built...)
}

// checkLayer holds a layer's declared role and cap against what the built universe
// reports, so a wish the backend cannot honour fails rather than downgrading silently.
func checkLayer(ctx context.Context, l scope.Section, u ranke.Universe) error {
	caps := u.Capabilities()
	if l.HasValue("mode") {
		mode, err := l.Get(ctx, "mode")
		if err != nil {
			return fmt.Errorf("mode: %w", err)
		}
		want := ranke.StorageTier(mode)
		switch want {
		case ranke.StorageTierAuthoritative, ranke.StorageTierEager,
			ranke.StorageTierBackground, ranke.StorageTierLazy:
		default:
			return fmt.Errorf("mode: unknown %q (want authoritative|eager|background|lazy)", mode)
		}
		if want != caps.Tier {
			return fmt.Errorf("mode %q: this backend serves the %q tier; the tier is an adapter option in ranke-go, not a stack option — drop the key, or use a backend that offers one", want, caps.Tier)
		}
	}
	if l.HasValue("maxContentSize") {
		raw, err := l.Get(ctx, "maxContentSize")
		if err != nil {
			return fmt.Errorf("maxContentSize: %w", err)
		}
		size, err := parseSize(raw)
		if err != nil {
			return fmt.Errorf("maxContentSize: %w", err)
		}
		if size != caps.ContentCap {
			return fmt.Errorf("maxContentSize %d: this backend caps content at %d (0 = uncapped); the cap is an adapter option in ranke-go, not a stack option", size, caps.ContentCap)
		}
	}
	if l.HasValue("noReadFill") {
		v, err := l.Get(ctx, "noReadFill")
		if err != nil {
			return fmt.Errorf("noReadFill: %w", err)
		}
		if v == "true" {
			return fmt.Errorf("noReadFill: ranke-go's stack repairs a read miss itself; the switch no longer exists — remove the key")
		}
	}
	return nil
}

// buildPartition routes content by id mod shard-count. Shards are bare universes.
func buildPartition(ctx context.Context, shards []scope.Section) (ranke.Universe, error) {
	if len(shards) == 0 {
		return nil, fmt.Errorf("storage: partition has no shards")
	}
	us := make([]ranke.Universe, 0, len(shards))
	for i, sh := range shards {
		u, err := build(ctx, sh)
		if err != nil {
			return nil, fmt.Errorf("storage: shard %d: %w", i, err)
		}
		us = append(us, u)
	}
	return partition.NewPartition(us...)
}

// typeOf reads the descriptor's "type", defaulting to "" when absent.
func typeOf(ctx context.Context, sec scope.Section) (string, error) {
	if !sec.HasValue("type") {
		return "", nil
	}
	return sec.Get(ctx, "type")
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
