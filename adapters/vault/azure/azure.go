// package: azure / secrets
// type:    adapter
// job:     resolve vault(ref) secrets from Azure Key Vault
// limits:  SCAFFOLD — construction only, no fetching yet (-> adapters/vault)
//
// Package azure is the Azure Key Vault secret backend. It is scaffolded: it
// constructs from the vault section's "url" (the Key Vault URI) so the dispatch
// and config shape are in place, but Secret is not yet implemented and it has no
// test — the real fetch (against a live Key Vault) lands in a dedicated pass.
package azure

import (
	"context"
	"fmt"

	"github.com/rankegraph/ranke-db/config/scope"
)

// Vault is the Azure Key Vault backend (scaffold).
type Vault struct {
	url string
}

// New reads the optional "url" (Key Vault URI) from the vault section. Scaffold:
// construction succeeds; secret resolution is not yet wired.
func New(ctx context.Context, cfg scope.Section) (*Vault, error) {
	var url string
	if cfg.HasValue("url") {
		var err error
		if url, err = cfg.Get(ctx, "url"); err != nil {
			return nil, fmt.Errorf("vault/azure: url: %w", err)
		}
	}
	return &Vault{url: url}, nil
}

// Secret is not yet implemented.
func (v *Vault) Secret(context.Context, string) (string, error) {
	return "", fmt.Errorf("vault/azure: Azure Key Vault backend not yet implemented")
}
