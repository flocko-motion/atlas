// package: config / composition
// type:    func
// job:     expand a single env(KEY)/vault(ref) delegation to its plaintext value
// limits:  whole-value match only; no embedded interpolation (-> scopeOf)
//
// This file holds delegation resolution. A config value is either a literal, or
// the whole value is one env(KEY) or vault(ref) placeholder — there is no
// embedded interpolation, so a secret cannot be smuggled into a larger string.
// vault() requires a Vault, which the composition root builds from the vault
// section (itself env-only, since it cannot resolve vault() before it exists).
package config

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/flocko-motion/rankedb/adapters/vault"
)

var (
	envRef   = regexp.MustCompile(`^env\((.+)\)$`)
	vaultRef = regexp.MustCompile(`^vault\((.+)\)$`)
)

// resolveValue expands an env() or vault() delegation in s, or returns s
// unchanged when it is a literal. An env(KEY) whose variable is unset, or a
// vault(ref) with no vault configured, is an error — Build fails loud rather
// than serving with a missing secret.
func resolveValue(ctx context.Context, s string, v vault.Vault) (string, error) {
	if m := envRef.FindStringSubmatch(s); m != nil {
		val, ok := os.LookupEnv(m[1])
		if !ok {
			return "", fmt.Errorf("env(%s) is not set", m[1])
		}
		return val, nil
	}
	if m := vaultRef.FindStringSubmatch(s); m != nil {
		if v == nil {
			return "", fmt.Errorf("vault(%s) referenced but no vault configured", m[1])
		}
		return v.Secret(ctx, m[1])
	}
	return s, nil
}
