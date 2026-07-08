// package: config / composition
// type:    func
// job:     expand a single env(KEY)/vault(ref) delegation to its plaintext value
// limits:  whole-value match only; no embedded interpolation (-> scopeOf)
//
// This file holds delegation resolution. A config value is either a literal, or
// the whole value is one env(KEY) or vault(ref) placeholder — there is no
// embedded interpolation, so a secret cannot be smuggled into a larger string.
// vault() requires the secret store, which the section's vaultBox builds lazily
// from the vault section (itself env-only, since it cannot resolve vault() before
// it exists) — so a config that never references vault() never dials one.
package config

import (
	"context"
	"fmt"
	"os"
	"regexp"
)

var (
	envRef   = regexp.MustCompile(`^env\((.+)\)$`)
	vaultRef = regexp.MustCompile(`^vault\((.+)\)$`)
)

// resolveValue expands an env() or vault() delegation in s, or returns s
// unchanged when it is a literal. An env(KEY) whose variable is unset, or a
// vault(ref) the box cannot satisfy (no vault section, or the vault fails to
// build), is an error — resolution fails loud rather than yielding a missing
// secret. A vault() reference builds the store through box on first use.
func resolveValue(ctx context.Context, s string, box *vaultBox) (string, error) {
	if m := envRef.FindStringSubmatch(s); m != nil {
		val, ok := os.LookupEnv(m[1])
		if !ok {
			return "", fmt.Errorf("env(%s) is not set", m[1])
		}
		return val, nil
	}
	if m := vaultRef.FindStringSubmatch(s); m != nil {
		v, err := box.get(ctx)
		if err != nil {
			return "", fmt.Errorf("vault(%s): %w", m[1], err)
		}
		return v.Secret(ctx, m[1])
	}
	return s, nil
}
