// package: coreapi / port
// type:    interface
// job:     the ranke-db API as a Go interface — the capability surface core exposes and endpoints call
// limits:  contract + its domain types only; core implements it, endpoint adapters call it
//
// Package coreapi is the boundary between the application core and the endpoint
// adapters (REST/HTTP, MCP/HTTP): core implements API, an adapter is handed it and
// calls it to serve each request. It lives in its own package so both sides can
// import it without a cycle — the endpoint dispatcher imports the adapters, the
// adapters import coreapi, and coreapi imports neither.
//
// API mirrors the openapi/openapi.yaml REST contract, one method per operation:
//
//	Query               POST   /query
//	Contribute          POST   /contribute?branch=
//	Head                GET    /{branch}/head
//	Claim               GET    /{branch}/claim/{id}
//	Content             GET    /{branch}/content/{hash}
//	UniverseClaim       GET    /$universe/claim/{id}
//	UniverseContent     GET    /$universe/content/{hash}
//	Health              GET    /health
//	Layers              GET    /system/layers
//	StartVerification   POST   /system/verification
//	Verifications       GET    /system/verification
//	Verification        GET    /system/verification/{id}
//	CancelVerification  POST   /system/verification/{id}/cancel
//	DeleteVerification  DELETE /system/verification/{id}
//
// Each call acts as the authenticated Subject and returns domain values — claims,
// ids, streams — or one of the sentinel errors below.
package coreapi

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/flocko-motion/ranke-go"
)

// API is the ranke-db capability surface: read, contribute, and operate.
type API interface {
	// Query runs query q and streams the claims it selects, in order.
	Query(ctx context.Context, subj Subject, q Query) (ResultStream, error)

	// Contribute adds the signed-CBOR claims in r to branch in one atomic step —
	// all or none — and returns the new head and their ids. Re-adding the same
	// claims is a no-op. ErrConflict if branch's head has moved on.
	Contribute(ctx context.Context, subj Subject, branch string, r io.Reader) (Contribution, error)

	// Head returns branch's current head id.
	Head(ctx context.Context, subj Subject, branch string) (ranke.Id, error)

	// Claim returns claim id if branch's closure holds it, else ErrNotFound.
	Claim(ctx context.Context, subj Subject, branch string, id ranke.Id) (ranke.Claim, error)

	// Content streams the blob hash if branch's closure holds it, else ErrNotFound.
	Content(ctx context.Context, subj Subject, branch string, hash ranke.Id) (io.ReadCloser, error)

	// UniverseClaim returns claim id from anywhere in the graph — the privileged
	// $universe read.
	UniverseClaim(ctx context.Context, subj Subject, id ranke.Id) (ranke.Claim, error)

	// UniverseContent streams the blob hash from anywhere in the graph — the
	// privileged $universe read.
	UniverseContent(ctx context.Context, subj Subject, hash ranke.Id) (io.ReadCloser, error)

	// Health reports liveness and the signing identity.
	Health(ctx context.Context) (Health, error)

	// Layers lists the storage layers by name and type.
	Layers(ctx context.Context, subj Subject) ([]StorageLayer, error)

	// StartVerification starts an asynchronous run and returns its running report
	// at once. ErrBusy if too many runs are already active.
	StartVerification(ctx context.Context, subj Subject, cfg VerificationConfig) (VerificationReport, error)

	// Verifications lists all runs, newest first.
	Verifications(ctx context.Context, subj Subject) ([]VerificationReport, error)

	// Verification returns one run's report, or ErrNotFound.
	Verification(ctx context.Context, subj Subject, id string) (VerificationReport, error)

	// CancelVerification stops a running run and keeps its report; a no-op on a run
	// that has already finished.
	CancelVerification(ctx context.Context, subj Subject, id string) (VerificationReport, error)

	// DeleteVerification removes a run and its report, or ErrNotFound.
	DeleteVerification(ctx context.Context, subj Subject, id string) error
}

// Subject is the authenticated account a call acts as.
type Subject string

// Sentinel errors, with the status an endpoint maps each to. A missing or invalid
// credential (401) is auth.ErrUnauthenticated, returned before a call reaches here.
var (
	// ErrNotFound — an unknown branch, claim, or content, or one outside the named
	// branch's closure; the two are indistinguishable (404).
	ErrNotFound = errors.New("coreapi: not found")
	// ErrForbidden — the subject may not do this (403).
	ErrForbidden = errors.New("coreapi: access denied")
	// ErrConflict — a contribution clashes with the branch's current head (409).
	ErrConflict = errors.New("coreapi: head conflict")
	// ErrNotImplemented — the request needs an optional capability the stack does
	// not offer (501).
	ErrNotImplemented = errors.New("coreapi: capability not configured")
	// ErrBusy — too many verification runs are active to start another (429).
	ErrBusy = errors.New("coreapi: verification run limit reached")
)

// --- Query -----------------------------------------------------------------

// Query is a declarative read: generate a set of claims (Select), filter it
// (Where), shape each result (Output), then order and bound the read.
type Query struct {
	Select    Select
	Where     *Where // nil = no filter
	Output    Output
	Order     *Order // nil = default order, (created_at, id)
	Limit     Limit
	Execution Execution
}

// Select is a generator: a starting point and a traversal. Branch roots at a
// branch's current head; Claim optionally roots at a claim id within it; Claim
// with an empty Branch is the privileged $universe read. An empty Path follows
// every edge outward to the full closure.
type Select struct {
	Branch string
	Claim  ranke.Id
	Path   []PathStep
}

// PathStep follows typed edges to a bounded depth, optionally constraining the
// endpoint node types. A leading "-" on an Edges or Nodes entry excludes that type.
type PathStep struct {
	Edges []string
	Dir   Direction // default DirProvenance
	Depth int       // max hops; 0 = unbounded for this step
	Nodes []string
}

