// package: core / orchestration
// type:    orchestrator
// job:     run a Request through the pipeline — authenticate, authorize, execute — driving the ports
// limits:  the composition of the ports, assembled by config; execution is a scaffold stub (-> config, adapters/*)
//
// Core is what the endpoints drive and what drives storage, the sequencer and the
// signer. Handle is the pipeline: it authenticates the request's credentials to a
// Principal (auth), authorizes the Principal for the operation's right on its
// branch (access), then executes against the ports. Each stage enriches the same
// Request and records a line in its Report, so the object that comes back carries
// both the result and the trace of how it got there.
package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/flocko-motion/rankedb/adapters/auth"
	"github.com/flocko-motion/rankedb/adapters/sequencer"
	"github.com/flocko-motion/rankedb/adapters/storage"
	"github.com/flocko-motion/rankedb/internal/core/access"
)

// ErrForbidden reports that an authenticated principal lacks a grant for the
// requested action — an endpoint maps it to 403. Distinct from
// auth.ErrUnauthenticated (401): identity was established, authority was not.
var ErrForbidden = errors.New("core: forbidden")

// ErrNotImplemented marks a pipeline stage that is still a scaffold.
var ErrNotImplemented = errors.New("core: not implemented")

// Core composes the ports into the request pipeline. It is assembled once by
// config and shared by every endpoint.
type Core struct {
	auth   *auth.Set
	access *access.Checker
	seq    sequencer.Sequencer
	store  storage.Storage
}

// New assembles the core from the ports config built.
func New(a *auth.Set, chk *access.Checker, seq sequencer.Sequencer, store storage.Storage) *Core {
	return &Core{auth: a, access: chk, seq: seq, store: store}
}

// Handle runs a request through the pipeline, enriching it in place: authenticate
// → authorize → execute. It stops at the first stage that fails, leaving the
// Report showing how far it got.
func (c *Core) Handle(ctx context.Context, req *Request) error {
	if err := c.authenticate(ctx, req); err != nil {
		return err
	}
	if err := c.authorize(req); err != nil {
		return err
	}
	return c.execute(ctx, req)
}

// authenticate resolves the request's credentials to a Principal.
func (c *Core) authenticate(ctx context.Context, req *Request) error {
	p, err := c.auth.Authenticate(ctx, req.Credential)
	if err != nil {
		return err
	}
	req.Principal = p
	req.Report.step("authenticated as %q", p.Account)
	return nil
}

// authorize confirms the principal holds the operation's right on the branch.
func (c *Core) authorize(req *Request) error {
	right := req.Op.Right()
	if !c.access.Allow(req.Principal, right, req.Branch) {
		return fmt.Errorf("%w: %q may not %c %s", ErrForbidden, req.Principal.Account, right, req.Branch)
	}
	req.Report.step("authorized %c on %s", right, req.Branch)
	return nil
}

// execute runs the operation against the ports. Scaffold: the query and
// contribution engines over storage and the sequencer land here.
func (c *Core) execute(ctx context.Context, req *Request) error {
	req.Report.step("execute %v: not yet implemented", req.Op)
	return ErrNotImplemented
}
