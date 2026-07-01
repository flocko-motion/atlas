package config

import (
	"context"
	"strings"
	"testing"

	"github.com/flocko-motion/ranke-go"
)

// TestBuildStorageStack assembles a two-layer stack (eager mem over lazy fs)
// from config and round-trips content through it, proving the storage section
// resolves into a working Universe with the layers composed in order.
func TestBuildStorageStack(t *testing.T) {
	dir := t.TempDir()
	cfgJSON := `{
		"storage": [
			{"mode": "eager", "type": "mem", "maxContentSize": "8kb"},
			{"mode": "lazy",  "type": "fs", "dir": "` + dir + `"}
		]
	}`

	c, err := Load(strings.NewReader(cfgJSON))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	app, err := c.Build(context.Background(), nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if app.Storage == nil {
		t.Fatal("nil storage universe")
	}
	t.Cleanup(func() { _ = app.Storage.Close() })

	ctx := context.Background()
	content := []byte("ranke storage stack round-trip")
	hash, err := ranke.HashContent(content)
	if err != nil {
		t.Fatalf("HashContent: %v", err)
	}
	if err := ranke.PutContent(ctx, app.Storage, hash, content); err != nil {
		t.Fatalf("PutContent: %v", err)
	}
	got, err := ranke.GetContent(ctx, app.Storage, hash, uint64(len(content)))
	if err != nil {
		t.Fatalf("GetContent: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("round-trip = %q, want %q", got, content)
	}
}

// TestParseSize covers the human-readable size suffixes the maxContentSize
// field accepts.
func TestParseSize(t *testing.T) {
	cases := map[string]uint64{
		"":      0,
		"512":   512,
		"8kb":   8 << 10,
		"2 MB":  2 << 20,
		"1gb":   1 << 30,
		"4096b": 4096,
	}
	for in, want := range cases {
		got, err := parseSize(in)
		if err != nil {
			t.Errorf("parseSize(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("parseSize(%q) = %d, want %d", in, got, want)
		}
	}
	if _, err := parseSize("not-a-size"); err == nil {
		t.Error("parseSize(\"not-a-size\") = nil error, want error")
	}
}
