package scope

import (
	"context"
	"testing"
)

func TestLiteralHasNoArraysOrSections(t *testing.T) {
	s := Literal(map[string]string{"key": "value"})
	if s.HasArray("key") || s.HasSection("key") {
		t.Fatal("Literal reported an array or section for a flat value")
	}
	if got := s.GetArray("missing"); got != nil {
		t.Fatalf("GetArray(missing) = %v, want nil", got)
	}
}

func TestLiteralArrayCarriesArrayValuedKeys(t *testing.T) {
	inner := Literal(map[string]string{"account": "webapp", "sha256": "deadbeef"})
	s := LiteralArray(map[string]string{"type": "apikey"}, map[string][]Section{"keys": {inner}})

	if got, err := s.Get(context.Background(), "type"); err != nil || got != "apikey" {
		t.Fatalf("Get(type) = %q, %v, want apikey, nil", got, err)
	}
	if !s.HasArray("keys") || s.HasValue("keys") {
		t.Fatal("keys should report as an array, not a flat value")
	}
	arr := s.GetArray("keys")
	if len(arr) != 1 {
		t.Fatalf("GetArray(keys) has %d elements, want 1", len(arr))
	}
	if got, err := arr[0].Get(context.Background(), "account"); err != nil || got != "webapp" {
		t.Fatalf("GetArray(keys)[0].Get(account) = %q, %v, want webapp, nil", got, err)
	}
	if !s.HasKey("type") || !s.HasKey("keys") || s.HasKey("absent") {
		t.Fatal("HasKey should report both flat and array keys, and nothing absent")
	}
}
