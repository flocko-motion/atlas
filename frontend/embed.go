//go:build explorer

// package: frontend / static asset
// type:    embed shim
// job:     embeds the explorer's distributable into the ranke-db binary
// limits:  opt-in via the `explorer` build tag only (-> rest_http)
package frontend

import "embed"

// Explorer holds dist/explorer.html. An embed.FS, not a []byte: today's
// distributable is one self-contained file, but the shape holds unchanged if that
// ever becomes a directory instead.
//
//go:embed dist/explorer.html
var Explorer embed.FS
