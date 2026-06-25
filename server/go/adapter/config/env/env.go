// package: env / adapter
// type:    adapter
// job:     read config from prefix-filtered process env vars (a read-only overlay), mapping NAMEs to dotted keys
// limits:  read-only — Save is unsupported (-> file/postgres for writable stores)
//
// Package env loads the ranke-db server config from process environment
// variables — a read-only overlay source, typically used to override
// selected fields. Variables are filtered by prefix; the remainder maps to
// a dotted config key by lowercasing and turning "__" into ".":
//
//	RANKE_SERVER__PORT=9090   (prefix "RANKE_")  ->  server.port = 9090
//	RANKE_LOG_LEVEL=debug                        ->  log_level   = debug
//
// (Double underscore = level separator; single underscore is literal.)
package env

import (
	"context"
	"errors"
	"os"
	"strings"

	"rankedb/adapter/config"
)

// New returns a read-only config Store over the process environment, reading
// variables named prefix+KEY.
func New(prefix string) config.Store { return &store{prefix: prefix} }

type store struct{ prefix string }

func (s *store) Load(_ context.Context) (config.Entries, error) {
	out := config.Entries{}
	for _, kv := range os.Environ() {
		i := strings.IndexByte(kv, '=')
		if i < 0 {
			continue
		}
		name, val := kv[:i], kv[i+1:]
		if !strings.HasPrefix(name, s.prefix) {
			continue
		}
		out[key(strings.TrimPrefix(name, s.prefix))] = val
	}
	return out, nil
}

// Save is unsupported: the process environment is read-only at runtime.
func (s *store) Save(context.Context, config.Entries) error {
	return errors.New("adapter/config/env: read-only (cannot Save to the process environment)")
}

func (s *store) Close() error { return nil }

// key maps an env var name (prefix already stripped) to a dotted config key:
// lowercase, "__" -> ".".
func key(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), "__", ".")
}
