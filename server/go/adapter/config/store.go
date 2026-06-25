// package: config / store
// type:    interface
// job:     define the config storage seam — Entries (flat dotted-key map) and the Store backend interface
// limits:  no schema/provenance (-> field.go), no base64/single-key mediation (-> cell.go), no backend impl (-> env/file/mem/postgres)
package config

import "context"

// Entries is a flat, dotted-key view of stored config ("server.port" ->
// "8080"). It is the lowest-common-denominator a config Store persists; the
// schema + provenance (Field) layer is built on top of it.
type Entries map[string]string

// Store is a storage backend for the server config — the config-family
// analog of ranke's Universe / BranchTableHead seams. Backends: yaml, env,
// the framework's built-in Postgres, any external Postgres, …
type Store interface {
	Load(ctx context.Context) (Entries, error)
	Save(ctx context.Context, e Entries) error
	Close() error
}
