// package: mem / store
// type:    adapter
// job:     hold access grants and the disabled set in memory (an access.Store)
// limits:  no persistence — nothing survives a restart; for tests and dev only (-> postgres)
package mem

import (
	"context"
	"sync"

	"rankedb/access"
)

// New returns an empty in-memory store.
func New() *Store {
	return &Store{
		disabled: map[string]bool{},
		grants:   map[string][]access.Grant{},
	}
}

// Store is an in-memory access.Store guarded by a single mutex.
type Store struct {
	mu       sync.Mutex
	disabled map[string]bool
	grants   map[string][]access.Grant // keyed by subject
}

// Disabled reports whether a subject is disabled (false for an unknown subject).
func (s *Store) Disabled(_ context.Context, subject string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.disabled[subject], nil
}

// GrantsFor returns a copy of all grants held by a subject.
func (s *Store) GrantsFor(_ context.Context, subject string) ([]access.Grant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	src := s.grants[subject]
	out := make([]access.Grant, len(src))
	copy(out, src)
	return out, nil
}

// PutGrant adds the (subject, scope, role) grant idempotently. Grants are
// additive: a subject may hold several roles on one scope, so an existing role
// on the same scope is left intact.
func (s *Store) PutGrant(_ context.Context, g access.Grant) error {
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
func (s *Store) DeleteGrant(_ context.Context, subject string, scope access.Scope, role access.Role) error {
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

// SetDisabled marks a subject disabled (test/dev helper; not part of access.Store).
func (s *Store) SetDisabled(subject string, disabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.disabled[subject] = disabled
}