// Direction is which way a step follows an edge.
type Direction string

const (
	DirProvenance  Direction = "provenance"  // outgoing (default)
	DirUses        Direction = "uses"        // incoming
	DirConnections Direction = "connections" // either
)

// Where is a boolean tree of comparisons. Exactly one of And, Or, Not, or a leaf
// (Field + Test) is set on any node.
type Where struct {
	And   []Where
	Or    []Where
	Not   *Where
	Field string      // leaf: the field tested
	Test  *Comparison // leaf: the comparison on Field
}

// Comparison tests one field with exactly one operator.
type Comparison struct {
	Eq   any
	Ne   any
	Lt   any
	Le   any
	Gt   any
	Ge   any
	In   []any  // set membership
	Glob string // shell-style wildcard
}

// Output shapes each result.
type Output struct {
	Detail   Detail
	Content  int64    // max inlined content bytes per claim; 0 = none
	Overflow Overflow // how content past Content is handled
}

// Detail sets how much each result carries.
type Detail string

const (
	DetailID    Detail = "id"    // just the id
	DetailClaim Detail = "claim" // the reached claim (default)
	DetailPath  Detail = "path"  // the whole route to it
)

// Overflow is how content larger than Output.Content is handled.
type Overflow string

const (
	OverflowCutoff    Overflow = "cutoff"    // truncate at the cap
	OverflowOmit      Overflow = "omit"      // drop the content
	OverflowReference Overflow = "reference" // return a hash stub in its place
)

// Order is a named sort; without it, results order by (created_at, id).
type Order struct {
	Field string
	Desc  bool
}

// Limit bounds a read.
type Limit struct {
	Results int           // max claims; 0 = unbounded
	Time    time.Duration // execution budget; 0 = none
}

// Execution selects where the query runs and whether it reports on itself.
type Execution struct {
	Layer  string // pin to one named storage layer (see Layers); empty = ranke-db chooses
	Report bool   // when true, the stream ends with a QueryReport
}

// ResultStream streams query results, one at a time, in the query's order. After
// Next returns false, check Err; Report is non-nil only when Execution.Report was set.
type ResultStream interface {
	// Next advances to the next result, returning false at end of stream or error.
	Next() bool
	// Result returns the current result (valid after Next returned true).
	Result() QueryResult
	// Report returns the query's report, available after Next returns false when
	// Execution.Report was set; nil otherwise.
	Report() *QueryReport
	// Err returns the first error that stopped the stream, if any.
	Err() error
	// Close releases the stream's resources.
	Close() error
}

// QueryResult is one reached claim, shaped per Output.
type QueryResult struct {
	Claim ranke.Claim
	// Path is the full route to the claim, set only when Output.Detail is DetailPath.
	Path []ranke.Claim
	// Content is the claim's inlined content, present only when Output.Content > 0;
	// truncated per Output.Overflow when it exceeds the cap.
	Content []byte
}

// QueryReport ends a stream when Execution.Report is set — a report of how the
// query ran.
type QueryReport struct {
	Engine    string        // which engine ran the query
	Layer     string        // which storage layer answered
	Lowered   string        // the query as that engine ran it
	Elapsed   time.Duration // wall-clock time
	Results   int           // items emitted
	Truncated bool          // whether Limit cut the read short
}

// --- Contribute / operate --------------------------------------------------

// Contribution is the outcome of a merge: the new head and the contributed ids.
type Contribution struct {
	Head ranke.Id
	Ids  []ranke.Id
}

// Health is liveness plus the contributor identity the stack signs merges with.
type Health struct {
	Status string // "ok" when serving
	Signer string // signing/contributor identity (e.g. "ed25519:…")
}

// StorageLayer names one storage layer — name and type only, by design.
type StorageLayer struct {
	Name string
	Type string // backend kind: mem, fs, sqlite, s3, postgres, neo4j, …
}

// --- Verification ----------------------------------------------------------

// VerificationConfig parameters a run.
type VerificationConfig struct {
	Closure          string            // root — a branch name or a head id
	Layer            string            // optional single storage layer to check directly
	Depth            VerificationDepth //
	ContentThreshold int64             // for full-content, max bytes checked per claim; 0 = all
}

// VerificationDepth is how thoroughly a run checks each claim.
type VerificationDepth string

const (
	DepthCompleteness      VerificationDepth = "completeness"       // every referenced claim is present
	DepthRecordCorrectness VerificationDepth = "record-correctness" // ids and signatures check out
	DepthFullContent       VerificationDepth = "full-content"       // content matches its hash too
)

// RunStatus is the lifecycle state of a verification run.
type RunStatus string

const (
	RunRunning  RunStatus = "running"
	RunComplete RunStatus = "complete"
	RunStopped  RunStatus = "stopped" // cancelled; partial findings kept
	RunError    RunStatus = "error"
)

// VerificationReport is a point-in-time record of a run.
type VerificationReport struct {
	ID            string
	Config        VerificationConfig
	Head          ranke.Id // the head the run pinned at start
	Status        RunStatus
	StartedAt     time.Time
	CompletedAt   time.Time // zero while running
	ClaimsChecked int64
	BytesRead     int64
	OK            bool // true when no failures were found
	Failures      []VerificationFailure
}

// VerificationFailure is one integrity problem a run found.
type VerificationFailure struct {
	ID     ranke.Id
	Mode   FailureMode
	Layer  string // where it was found
	Detail string
}

// FailureMode classifies an integrity problem.
type FailureMode string

const (
	FailureCorruptBytes   FailureMode = "corrupt-bytes"   // stored bytes do not match their hash
	FailureInvalidContent FailureMode = "invalid-content" // the claim itself does not validate
)
