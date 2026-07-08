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
	"github.com/flocko-motion/rankedb/internal/core/access"
)

// Auth is the no-auth backend: every Authenticate returns the same subject.
type Auth struct{ subject string }

// New returns a no-auth backend that authenticates every request as the section's
// "subject" value. An empty or absent subject defaults to "anonymous" so
// downstream authorization always has a non-empty account to key on.
func New(ctx context.Context, cfg scope.Section) (*Auth, error) {
	subject := "anonymous"
	if cfg.HasValue("subject") {
		s, err := cfg.Get(ctx, "subject")
		if err != nil {
			return nil, err
		}
		if s != "" {
			subject = s
		}
	}
	return &Auth{subject: subject}, nil
}

// Authenticate ignores the token and returns the configured account with no
// caveats — the open backend never attenuates.
func (a *Auth) Authenticate(_ context.Context, _ string) (access.Principal, error) {
	return access.Principal{Account: a.subject}, nil
}

// Scheme reports the empty (NoAuth) scheme: this backend consumes no credential.
// Returned as a literal so noauth need not import the auth package that dispatches
// to it (which would cycle); the value equals auth.SchemeNone.
func (a *Auth) Scheme() string { return "" }
