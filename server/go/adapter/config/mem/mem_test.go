package mem_test

import (
	"context"
	"testing"

	"rankedb/adapter/config"
	"rankedb/adapter/config/mem"
)

func TestNewFromSeedsAndCopies(t *testing.T) {
	ctx := context.Background()
	seed := config.Entries{"server.addr": ":8080"}
	s := mem.NewFrom(seed)

	got, err := s.Load(ctx)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got["server.addr"] != ":8080" {
		t.Fatalf("Load = %v; want seeded server.addr", got)
	}

	// NewFrom copies the seed: mutating it afterwards must not reach the store.
	seed["server.addr"] = ":9999"
	if got, _ := s.Load(ctx); got["server.addr"] != ":8080" {
		t.Fatalf("store reflected a post-construction seed mutation: %v", got)
	}
}
