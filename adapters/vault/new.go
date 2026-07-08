// package: vault / secrets
// type:    factory
// job:     build the configured vault backend from the vault section
// limits:  dispatch only; secret fetching lives in the backend (-> adapters/vault/openbao, azure)
//
// This file is the vault port's composition seam. It takes the vault section and
// dispatches to the chosen backend, passing the section on. The vault is built
// env-only (secret-zero cannot resolve vault() before the vault exists).
package vault

import (
	"context"
	"fmt"

	"github.com/flocko-motion/rankedb/adapters/vault/azure"
	"github.com/flocko-motion/rankedb/adapters/vault/openbao"
	"github.com/flocko-motion/rankedb/config/scope"
)

// New builds the vault backend named by the section's "type": "openbao" (KV v2)
// or "azure" (Azure Key Vault, scaffold). An empty or unknown type is an error.
func New(ctx context.Context, cfg scope.Section) (Vault, error) {
	var t string
	if cfg.HasValue("type") {
		var err error
		if t, err = cfg.GetValue("type").Get(ctx); err != nil {
			return nil, fmt.Errorf("vault: type: %w", err)
		}
	}
	switch t {
	case "openbao":
		return openbao.New(ctx, cfg)
	case "azure":
		return azure.New(ctx, cfg)
	case "":
		return nil, fmt.Errorf("vault: no backend type configured")
	default:
		return nil, fmt.Errorf("vault: unknown backend type %q", t)
	}
}
