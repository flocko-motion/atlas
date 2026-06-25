// package: mem / store
// type:    adapter
// job:     hold (subject,scope,role) grants and the disabled set in memory (a grants.Store)
// limits:  no persistence — nothing survives a restart; for tests and dev only (-> postgres)
package mem

import (
	"context"
	"sync"

	"rankedb/adapter/grants"
)

// New returns an empty in-memory store.
func New() *Store {
	return &Store{
		disabled: map[string]bool{},
		grants:   map[string][]grants.Grant{},
	}
}

// Store is an in-memory grants.Store guarded by a single mutex.
type Store struct {
	mu       sync.Mutex
	disabled map[string]bool
	grants   map[string][]grants.Grant // keyed by subject
}

// Disabled reports whether a subject is disabled (false for an unknown subject).
func (s *Store) Disabled(_ context.Context, subject string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.disabled[subject], nil
}

// GrantsFor returns a copy of all grants held by a subject.
func (s *Store) GrantsFor(_ context.Context, subject string) ([]grants.Grant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	src := s.grants[subject]
	out := make([]grants.Grant, len(src))
	copy(out, src)
	return out, nil
}

// PutGrant adds the (subject, scope, role) grant idempotently. Grants are
// additive: a subject may hold several roles on one scope, so an existing role
// on the same scope is left intact.
func (s *Store) PutGrant(_ context.Context, g grants.Grant) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.grants[g.Subject]
	for _, e := range list {
		if e.Scope == g.Scope && e.Role == g.Role { // already present
			return nil
		}
	}
	s.grants[g.Subject] = append(list, g)
	return nil
}

// DeleteGrant removes the (subject, scope, role) grant if present (a no-op
// otherwise), leaving any other roles on that scope intact.
func (s *Store) DeleteGrant(_ context.Context, subject string, scope grants.Scope, role grants.Role) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.grants[subject]
	for i, e := range list {
		if e.Scope == scope && e.Role == role {
			s.grants[subject] = append(list[:i], list[i+1:]...)
			return nil
		}
	}
	return nil
}

// SetDisabled marks a subject disabled (test/dev helper; not part of grants.Store).
func (s *Store) SetDisabled(subject string, disabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.disabled[subject] = disabled
}
