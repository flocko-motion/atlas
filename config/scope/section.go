// package: scope / config
// type:    interface
// job:     the navigable, lazily-resolved handoff between the composition root and one adapter
// limits:  interfaces only; the concrete tree + env()/vault() resolution live in config (-> config)
//
// This file defines the handoff contract between the composition root and an
// adapter. The root parses the launch config, then hands each adapter one
// Section — its own slice — and nothing else. A Section is navigable (nested
// Sections, leaf Values) but holds no path back to the rest of the config, so
// the narrowing is by containment, not visibility: an adapter can read neither
// a sibling instance's settings nor another port's secrets, because they are
// simply not in the object it was handed. A Value resolves lazily on Get, and
// whether it came from a literal, an env() reference, or a vault() reference is
// decided by config and opaque to the adapter.
package scope

import "context"

// Section is one instance's slice of the launch config: a navigable node that
// yields nested Sections and reads leaf values. It holds no reference to its
// parent or siblings, so an adapter handed a Section can reach only what is inside
// it.
type Section interface {
	// GetSection descends into a nested object. A missing or non-object key
	// yields an empty Section (never nil), so navigation cannot panic.
	GetSection(key string) Section

	// Get resolves the leaf at key to its plaintext, expanding an env()/vault()
	// reference on demand — so a rotating vault secret is fetched when the adapter
	// reads it rather than frozen at launch. The source (literal, env(), vault())
	// is chosen by config and opaque here. It errors when the key is absent, or
	// when resolving its source (an unset env var, an unreachable vault) fails; an
	// adapter judges required-ness from that error, or checks HasValue first.
	Get(ctx context.Context, key string) (string, error)

	// GetArray returns the elements of an array-valued key, each wrapped as a
	// Section. A missing key or a non-array value yields an empty slice. Config
	// arrays are always arrays of objects (a stack's layers, a partition's shards,
	// the endpoints, the accounts), so callers range over the result.
	GetArray(key string) []Section

	// HasValue reports whether key is present as a leaf value — not a nested
	// section and not an array. Use it to make an optional setting default cleanly.
	HasValue(key string) bool

	// HasSection reports whether key is present as a nested section.
	HasSection(key string) bool

	// HasArray reports whether key is present as an array.
	HasArray(key string) bool

	// HasKey reports whether a key is present at all.
	HasKey(key string) bool

	// Keys lists the present keys in this section, resolving nothing. It is for
	// structural iteration (e.g. a list of configured endpoints).
	Keys() []string
}
