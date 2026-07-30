// package: core / orchestration
// type:    orchestrator
// job:     run a Request through the pipeline — authenticate, authorize, execute — driving the ports
// limits:  the composition of the ports, assembled by config (-> config, adapters/*)
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
	"github.com/flocko-motion/rankedb/adapters/signer"
	"github.com/flocko-motion/rankedb/adapters/storage"
	"github.com/flocko-motion/rankedb/internal/core/access"
)

// Sentinel errors the pipeline returns, with the status an endpoint maps each to. A
// bad credential (401) is auth.ErrUnauthenticated, raised before these apply.
var (
	// ErrForbidden — an authenticated principal lacks a grant for the action (403).
	// Distinct from auth.ErrUnauthenticated: identity was established, authority not.
	ErrForbidden = errors.New("core: forbidden")
	// ErrInvalidRequest — the request is malformed: a body that does not decode, or a
	// field the bound engine cannot carry (400).
	ErrInvalidRequest = errors.New("core: invalid request")
	// ErrNotFound — unknown, or outside the named scope's closure; indistinguishable (404).
	ErrNotFound = errors.New("core: not found")
	// ErrConflict — a contribution clashes with the branch's current head (409).
	ErrConflict = errors.New("core: head conflict")
	// ErrBusy — too many verification runs are active to start another (429).
	ErrBusy = errors.New("core: verification run limit reached")
	// ErrNotImplemented — a capability the stack does not yet offer (501).
	ErrNotImplemented = errors.New("core: not implemented")
)

// Category classifies a failure transport-neutrally: core names no HTTP status, an
// endpoint maps the Category to its own and echoes it as the machine-readable code.
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

// Categorize walks the sentinel chain to a Category, so an endpoint translates in one
// place. A nil error yields the empty category.
func Categorize(err error) Category {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, auth.ErrUnauthenticated):
		return CatUnauthenticated
	case errors.Is(err, auth.ErrAmbiguousCredentials):
		return CatInvalid
	case errors.Is(err, ErrInvalidRequest):
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
	signer signer.Signer
	// layerInfo names the storage layers by name and type — all config retains of
	// them, and all layer introspection reports.
	layerInfo []StorageLayer
	// runs is the verification-run registry — the identity and history the library's
	// live handle has none of.
	runs *registry
}

// New assembles the core from the ports config built.
func New(a *auth.Set, chk *access.Checker, seq sequencer.Sequencer, store storage.Storage, opts ...Option) *Core {
	c := &Core{auth: a, access: chk, seq: seq, store: store, runs: newRegistry(0, nil)}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Option supplies what only some stacks have: a signer to report an identity from, the
// layer names config parsed. A core assembled without them still authenticates,
// authorizes and reads.
type Option func(*Core)

// WithSigner binds the signing identity health reports.
func WithSigner(s signer.Signer) Option { return func(c *Core) { c.signer = s } }

// WithLayers binds the storage layers, name and type only, that introspection reports.
func WithLayers(layers []StorageLayer) Option {
	return func(c *Core) { c.layerInfo = layers }
}

// WithMaxVerificationRuns bounds how many verification runs may walk at once. Beyond it
// a start is refused as busy; the server never stops a run to make room.
func WithMaxVerificationRuns(n int) Option {
	return func(c *Core) { c.runs = newRegistry(n, nil) }
}

// Handle runs a request through authenticate → authorize → execute, enriching it in
// place. A pre-stream failure returns the error alone; success returns a Stream.
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

// authorize confirms the principal holds the operation's right on the branch; a Right
// of 0 needs no grant and passes on identity alone.
func (c *Core) authorize(req *Request) error {
	right := req.Op.Right()
	if right == 0 {
		req.Report.step("no grant required")
		return nil
	}
	if req.Op == OpClaimContribute {
		// A contribution names the branch each claim joins, and may name several, so
		// its grants are checked once the body is decoded — the same shape as the
		// cross-branch delete rule, which is core calling Allow per branch.
		req.Report.step("authorized per branch once the body is read")
		return nil
	}
	return c.allow(req, right, req.Branch)
}

// allow checks one grant and records it.
func (c *Core) allow(req *Request, right access.Right, branch string) error {
	if !c.access.Allow(req.Principal, right, branch) {
		return fmt.Errorf("%w: %q may not %c %s", ErrForbidden, req.Principal.Account, right, branch)
	}
	req.Report.step("authorized %c on %s", right, branch)
	return nil
}
