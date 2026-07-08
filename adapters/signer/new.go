// package: signer / crypto
// type:    factory
// job:     build the configured signer backend from the narrow signer section
// limits:  dispatch only; key loading lives in the backend (-> adapters/signer/inmemory)
//
// This file is the signer port's composition seam. It takes only the signer
// instance's section — a navigable view holding no field outside the signer
// section — and dispatches to the chosen backend, passing the section on.
package signer

import (
	"context"
	"fmt"

	"github.com/flocko-motion/rankedb/adapters/signer/inmemory"
	"github.com/flocko-motion/rankedb/config/scope"
)

// New builds the signer backend named by the section's "type" value, handing the
// backend the same section to read its key material from. It returns an error for
// an empty or unknown type. Delegated backends (openbao, azure) land here as
// they are added.
func New(ctx context.Context, cfg scope.Section) (Signer, error) {
	var t string
	if cfg.HasValue("type") {
		var err error
		if t, err = cfg.GetValue("type").Get(ctx); err != nil {
			return nil, fmt.Errorf("signer: type: %w", err)
		}
	}
	switch t {
	case "inmemory":
		return inmemory.New(ctx, cfg)
	case "":
		return nil, fmt.Errorf("signer: no backend type configured")
	default:
		return nil, fmt.Errorf("signer: unknown backend type %q", t)
	}
}
