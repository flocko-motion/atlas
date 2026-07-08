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

// Value is one config leaf handed to the adapter that needs it. Get returns the
// plaintext, resolving an env()/vault() reference on demand — so a rotating
// vault secret is fetched when the adapter uses it rather than frozen at launch.
// The source (literal, env(), or vault()) is chosen by config when it builds the
// Value and is opaque here: the adapter triggers resolution, it does not pick the
// source or reach the environment or vault itself.
type Value interface {
	// Get resolves and returns the value. It errors when the key was absent, or
	// when resolving its source (an unset env var, an unreachable vault) fails.
	Get(ctx context.Context) (string, error)
}

// Section is one instance's slice of the launch config: a navigable node that
// yields nested Sections and leaf Values. It holds no reference to its parent or
// siblings, so an adapter handed a Section can reach only what is inside it.
type Section interface {
	// GetSection descends into a nested object. A missing or non-object key
	// yields an empty Section (never nil), so navigation cannot panic.
	GetSection(key string) Section

	// GetValue returns the leaf at key. A missing key yields a Value whose Get
	// reports the absence, so callers may resolve first and judge required-ness
	// from the error.
	GetValue(key string) Value

	// HasValue reports whether key representing a value is present, resolving nothing.
	// Use it to make an optional setting default cleanly.
	HasValue(key string) bool

	// HasSection reports wether key representing a section is present
	HasSection(key string) bool

	// HasKey reports whether a key is present
	HasKey(key string) bool

	// Keys lists the present keys in this section, resolving nothing. It is for
	// structural iteration (e.g. a list of configured endpoints).
	Keys() []string
}
