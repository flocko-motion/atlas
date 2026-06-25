package config_test

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	"rankedb/adapter/config"
	"rankedb/adapter/config/mem"
)

func TestCellRoundTripAndBase64(t *testing.T) {
	ctx := context.Background()
	store := mem.New()
	c := config.NewCell(store, "vault.value.age")

	if _, err := c.Get(ctx); !errors.Is(err, config.ErrCellEmpty) {
		t.Fatalf("Get on empty cell = %v, want ErrCellEmpty", err)
	}

	want := []byte("ciphertext-bytes")
	if err := c.Set(ctx, want); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := c.Get(ctx)
	if err != nil || string(got) != string(want) {
		t.Fatalf("Get = (%q, %v), want %q", got, err, want)
	}

	// The stored value is base64 under exactly the bound key, nothing else.
	e, _ := store.Load(ctx)
	if len(e) != 1 {
		t.Fatalf("store has %d keys, want 1", len(e))
	}
	if e["vault.value.age"] != base64.StdEncoding.EncodeToString(want) {
		t.Fatalf("stored value is not base64 of the bytes under the bound key")
	}
}
