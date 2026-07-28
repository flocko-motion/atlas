package rest_http

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
	"time"

	ranke "github.com/flocko-motion/ranke-go"

	"github.com/flocko-motion/rankedb/openapi"
)

// TestRankeQuery drives the wire→RQL mapping from actual request JSON, since that
// is the only thing a client sends. Each case pins a translation the wire schema
// and ranke-go's RQL do not share outright: the wire's folded `detail`, its
// bool|int|string content cap, its sequence framing, its single sort key, its
// boolean-tree union, and the unconfined read it spells as a claim without a
// branch.
func TestRankeQuery(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want func(*testing.T, ranke.Query)
	}{
		{
			name: "branch roots the scope",
			body: `{"select": {"branch": "foo"}}`,
			want: func(t *testing.T, q ranke.Query) {
				if q.Select.Branch != "foo" {
					t.Fatalf("branch = %q, want foo", q.Select.Branch)
				}
			},
		},
		{
			name: "a claim without a branch is the unconfined read",
			body: `{"select": {"claim": "` + testClaimID + `"}}`,
			want: func(t *testing.T, q ranke.Query) {
				if q.Select.Branch != ranke.BranchUniverse {
					t.Fatalf("branch = %q, want %q", q.Select.Branch, ranke.BranchUniverse)
				}
				if q.Select.Claim == nil {
					t.Fatal("claim id not carried")
				}
			},
		},
		{
			name: "path depth is a max-hop bound",
			body: `{"select": {"branch": "foo", "path": [{"edges": ["cites"], "dir": "uses", "depth": 3, "nodes": ["source/*"]}]}}`,
			want: func(t *testing.T, q ranke.Query) {
				want := ranke.PathStep{Edges: []string{"cites"}, Dir: ranke.DirUses, Max: 3, Nodes: []string{"source/*"}}
				if !reflect.DeepEqual(q.Select.Path, []ranke.PathStep{want}) {
					t.Fatalf("path = %+v, want %+v", q.Select.Path, want)
				}
			},
		},
		{
			name: "detail path unfolds into a shape",
			body: `{"select": {"branch": "foo"}, "output": {"detail": "path"}}`,
			want: func(t *testing.T, q ranke.Query) {
				if q.Output.Shape != ranke.ShapePath || q.Output.Detail != ranke.DetailClaims {
					t.Fatalf("shape/detail = %q/%q, want path/claims", q.Output.Shape, q.Output.Detail)
				}
			},
		},
		{
			name: "detail id carries identities only",
			body: `{"select": {"branch": "foo"}, "output": {"detail": "id"}}`,
			want: func(t *testing.T, q ranke.Query) {
				if q.Output.Detail != ranke.DetailID || q.Output.Shape != "" {
					t.Fatalf("shape/detail = %q/%q, want /id", q.Output.Shape, q.Output.Detail)
				}
			},
		},
		{
			name: "content false carries none",
			body: `{"select": {"branch": "foo"}, "output": {"content": false}}`,
			want: func(t *testing.T, q ranke.Query) {
				if q.Output.Content != nil {
					t.Fatalf("content = %+v, want nil", q.Output.Content)
				}
			},
		},
		{
			name: "content true is uncapped",
			body: `{"select": {"branch": "foo"}, "output": {"content": true}}`,
			want: func(t *testing.T, q ranke.Query) {
				if q.Output.Content == nil || q.Output.Content.Max != math.MaxInt64 {
					t.Fatalf("content = %+v, want max", q.Output.Content)
				}
			},
		},
		{
			name: "content number is bytes, with overflow",
			body: `{"select": {"branch": "foo"}, "output": {"content": 4096, "overflow": "reference"}}`,
			want: func(t *testing.T, q ranke.Query) {
				if q.Output.Content == nil || q.Output.Content.Max != 4096 {
					t.Fatalf("content = %+v, want 4096", q.Output.Content)
				}
				if q.Output.Content.Overflow != ranke.OverflowReference {
					t.Fatalf("overflow = %q, want reference", q.Output.Content.Overflow)
				}
			},
		},
		{
			name: "content string is a human size",
			body: `{"select": {"branch": "foo"}, "output": {"content": "4kb"}}`,
			want: func(t *testing.T, q ranke.Query) {
				if q.Output.Content == nil || q.Output.Content.Max != 4096 {
					t.Fatalf("content = %+v, want 4096", q.Output.Content)
				}
			},
		},
		{
			name: "cbor-seq framing selects the verifiable claim form",
			body: `{"select": {"branch": "foo"}, "output": {"encoding": "cbor-seq"}}`,
			want: func(t *testing.T, q ranke.Query) {
				if q.Output.Encoding != ranke.ResultCBOR {
					t.Fatalf("encoding = %q, want cbor", q.Output.Encoding)
				}
			},
		},
		{
			name: "one sort key becomes the key list",
			body: `{"select": {"branch": "foo"}, "order": {"field": "created_at", "dir": "desc"}}`,
			want: func(t *testing.T, q ranke.Query) {
				want := []ranke.OrderKey{{Field: "created_at", Dir: ranke.SortDesc}}
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
			name: "report true grades to info",
			body: `{"select": {"branch": "foo"}, "execution": {"layer": "hot", "report": true}}`,
			want: func(t *testing.T, q ranke.Query) {
				if q.Execution.Layer != "hot" || q.Execution.Report != ranke.ReportInfo {
					t.Fatalf("execution = %+v, want hot/info", q.Execution)
				}
			},
		},
		{
			name: "a field map is a leaf",
			body: `{"select": {"branch": "foo"}, "where": {"type": {"glob": "source/*"}}}`,
			want: func(t *testing.T, q ranke.Query) {
				if q.Where == nil || q.Where.Field != "type" || q.Where.Test == nil || q.Where.Test.Glob != "source/*" {
					t.Fatalf("where = %+v, want type glob leaf", q.Where)
				}
			},
		},
		{
			name: "several fields conjoin in a stable order",
			body: `{"select": {"branch": "foo"}, "where": {"type": {"eq": "source"}, "author": {"eq": "ada"}}}`,
			want: func(t *testing.T, q ranke.Query) {
				if q.Where == nil || len(q.Where.And) != 2 {
					t.Fatalf("where = %+v, want a 2-leaf conjunction", q.Where)
				}
				if q.Where.And[0].Field != "author" || q.Where.And[1].Field != "type" {
					t.Fatalf("fields = %q,%q, want author,type (sorted)", q.Where.And[0].Field, q.Where.And[1].Field)
				}
			},
		},
		{
			name: "and/or/not nest",
			body: `{"select": {"branch": "foo"}, "where": {"and": [
				{"or": [{"type": {"eq": "source"}}, {"type": {"eq": "derivation"}}]},
				{"not": {"author": {"eq": "ada"}}}
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
			body: `{"select": {"branch": "foo"}, "where": {"height": {"in": [1, 2, 3]}}}`,
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
		{"invalid claim id", `{"select": {"claim": "not-an-id"}}`},
		{"unknown detail", `{"select": {"branch": "foo"}, "output": {"detail": "everything"}}`},
		{"unknown encoding", `{"select": {"branch": "foo"}, "output": {"encoding": "yaml"}}`},
		{"unparseable size", `{"select": {"branch": "foo"}, "output": {"content": "4 furlongs"}}`},
		{"unparseable duration", `{"select": {"branch": "foo"}, "limit": {"time": "soon"}}`},
		{"where neither tree nor map", `{"select": {"branch": "foo"}, "where": {"and": []}}`},
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
