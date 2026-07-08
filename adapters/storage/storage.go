// package: storage / composition
// type:    factory
// job:     build one ranke.Universe from a storage section — a leaf backend, or a stack/partition composed of more universes
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

// Storage is the storage port's product: a ranke.Universe. Because a composed
// stack or partition implements Universe too, the whole store presents as this
// one type regardless of how it is layered.
type Storage = ranke.Universe

// New builds the storage universe described by sec (see the package doc for the
// descriptor shape).
func New(ctx context.Context, sec scope.Section) (Storage, error) {
	return build(ctx, sec)
}

// build dispatches one descriptor to a leaf backend or a composite, recursing
// for composites.
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
		dir, err := sec.GetValue("dir").Get(ctx)
		if err != nil {
			return nil, err
		}
		return fs.New(dir)
	case "sqlite":
		dsn, err := sec.GetValue("dsn").Get(ctx)
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

// buildStack composes ordered layer descriptors into an eager/lazy stack. Each
// layer is a universe descriptor plus the stack options mode (eager|lazy),
// maxContentSize, and noReadFill. The stack reads top-down and the first layer
// must be eager; the order is preserved from the config.
func buildStack(ctx context.Context, layers []scope.Section) (ranke.Universe, error) {
	if len(layers) == 0 {
		return nil, fmt.Errorf("storage: stack has no layers")
	}
	built := make([]stack.Layer, 0, len(layers))
	for i, l := range layers {
		u, err := build(ctx, l)
		if err != nil {
			return nil, fmt.Errorf("storage: stack layer %d: %w", i, err)
		}
		opts, err := layerOpts(ctx, l)
		if err != nil {
			return nil, fmt.Errorf("storage: stack layer %d: %w", i, err)
		}
		mode := "eager"
		if l.HasValue("mode") {
			if mode, err = l.GetValue("mode").Get(ctx); err != nil {
				return nil, fmt.Errorf("storage: stack layer %d: mode: %w", i, err)
			}
		}
		switch mode {
		case "eager", "":
			built = append(built, stack.Eager(u, opts...))
		case "lazy":
			built = append(built, stack.Lazy(u, opts...))
		default:
			return nil, fmt.Errorf("storage: stack layer %d: unknown mode %q (want eager|lazy)", i, mode)
		}
	}
	return stack.NewStack(built...)
}

// layerOpts reads a layer's optional stack options (maxContentSize, noReadFill).
func layerOpts(ctx context.Context, l scope.Section) ([]stack.Option, error) {
	var opts []stack.Option
	if l.HasValue("maxContentSize") {
		raw, err := l.GetValue("maxContentSize").Get(ctx)
		if err != nil {
			return nil, fmt.Errorf("maxContentSize: %w", err)
		}
		size, err := parseSize(raw)
		if err != nil {
			return nil, fmt.Errorf("maxContentSize: %w", err)
		}
		if size > 0 {
			opts = append(opts, stack.MaxContentSize(size))
		}
	}
	if l.HasValue("noReadFill") {
		v, err := l.GetValue("noReadFill").Get(ctx)
		if err != nil {
			return nil, fmt.Errorf("noReadFill: %w", err)
		}
		if v == "true" {
			opts = append(opts, stack.NoReadFill())
		}
	}
	return opts, nil
}

// buildPartition composes shard descriptors into a partition that routes content
// by id mod shard-count. Shards are bare universes, not stack layers.
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
	return sec.GetValue("type").Get(ctx)
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
