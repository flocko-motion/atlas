package storage

import (
	"context"
	"testing"

	"github.com/flocko-motion/rankedb/config/scope"
)

// TestParseSize covers the human-readable size suffixes the maxContentSize field
// accepts.
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

// TestBuildRedis covers the redis wiring: client construction is local (no
// dial), so this needs no reachable redis instance.
func TestBuildRedis(t *testing.T) {
	if _, err := New(context.Background(), scope.Literal(map[string]string{"type": "redis"})); err == nil {
		t.Error("New(redis without addr) = nil error, want error")
	}
	if _, err := New(context.Background(), scope.Literal(map[string]string{
		"type": "redis", "addr": "127.0.0.1:6379", "db": "not-a-number",
	})); err == nil {
		t.Error("New(redis, db=not-a-number) = nil error, want error")
	}
	u, err := New(context.Background(), scope.Literal(map[string]string{
		"type": "redis", "addr": "127.0.0.1:6379", "password": "secret", "db": "3",
	}))
	if err != nil {
		t.Fatalf("New(redis): %v", err)
	}
	if u == nil {
		t.Fatal("New(redis) returned a nil Universe")
	}
}

// TestBuildS3 covers the s3 wiring: New only fails for a nil client or empty
// bucket, so an unreachable endpoint still builds — probeCaps swallows a probe
// failure into all-false capabilities rather than erroring construction.
func TestBuildS3(t *testing.T) {
	required := map[string]string{
		"type": "s3", "bucket": "b", "region": "us-east-1",
		"accessKeyId": "ak", "secretAccessKey": "sk",
	}
	for _, missing := range []string{"bucket", "region", "accessKeyId", "secretAccessKey"} {
		partial := map[string]string{}
		for k, v := range required {
			if k != missing {
				partial[k] = v
			}
		}
		if _, err := New(context.Background(), scope.Literal(partial)); err == nil {
			t.Errorf("New(s3 without %s) = nil error, want error", missing)
		}
	}
	full := map[string]string{}
	for k, v := range required {
		full[k] = v
	}
	full["endpoint"] = "http://127.0.0.1:1" // refused instantly, no live service needed
	full["usePathStyle"] = "true"
	u, err := New(context.Background(), scope.Literal(full))
	if err != nil {
		t.Fatalf("New(s3): %v", err)
	}
	if u == nil {
		t.Fatal("New(s3) returned a nil Universe")
	}
}

// TestBuildNeo4j covers the neo4j wiring: NewDriverWithContext validates the
// URI scheme and pools connections lazily, so this needs no reachable instance.
func TestBuildNeo4j(t *testing.T) {
	if _, err := New(context.Background(), scope.Literal(map[string]string{"type": "neo4j"})); err == nil {
		t.Error("New(neo4j without uri) = nil error, want error")
	}
	u, err := New(context.Background(), scope.Literal(map[string]string{
		"type": "neo4j", "uri": "bolt://127.0.0.1:7687",
	}))
	if err != nil {
		t.Fatalf("New(neo4j, no auth): %v", err)
	}
	if u == nil {
		t.Fatal("New(neo4j, no auth) returned a nil Universe")
	}
	u, err = New(context.Background(), scope.Literal(map[string]string{
		"type": "neo4j", "uri": "bolt://127.0.0.1:7687",
		"username": "neo4j", "password": "secret", "database": "ranke",
	}))
	if err != nil {
		t.Fatalf("New(neo4j): %v", err)
	}
	if u == nil {
		t.Fatal("New(neo4j) returned a nil Universe")
	}
}
