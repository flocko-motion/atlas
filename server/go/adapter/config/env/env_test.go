package env_test

import (
	"context"
	"testing"

	"rankedb/adapter/config/env"
)

func TestLoadMapsKeys(t *testing.T) {
	t.Setenv("RANKE_SERVER__PORT", "9090") // __ -> .
	t.Setenv("RANKE_LOG_LEVEL", "debug")   // single _ stays literal
	t.Setenv("OTHER_THING", "ignored")     // wrong prefix

	e, err := env.New("RANKE_").Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := e["server.port"]; got != "9090" {
		t.Errorf("server.port = %q, want 9090", got)
	}
	if got := e["log_level"]; got != "debug" {
		t.Errorf("log_level = %q, want debug", got)
	}
	if _, ok := e["other.thing"]; ok {
		t.Errorf("wrong-prefix var leaked into entries")
	}
}

func TestSaveIsReadOnly(t *testing.T) {
	if err := env.New("RANKE_").Save(context.Background(), nil); err == nil {
		t.Fatal("expected read-only error from env.Save")
	}
}
