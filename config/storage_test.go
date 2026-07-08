package config

import (
	"context"
	"strings"
	"testing"

	"github.com/flocko-motion/ranke-go"
)

// TestBuildStorageStack assembles a two-layer stack (eager mem over lazy fs)
// from a single storage descriptor and round-trips content through it, proving
// the storage section resolves into a working Universe with its layers composed
// in order.
func TestBuildStorageStack(t *testing.T) {
	dir := t.TempDir()
	cfgJSON := `{
		"storage": {
			"type": "stack",
			"layers": [
				{"mode": "eager", "type": "mem", "maxContentSize": "8kb"},
				{"mode": "lazy",  "type": "fs", "dir": "` + dir + `"}
			]
		}
	}`

	app, err := Run(context.Background(), strings.NewReader(cfgJSON), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
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
