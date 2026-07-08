// package: storage / composition
// type:    factory
// job:     compose resolved layer specs into one ranke.Universe (stack of eager/lazy layers, partitioned shards)
// limits:  wiring only; the persistence logic is ranke-go's adapters (-> github.com/flocko-motion/ranke-go)
//
// Package storage is ranke-db's storage port: it assembles the launch config's
// layer specs into a single ranke.Universe. Each leaf spec names a ranke-go
// storage adapter (fs, sqlite, mem, minimal, s3) and carries that instance's
// resolved scope; storage builds the leaf, wraps it eager or lazy with its
// options, and composes the layers with ranke-go's stack. A partition spec
// composes its shard universes with ranke-go's partition. The composition root
// (config) owns slicing the config into per-instance scopes and resolving their
// secrets; this package owns turning the resolved specs into the Universe.
package storage

import (
	"fmt"

	"github.com/flocko-motion/ranke-go"
	"github.com/flocko-motion/ranke-go/adapter/storage/fs"
	"github.com/flocko-motion/ranke-go/adapter/storage/mem"
	"github.com/flocko-motion/ranke-go/adapter/storage/minimal"
	"github.com/flocko-motion/ranke-go/adapter/storage/partition"
	"github.com/flocko-motion/ranke-go/adapter/storage/sqlite"
	"github.com/flocko-motion/ranke-go/adapter/storage/stack"

	"github.com/flocko-motion/rankedb/config/scope"
)

// Spec is one resolved storage layer. Mode is "eager" or "lazy"; Type names the
// backend; Scope carries that instance's resolved settings (a leaf's connection
// details, never another instance's). MaxContentSize caps what a cache layer
// holds (0 = uncapped); NoReadFill disables back-filling this layer on a read
// hit below it. Shards is non-empty only for the "partition" type, holding the
// shard sub-specs the partition routes across.
type Spec struct {
	Mode           string
	Type           string
	Scope          scope.Config
	MaxContentSize uint64
	NoReadFill     bool
	Shards         []Spec
}

// New composes specs into one Universe: each becomes an eager or lazy stack
// layer over a freshly built leaf (or partition). The order is significant —
// the stack reads top-down and the first layer must be eager — and is preserved
// from the config. An empty specs list is an error.
func New(specs []Spec) (ranke.Universe, error) {
	if len(specs) == 0 {
		return nil, fmt.Errorf("storage: no layers configured")
	}
	layers := make([]stack.Layer, 0, len(specs))
	for i, s := range specs {
		u, err := leaf(s)
		if err != nil {
			return nil, fmt.Errorf("storage: layer %d (%s): %w", i, s.Type, err)
		}
		var opts []stack.Option
		if s.MaxContentSize > 0 {
			opts = append(opts, stack.MaxContentSize(s.MaxContentSize))
		}
		if s.NoReadFill {
			opts = append(opts, stack.NoReadFill())
		}
		switch s.Mode {
		case "eager", "":
			layers = append(layers, stack.Eager(u, opts...))
		case "lazy":
			layers = append(layers, stack.Lazy(u, opts...))
		default:
			return nil, fmt.Errorf("storage: layer %d: unknown mode %q (want eager|lazy)", i, s.Mode)
		}
	}
	return stack.NewStack(layers...)
}

// leaf builds a single backend Universe from a spec. A partition spec recurses,
// composing its shard sub-specs; the storage backends draw their settings from
// the spec's resolved scope.
func leaf(s Spec) (ranke.Universe, error) {
	switch s.Type {
	case "fs":
		dir, err := s.Scope.Require("dir")
		if err != nil {
			return nil, err
		}
		return fs.New(dir)
	case "sqlite":
		dsn, err := s.Scope.Require("dsn")
		if err != nil {
			return nil, err
		}
		return sqlite.New(dsn)
	case "mem":
		return mem.New(), nil
	case "minimal":
		return minimal.New(), nil
	case "partition":
		return buildPartition(s.Shards)
	case "s3":
		// ranke-go's s3 adapter takes a constructed *s3.Client; wiring the AWS
		// SDK (region/endpoint/credentials from the scope) is a dedicated pass.
		return nil, fmt.Errorf("s3 backend not yet wired")
	case "":
		return nil, fmt.Errorf("missing type")
	default:
		return nil, fmt.Errorf("unknown type %q", s.Type)
	}
}

// buildPartition builds each shard sub-spec as a bare leaf (shards are not
// themselves eager/lazy stack layers) and composes them with ranke-go's
// partition, which routes content by id mod shard-count.
func buildPartition(shards []Spec) (ranke.Universe, error) {
	if len(shards) == 0 {
		return nil, fmt.Errorf("partition: no shards configured")
	}
	us := make([]ranke.Universe, 0, len(shards))
	for i, sh := range shards {
		u, err := leaf(sh)
		if err != nil {
			return nil, fmt.Errorf("shard %d (%s): %w", i, sh.Type, err)
		}
		us = append(us, u)
	}
	return partition.NewPartition(us...)
}
