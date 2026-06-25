package config

import "testing"

func TestProvenanceSingleSource(t *testing.T) {
	f := Field[int]{Value: 8080, Trail: []Origin{
		{Backend: "seed", Locator: "./dev.yaml", Value: "8080"},
	}}
	if got := f.From().Backend; got != "seed" {
		t.Fatalf("From().Backend = %q, want seed", got)
	}
	if got, want := f.Provenance(), "8080 ⟵ seed:./dev.yaml"; got != want {
		t.Fatalf("Provenance() = %q, want %q", got, want)
	}
}

func TestProvenanceOverlayShadows(t *testing.T) {
	f := Field[int]{Value: 9090, Trail: []Origin{
		{Backend: "env", Locator: "RANKE_PORT", Value: "9090"},
		{Backend: "seed", Locator: "./dev.yaml", Value: "8080"},
	}}
	want := "9090 ⟵ env:RANKE_PORT (shadowed seed:./dev.yaml=8080)"
	if got := f.Provenance(); got != want {
		t.Fatalf("Provenance() = %q, want %q", got, want)
	}
}

func TestProvenanceSealed(t *testing.T) {
	f := Field[string]{Value: "main", Sealed: true, Trail: []Origin{
		{Backend: "seed", Locator: "./dev.yaml", Value: "main"},
	}}
	want := "main ⟵ seed:./dev.yaml [sealed]"
	if got := f.Provenance(); got != want {
		t.Fatalf("Provenance() = %q, want %q", got, want)
	}
}

func TestUnsetField(t *testing.T) {
	var f Field[int]
	if f.From() != (Origin{}) {
		t.Fatalf("From() on unset = %+v, want zero", f.From())
	}
	if got, want := f.Provenance(), "0 ⟵ (unset)"; got != want {
		t.Fatalf("Provenance() = %q, want %q", got, want)
	}
}
