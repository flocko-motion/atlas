// Package adapter wraps mstrYoda/goraphdb behind a minimal interface so a
// conformance harness (the openCypher TCK) can drive it like any Cypher engine.
//
// goraphdb is an embedded, file-backed graph DB. Each Engine owns a fresh
// on-disk database in a temp dir, so every TCK scenario gets an isolated graph.
package adapter

import (
	"context"
	"fmt"
	"os"

	graphdb "github.com/mstrYoda/goraphdb"
)

// Engine is one isolated graph database instance.
type Engine struct {
	db  *graphdb.DB
	dir string
}

// Result is the engine-agnostic shape the harness compares against expected
// TCK tables: ordered column names plus rows keyed by column.
type Result struct {
	Columns []string
	Rows    []map[string]any
}

// New opens a fresh goraphdb instance in a temporary directory.
func New() (*Engine, error) {
	dir, err := os.MkdirTemp("", "goraphdb-tck-*")
	if err != nil {
		return nil, err
	}
	db, err := graphdb.Open(dir, graphdb.DefaultOptions())
	if err != nil {
		os.RemoveAll(dir)
		return nil, fmt.Errorf("open goraphdb: %w", err)
	}
	return &Engine{db: db, dir: dir}, nil
}

// Run executes a Cypher query with optional parameters.
func (e *Engine) Run(ctx context.Context, query string, params map[string]any) (*Result, error) {
	var res *graphdb.CypherResult
	var err error
	if len(params) > 0 {
		res, err = e.db.CypherWithParams(ctx, query, params)
	} else {
		res, err = e.db.Cypher(ctx, query)
	}
	if err != nil {
		return nil, err
	}
	return &Result{Columns: res.Columns, Rows: res.Rows}, nil
}

// Snapshot is a coarse picture of graph state used to derive query side
// effects by diffing before/after. Label counts are distinct label names;
// Props is the total number of property entries across all nodes and edges.
type Snapshot struct {
	Nodes  int
	Rels   int
	Labels map[string]struct{}
	Props  int
}

// Snapshot captures current graph state. It pages through all nodes and edges,
// so it is O(graph) — fine for the small graphs TCK scenarios build.
func (e *Engine) Snapshot() Snapshot {
	s := Snapshot{Labels: map[string]struct{}{}}
	_ = e.db.ForEachNode(func(n *graphdb.Node) error {
		s.Nodes++
		s.Props += len(n.Props)
		for _, l := range n.Labels {
			s.Labels[l] = struct{}{}
		}
		return nil
	})
	var cursor graphdb.EdgeID
	for {
		page, err := e.db.ListEdges(cursor, 1000)
		if err != nil || page == nil {
			break
		}
		for _, ed := range page.Edges {
			s.Rels++
			s.Props += len(ed.Props)
		}
		if !page.HasMore || page.NextCursor == 0 {
			break
		}
		cursor = page.NextCursor
	}
	return s
}

// Close shuts the DB and deletes its on-disk files.
func (e *Engine) Close() error {
	err := e.db.Close()
	os.RemoveAll(e.dir)
	return err
}
