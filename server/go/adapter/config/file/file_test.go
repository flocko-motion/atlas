package file_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"rankedb/adapter/config"
	"rankedb/adapter/config/file"
)

// load writes content to a temp file with the given name (suffix drives
// format detection) and returns the loaded Entries.
func load(t *testing.T, name, content string) config.Entries {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := file.New(path)
	if err != nil {
		t.Fatalf("file.New(%s): %v", name, err)
	}
	e, err := s.Load(context.Background())
	if err != nil {
		t.Fatalf("Load(%s): %v", name, err)
	}
	return e
}

func assertEq(t *testing.T, e config.Entries, key, want string) {
	t.Helper()
	if got := e[key]; got != want {
		t.Errorf("entry %q = %q, want %q", key, got, want)
	}
}

func TestYAMLNestedFlattens(t *testing.T) {
	e := load(t, "c.yaml", "server:\n  port: 8080\naccounts:\n  - name: alice\n")
	assertEq(t, e, "server.port", "8080")
	assertEq(t, e, "accounts.0.name", "alice")
}

func TestJSON(t *testing.T) {
	e := load(t, "c.json", `{"server":{"port":9090},"log_level":"debug"}`)
	assertEq(t, e, "server.port", "9090")
	assertEq(t, e, "log_level", "debug")
}

func TestTOML(t *testing.T) {
	e := load(t, "c.toml", "log_level = \"info\"\n[server]\nport = 7000\n")
	assertEq(t, e, "server.port", "7000")
	assertEq(t, e, "log_level", "info")
}

func TestDotEnvVerbatimKeys(t *testing.T) {
	e := load(t, "c.env", "# comment\nexport server.port = \"8080\"\nlog_level=debug\n")
	assertEq(t, e, "server.port", "8080")
	assertEq(t, e, "log_level", "debug")
}

func TestUnsupportedSuffix(t *testing.T) {
	if _, err := file.New("/tmp/c.ini"); err == nil {
		t.Fatal("expected error for unsupported suffix .ini")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	for _, name := range []string{"c.yaml", "c.json", "c.env"} {
		path := filepath.Join(t.TempDir(), name)
		s, err := file.New(path)
		if err != nil {
			t.Fatalf("New(%s): %v", name, err)
		}
		in := config.Entries{"server.port": "8080", "log_level": "debug"}
		if err := s.Save(context.Background(), in); err != nil {
			t.Fatalf("Save(%s): %v", name, err)
		}
		got, err := s.Load(context.Background())
		if err != nil {
			t.Fatalf("Load(%s): %v", name, err)
		}
		for k, v := range in {
			if got[k] != v {
				t.Errorf("%s round-trip: %q = %q, want %q", name, k, got[k], v)
			}
		}
	}
}
