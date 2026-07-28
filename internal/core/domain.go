// package: core / orchestration
// type:    domain types
// job:     the capability surface's domain values — the server's own: merge outcome, liveness, storage introspection, verification runs
// limits:  types only; the query language and its results are ranke-go's (-> github.com/flocko-motion/ranke-go)
//
// These are the typed values a Request carries in and out. What they are NOT is
// the graph's vocabulary: the RQL query AST and its result/report shapes were
// drafted here while ranke-go lacked them, and ranke-go now owns them — a Request
// carries a *ranke.Query and the engine answers with a ranke.ResultStream. What
// remains here is genuinely the server's: the outcome of a merge, liveness, the
// storage layers it composed, and the verification-run model it manages.
package core

import (
	"io"
	"time"

	"github.com/flocko-motion/ranke-go"
)

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

// Content streams a blob's bytes; the read side of Contribute's io.Reader body.
type Content = io.ReadCloser

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
