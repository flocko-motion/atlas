package assembler_test

import (
	"testing"

	"rankedb/assembler"
)

func TestSpecFromConfig(t *testing.T) {
	entries := map[string]string{
		"archives.main.storage.backend":   "fs",
		"archives.main.storage.dir":       "/data/main",
		"archives.main.sequencer.backend": "postgres",
		"archives.main.sequencer.dsn":     "postgres://x",
		// no sequencer.key → defaults to the archive name
		"archives.other.storage.backend": "mem", // a second archive must not bleed in
	}
	spec, err := assembler.SpecFromConfig(entries, "main")
	if err != nil {
		t.Fatalf("SpecFromConfig: %v", err)
	}
	if spec.Storage.Backend != "fs" || spec.Storage.Dir != "/data/main" {
		t.Fatalf("storage = %+v", spec.Storage)
	}
	if spec.Sequencer.Backend != "postgres" || spec.Sequencer.DSN != "postgres://x" {
		t.Fatalf("sequencer = %+v", spec.Sequencer)
	}
	if spec.Sequencer.Key != "main" {
		t.Fatalf("postgres sequencer key = %q, want default %q", spec.Sequencer.Key, "main")
	}
}

func TestSpecFromConfigRejectsIncomplete(t *testing.T) {
	cases := map[string]map[string]string{
		"empty":             {},
		"missing storage":   {"archives.main.sequencer.backend": "mem"},
		"missing sequencer": {"archives.main.storage.backend": "mem"},
	}
	for name, entries := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := assembler.SpecFromConfig(entries, "main"); err == nil {
				t.Fatalf("SpecFromConfig(%v) = nil error, want rejection", entries)
			}
		})
	}
	// Empty name is rejected regardless of entries.
	if _, err := assembler.SpecFromConfig(map[string]string{}, ""); err == nil {
		t.Fatal("SpecFromConfig with empty name should error")
	}
}
