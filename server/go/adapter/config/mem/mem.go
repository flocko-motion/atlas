// package: mem / adapter
// type:    adapter
// job:     in-memory config Store for tests, examples, and as the assembler's scratch overlay target
// limits:  non-persistent — nothing survives a process restart (-> file/postgres for durable stores)
//
// Package mem is an in-memory config Store — for tests, examples, and as the
// scratch target the assembler can fold overlays into. Nothing persists
// across process restarts.
package mem

import (
	"context"
	"sync"

	"rankedb/adapter/config"
)

// New returns an empty in-memory config Store.
func New() config.Store { return &store{m: config.Entries{}} }

// NewFrom returns an in-memory Store seeded with a copy of e.
func NewFrom(e config.Entries) config.Store {
	m := make(config.Entries, len(e))
	for k, v := range e {
		m[k] = v
	}
	return &store{m: m}
}

type store struct {
	mu sync.Mutex
	m  config.Entries
}

func (s *store) Load(context.Context) (config.Entries, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(config.Entries, len(s.m))
	for k, v := range s.m {
		out[k] = v
	}
	return out, nil
}

func (s *store) Save(_ context.Context, e config.Entries) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := make(config.Entries, len(e))
	for k, v := range e {
		m[k] = v
	}
	s.m = m
	return nil
}

func (s *store) Close() error { return nil }
