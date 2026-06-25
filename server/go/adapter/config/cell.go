// package: config / cell
// type:    logic
// job:     mediate a single config key as raw bytes (base64 in the store), the narrow authority a vault is granted
// limits:  one key only — no schema/provenance (-> field.go), no whole-config access (-> store.go)
package config

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
)

// ErrCellEmpty is returned by Cell.Get when the bound key is not set.
var ErrCellEmpty = errors.New("config: cell is empty (key not set)")

// Cell is a narrow, single-value handle onto one config key — the minimal
// authority a component (e.g. a vault) is granted over config. The value is
// base64 in the store; callers work in raw bytes and never see — or get to
// name — any other config key. This is what lets a vault read/write only its
// own blob (e.g. "vault.value.age") and nothing else.
type Cell interface {
	Get(ctx context.Context) ([]byte, error)
	Set(ctx context.Context, b []byte) error
}

// NewCell binds a Cell to a single key over store, mediating base64.
func NewCell(store Store, key string) Cell {
	return &cell{store: store, key: key}
}

type cell struct {
	store Store
	key   string
}

func (c *cell) Get(ctx context.Context) ([]byte, error) {
	e, err := c.store.Load(ctx)
	if err != nil {
		return nil, err
	}
	v, ok := e[c.key]
	if !ok {
		return nil, ErrCellEmpty
	}
	b, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return nil, fmt.Errorf("config: key %q is not valid base64: %w", c.key, err)
	}
	return b, nil
}

func (c *cell) Set(ctx context.Context, b []byte) error {
	e, err := c.store.Load(ctx)
	if err != nil {
		return err
	}
	if e == nil {
		e = Entries{}
	}
	e[c.key] = base64.StdEncoding.EncodeToString(b)
	return c.store.Save(ctx, e)
}
