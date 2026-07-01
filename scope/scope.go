// package: scope / config
// type:    struct
// job:     a tiny resolved key/value config object scoped to exactly one adapter instance
// limits:  values are already plaintext; holds no reference to the full config (-> config)
//
// Package scope is the handoff between the composition root and an adapter. The
// root slices the launch config into one Scope per adapter instance, resolves
// every env()/vault() delegation, and hands the adapter only its slice. The
// narrowing is by containment, not visibility: a Scope physically holds only
// that instance's keys and no back-reference to the rest of the config, so an
// adapter can read neither another port's secrets nor a sibling instance's —
// there is nothing else in the object to read. It lives in its own leaf package
// so adapters depend on it without importing config (config -> adapter ->
// scope, never the reverse).
package scope

import "fmt"

// Config is a resolved, instance-scoped set of config values. The zero value is
// an empty scope. All values are plaintext: any env()/vault() delegation was
// expanded when the scope was built, so an adapter never sees a placeholder.
type Config struct {
	values map[string]string
}

// New returns a Scope over values. The map is taken as given (not copied);
// callers build a fresh map per instance and do not retain it.
func New(values map[string]string) Config {
	return Config{values: values}
}

// String returns the value for key, or "" if the scope has no such key. Use it
// for optional settings where empty is a sensible default.
func (c Config) String(key string) string { return c.values[key] }

// Has reports whether the scope contains key (even if its value is empty).
func (c Config) Has(key string) bool {
	_, ok := c.values[key]
	return ok
}

// Bool reports whether key is present and set to "true".
func (c Config) Bool(key string) bool { return c.values[key] == "true" }

// Require returns the value for key, or an error if the key is absent or empty.
// Use it for settings an adapter cannot run without.
func (c Config) Require(key string) (string, error) {
	v, ok := c.values[key]
	if !ok || v == "" {
		return "", fmt.Errorf("scope: required config key %q is missing", key)
	}
	return v, nil
}
