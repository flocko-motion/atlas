// package: scope / config
// type:    interface
// job:     the navigable, lazily-resolved handoff between the composition root and one adapter
// limits:  interfaces only; the concrete tree + env()/vault() resolution live in config (-> config)
//
// Narrowing is by containment: a sibling's settings and another port's secrets are simply
// not in the object an adapter was handed.
package scope

import "context"

// Section is one instance's slice of the launch config, reaching only inward.
type Section interface {
	// GetSection descends; a missing key yields an empty Section, never nil.
	GetSection(key string) Section

	// Get resolves the leaf at key, expanding an env()/vault() reference on demand, so a
	// rotating secret is fetched at use. The source (literal, env(), vault())
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
