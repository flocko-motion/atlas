package adapter

import (
	"context"
	"testing"
)

// TestSmoke confirms goraphdb behaves as its README claims: create, match,
// filter, traverse. This is the sanity check before running the full TCK.
func TestSmoke(t *testing.T) {
	ctx := context.Background()
	e, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer e.Close()

	steps := []struct {
		name    string
		query   string
		wantRow int // -1 = don't check
	}{
		{"create alice", `CREATE (n:Person {name: "Alice", age: 30}) RETURN n`, 1},
		{"create bob", `CREATE (n:Person {name: "Bob", age: 25}) RETURN n`, 1},
		{"match all", `MATCH (n) RETURN n`, 2},
		{"match label", `MATCH (n:Person) RETURN n`, 2},
		{"where filter", `MATCH (n) WHERE n.age > 25 RETURN n`, 1},
		{"prop filter", `MATCH (n {name: "Alice"}) RETURN n`, 1},
	}
	for _, s := range steps {
		res, err := e.Run(ctx, s.query, nil)
		if err != nil {
			t.Errorf("%s: query %q errored: %v", s.name, s.query, err)
			continue
		}
		if s.wantRow >= 0 && len(res.Rows) != s.wantRow {
			t.Errorf("%s: %q got %d rows, want %d (cols=%v rows=%v)",
				s.name, s.query, len(res.Rows), s.wantRow, res.Columns, res.Rows)
		} else {
			t.Logf("%s: ok (%d rows, cols=%v)", s.name, len(res.Rows), res.Columns)
		}
	}
}
