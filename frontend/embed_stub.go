//go:build !explorer

// package: frontend / static asset
// type:    embed shim
// job:     the default (non-embedding) build of Explorer
// limits:  the `!explorer` counterpart to embed.go's build tag
package frontend

import "embed"

// Explorer is an empty FS without the `explorer` build tag — ReadFile always
// misses, which is what a plain `go build` (dev, tests) is meant to do here.
var Explorer embed.FS
