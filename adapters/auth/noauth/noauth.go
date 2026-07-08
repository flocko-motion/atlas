// package: noauth / authn
// type:    adapter
// job:     authenticate every request as one fixed subject — the no-auth backend
// limits:  no credential checking; for single-tenant/dev stacks (-> auth.New)
//
// Package noauth is the open authentication backend: it ignores the request
// credential and returns one configured subject for every call. It suits a
// stack with no external auth — a single operator, a dev box, a trusted network
// — where authorization still keys on the returned subject.
package noauth

import (
	"context"

	"github.com/flocko-motion/rankedb/config/scope"
)

// Auth is the no-auth backend: every Authenticate returns the same subject.
type Auth struct{ subject string }

// New returns a no-auth backend that authenticates every request as the scope's
// "subject" value. An empty subject defaults to "anonymous" so downstream
// authorization always has a non-empty account to key on.
func New(cfg scope.Config) *Auth {
	subject := cfg.String("subject")
	if subject == "" {
		subject = "anonymous"
	}
	return &Auth{subject: subject}
}

// Authenticate ignores credential and returns the configured subject.
func (a *Auth) Authenticate(_ context.Context, _ string) (string, error) {
	return a.subject, nil
}
