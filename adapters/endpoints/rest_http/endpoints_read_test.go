package rest_http

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	ranke "github.com/flocko-motion/ranke-go"

	"github.com/flocko-motion/rankedb/openapi"
)

// TestRankeQuery drives the wire→RQL mapping from actual request JSON, since that
// is the only thing a client sends. The wire binds ranke-go's RQL field-for-field,
// so what each case pins is that the binding is *faithful*: every axis of the
// language reaches the library intact, with none folded away or unreachable, and
// the two rules the JSON schema cannot state (a scope is mandatory, $universe needs
// a head) enforced at the boundary.
func TestRankeQuery(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want func(*testing.T, ranke.Query)
	}{
		{
			name: "a branch name is the scope",
			body: `{"select": {"branch": "foo"}}`,
			want: func(t *testing.T, q ranke.Query) {
				if q.Select.Branch != "foo" {
					t.Fatalf("branch = %q, want foo", q.Select.Branch)
				}
			},
		},
		{
			name: "the archive scope reads across branches",
			body: `{"select": {"branch": "$archive"}}`,
			want: func(t *testing.T, q ranke.Query) {
				if q.Select.Branch != ranke.BranchArchive {
					t.Fatalf("branch = %q, want %q", q.Select.Branch, ranke.BranchArchive)
				}
			},
		},
		{
			name: "the universe scope carries its head",
			body: `{"select": {"branch": "$universe", "head": "` + testClaimID + `"}}`,
			want: func(t *testing.T, q ranke.Query) {
				if q.Select.Branch != ranke.BranchUniverse {
					t.Fatalf("branch = %q, want %q", q.Select.Branch, ranke.BranchUniverse)
				}
				if q.Select.Head == nil {
					t.Fatal("head id not carried")
				}
			},
		},
		{
			name: "head pins the closure and claim the start",
			body: `{"select": {"branch": "foo", "head": "` + testClaimID + `", "claim": "` + testClaimID + `"}}`,
			want: func(t *testing.T, q ranke.Query) {
				if q.Select.Head == nil || q.Select.Claim == nil {
					t.Fatalf("head/claim = %v/%v, want both carried", q.Select.Head, q.Select.Claim)
				}
			},
		},
		{
			name: "a step carries its edges, direction, hop range and nodes",
			body: `{"select": {"branch": "foo", "path": [{"edges": ["derivation/*"], "dir": "uses", "min": 0, "max": 3, "nodes": ["source/*"]}]}}`,
			want: func(t *testing.T, q ranke.Query) {
				want := ranke.PathStep{
					Edges: []string{"derivation/*"},
					Dir:   ranke.DirUses,
					Min:   ranke.Hops(0),
					Max:   3,
					Nodes: []string{"source/*"},
				}
				if !reflect.DeepEqual(q.Select.Path, []ranke.PathStep{want}) {
					t.Fatalf("path = %+v, want %+v", q.Select.Path, want)
				}
			},
		},
		{
			name: "an absent min leaves the library's one-hop default",
			body: `{"select": {"branch": "foo", "path": [{"edges": ["derivation/*"]}]}}`,
			want: func(t *testing.T, q ranke.Query) {
				if got := q.Select.Path[0].MinHops(); got != 1 {
					t.Fatalf("min hops = %d, want 1", got)
				}
			},
		},
		{
			name: "the output axes stay orthogonal",
			body: `{"select": {"branch": "foo"}, "output": {"shape": "path", "detail": "graph", "form": "original", "encoding": "cbor"}}`,
			want: func(t *testing.T, q ranke.Query) {
				want := ranke.Output{
					Shape:    ranke.ShapePath,
					Detail:   ranke.DetailGraph,
					Form:     ranke.FormOriginal,
					Encoding: ranke.ResultCBOR,
				}
				if !reflect.DeepEqual(q.Output, want) {
					t.Fatalf("output = %+v, want %+v", q.Output, want)
				}
			},
		},
		{
			name: "an output with no content axis maps cleanly",
			body: `{"select": {"branch": "foo"}, "output": {}}`,
			want: func(t *testing.T, q ranke.Query) {
				if !reflect.DeepEqual(q.Output, ranke.Output{}) {
					t.Fatalf("output = %+v, want the zero value", q.Output)
				}
			},
		},
		{
			name: "sort keys keep their priority order and collation",
			body: `{"select": {"branch": "foo"}, "order": [{"field": "height", "compare": "numeric", "dir": "desc"}, {"field": "type"}]}`,
			want: func(t *testing.T, q ranke.Query) {
				want := []ranke.OrderKey{
					{Field: "height", Compare: ranke.CompareNumeric, Dir: ranke.SortDesc},
					{Field: "type"},
				}
				if !reflect.DeepEqual(q.Order, want) {
					t.Fatalf("order = %+v, want %+v", q.Order, want)
				}
			},
		},
		{
			name: "limit bounds results and time",
			body: `{"select": {"branch": "foo"}, "limit": {"results": 10, "time": "5s"}}`,
			want: func(t *testing.T, q ranke.Query) {
				if q.Limit.Results != 10 || q.Limit.Time != 5*time.Second {
					t.Fatalf("limit = %+v, want 10/5s", q.Limit)
				}
			},
		},
		{
			name: "report carries the asked-for verbosity",
			body: `{"select": {"branch": "foo"}, "execution": {"layer": "hot", "report": "trace"}}`,
			want: func(t *testing.T, q ranke.Query) {
				if q.Execution.Layer != "hot" || q.Execution.Report != ranke.ReportTrace {
					t.Fatalf("execution = %+v, want hot/trace", q.Execution)
				}
			},
		},
		{
			name: "a field and its test are a leaf",
			body: `{"select": {"branch": "foo"}, "where": {"field": "type", "test": {"glob": "source/*"}}}`,
			want: func(t *testing.T, q ranke.Query) {
				if q.Where == nil || q.Where.Field != "type" || q.Where.Test == nil || q.Where.Test.Glob != "source/*" {
					t.Fatalf("where = %+v, want a type glob leaf", q.Where)
				}
			},
		},
		{
			name: "and/or/not nest",
			body: `{"select": {"branch": "foo"}, "where": {"and": [
				{"or": [{"field": "type", "test": {"eq": "source"}}, {"field": "type", "test": {"eq": "derivation"}}]},
				{"not": {"field": "author", "test": {"eq": "ada"}}}
			]}}`,
			want: func(t *testing.T, q ranke.Query) {
				if q.Where == nil || len(q.Where.And) != 2 {
					t.Fatalf("where = %+v, want a 2-arm conjunction", q.Where)
				}
				if len(q.Where.And[0].Or) != 2 {
					t.Fatalf("first arm = %+v, want a 2-arm disjunction", q.Where.And[0])
				}
				not := q.Where.And[1].Not
				if not == nil || not.Field != "author" {
					t.Fatalf("second arm = %+v, want not(author)", q.Where.And[1])
				}
			},
		},
		{
			name: "in and the scalar operators pass through",
			body: `{"select": {"branch": "foo"}, "where": {"field": "height", "test": {"in": [1, 2, 3]}}}`,
			want: func(t *testing.T, q ranke.Query) {
				if q.Where == nil || q.Where.Test == nil || len(q.Where.Test.In) != 3 {
					t.Fatalf("where = %+v, want a 3-value set", q.Where)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var wire openapi.Query
			if err := json.Unmarshal([]byte(tc.body), &wire); err != nil {
				t.Fatalf("decode wire body: %v", err)
			}
			q, err := rankeQuery(wire)
			if err != nil {
				t.Fatalf("rankeQuery: %v", err)
			}
			tc.want(t, q)
		})
	}
}

// TestRankeQueryRejects covers the wire values that cannot be honoured. They are
// rejected at the boundary so the engine never sees a half-understood query.
func TestRankeQueryRejects(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"no scope", `{"select": {}}`},
		{"universe without a head", `{"select": {"branch": "$universe"}}`},
		{"invalid head id", `{"select": {"branch": "foo", "head": "not-an-id"}}`},
		{"invalid claim id", `{"select": {"branch": "foo", "claim": "not-an-id"}}`},
		{"unparseable duration", `{"select": {"branch": "foo"}, "limit": {"time": "soon"}}`},
		{"where neither tree nor leaf", `{"select": {"branch": "foo"}, "where": {"and": []}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var wire openapi.Query
			if err := json.Unmarshal([]byte(tc.body), &wire); err != nil {
				t.Fatalf("decode wire body: %v", err)
			}
			if q, err := rankeQuery(wire); err == nil {
				t.Fatalf("rankeQuery accepted %s → %+v, want a refusal", tc.body, q)
			}
		})
	}
}

// testClaimID is a syntactically valid claim id — the mapping only parses it.
const testClaimID = "bafyreib2rxk3rybk3aobmv5cjuql3bm2twh4jo5uxgnrmtqjbwmjnzqxvi"
