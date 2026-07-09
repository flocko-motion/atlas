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

// Sentinel errors the pipeline returns, with the status an endpoint maps each to.
// A missing or invalid credential (401) is auth.ErrUnauthenticated, returned by
// the authenticate stage before these apply.
var (
	// ErrForbidden — an authenticated principal lacks a grant for the action (403).
	// Distinct from auth.ErrUnauthenticated: identity was established, authority not.
	ErrForbidden = errors.New("core: forbidden")
	// ErrNotFound — an unknown branch, claim, or content, or one outside the named
	// branch's closure; the two are indistinguishable (404).
	ErrNotFound = errors.New("core: not found")
	// ErrConflict — a contribution clashes with the branch's current head (409).
	ErrConflict = errors.New("core: head conflict")
	// ErrBusy — too many verification runs are active to start another (429).
	ErrBusy = errors.New("core: verification run limit reached")
	// ErrNotImplemented — a capability the stack does not (yet) offer (501). Every
	// execute path returns it until the engine behind it is built.
	ErrNotImplemented = errors.New("core: not implemented")
)

// Category is a transport-neutral, machine-readable classification of a failure.
// Core never names an HTTP status; an endpoint maps a Category to its transport's
// status (an HTTP code, an MCP error) and echoes the Category as the response's
// machine-readable code. Defined once here so every transport classifies alike.
type Category string

const (
	CatUnauthenticated Category = "unauthenticated" // no/invalid credential
	CatForbidden       Category = "forbidden"       // authenticated but ungranted
	CatNotFound        Category = "not_found"       // unknown/out-of-scope branch, claim, or content
	CatConflict        Category = "conflict"        // contribution clashes with the head
	CatBusy            Category = "busy"            // too many active runs
	CatInvalid         Category = "invalid"         // malformed request
	CatUnimplemented   Category = "unimplemented"   // capability not configured
	CatInternal        Category = "internal"        // anything else
)

// Categorize classifies an error into its Category by walking the sentinel chain,
// so an endpoint has one place to translate to its transport. A nil error yields
// the empty category (callers only classify a non-nil error).
func Categorize(err error) Category {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, auth.ErrUnauthenticated):
		return CatUnauthenticated
	case errors.Is(err, auth.ErrAmbiguousCredentials):
		return CatInvalid
	case errors.Is(err, ErrForbidden):
		return CatForbidden
	case errors.Is(err, ErrNotFound):
		return CatNotFound
	case errors.Is(err, ErrConflict):
		return CatConflict
	case errors.Is(err, ErrBusy):
		return CatBusy
	case errors.Is(err, ErrNotImplemented):
		return CatUnimplemented
	default:
		return CatInternal
	}
}

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

// Handle runs a request through the pipeline: authenticate → authorize → execute,
// enriching the request in place and returning the response as a Stream. A
// pre-stream failure (auth, access, bad input) comes back as the error with no
// stream; success returns a Stream the endpoint frames. The Report shows how far
// it got.
func (c *Core) Handle(ctx context.Context, req *Request) (Stream, error) {
	if err := c.authenticate(ctx, req); err != nil {
		return nil, err
	}
	if err := c.authorize(req); err != nil {
		return nil, err
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

// authorize confirms the principal holds the operation's right on the branch. An
// operation whose Right is 0 needs no grant (health, verification) and passes on
// identity alone.
func (c *Core) authorize(req *Request) error {
	right := req.Op.Right()
	if right == 0 {
		req.Report.step("no grant required")
		return nil
	}
	if !c.access.Allow(req.Principal, right, req.Branch) {
		return fmt.Errorf("%w: %q may not %c %s", ErrForbidden, req.Principal.Account, right, req.Branch)
	}
	req.Report.step("authorized %c on %s", right, req.Branch)
	return nil
}

// execute runs the operation against the ports and returns its response stream.
// Scaffold: the query, contribution, read, and verification engines over storage
// and the sequencer land here, each producing a Stream of Items.
func (c *Core) execute(ctx context.Context, req *Request) (Stream, error) {
	req.Report.step("execute %v: not yet implemented", req.Op)
	return nil, ErrNotImplemented
}
