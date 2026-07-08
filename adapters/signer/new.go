// package: signer / crypto
// type:    factory
// job:     build the configured signer backend from the narrow signer.Config view
// limits:  dispatch only; key loading lives in the backend (-> adapters/signer/inmemory)
//
// This file is the signer port's composition seam. It takes only the signer
// instance's scope — a resolved key/value object holding no field outside the
// signer section — and dispatches to the chosen backend, passing the scope on.
package signer

import (
	"fmt"

	"github.com/flocko-motion/rankedb/adapters/signer/inmemory"
	"github.com/flocko-motion/rankedb/config/scope"
)

// New builds the signer backend named by the scope's "type" value, handing the
// backend the same scope to read its key material from. It returns an error for
// an empty or unknown type. Delegated backends (openbao, azure) land here as
// they are added.
func New(cfg scope.Config) (Signer, error) {
	switch t := cfg.String("type"); t {
	case "inmemory":
		return inmemory.New(cfg)
	case "":
		return nil, fmt.Errorf("signer: no backend type configured")
	default:
		return nil, fmt.Errorf("signer: unknown backend type %q", t)
	}
}
