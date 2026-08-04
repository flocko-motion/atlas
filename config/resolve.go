// package: config / composition
// type:    func
// job:     expand a single env(KEY)/vault(ref) delegation to its plaintext value
// limits:  whole-value match only; no embedded interpolation (-> scopeOf)
//
// A value is a literal or one whole env(KEY)/vault(ref) placeholder — no interpolation,
// so no secret hides inside a larger string. vaultBox builds the store lazily.
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

// resolveValue expands an env()/vault() delegation, or returns a literal unchanged. A
// missing value is an error: resolution fails loud rather than yield an empty secret.
func resolveValue(ctx context.Context, s string, box *vaultBox) (string, error) {
	if m := envRef.FindStringSubmatch(s); m != nil {
		val, ok := os.LookupEnv(m[1])
		if !ok {
			return "", fmt.Errorf("env(%s) is not set", m[1])
		}
		return val, nil
	}
	if m := vaultRef.FindStringSubmatch(s); m != nil {
		val, err := box.secret(ctx, m[1])
		if err != nil {
			return "", fmt.Errorf("vault(%s): %w", m[1], err)
		}
		return val, nil
	}
	return s, nil
}
