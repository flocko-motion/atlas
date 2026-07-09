// package: core / orchestration
// type:    struct
// job:     the one request object that flows through the server, enriched at each stage
// limits:  transport-neutral; endpoints fill the ingress, core fills the rest (-> core.go, adapters/endpoints)
//
// Package core is the hexagon's center: it drives the driven ports (storage,
// sequencer, signer) and is driven by the driving ports (auth, endpoints). Its
// spine is one Request value that flows endpoint → auth → access → execution and
// back, gaining fields as it goes rather than being repackaged at each boundary.
// The endpoint fills the ingress fields from the wire; core fills the rest.
package core

import (
	"encoding/json"
	"fmt"

	"github.com/flocko-motion/rankedb/adapters/auth"
	"github.com/flocko-motion/rankedb/internal/core/access"
)

// Universe is the reserved branch for privileged by-head-id reads — re-exported
// from access so callers (endpoints) target it through core alone.
const Universe = access.Universe

// Operation is what a request asks the server to do. It fixes the access right the
// authorize stage checks (a right of 0 means the operation needs no grant — health
// and verification, per core-access) and selects the execute branch.
type Operation int

const (
	OpQuery              Operation = iota // read a filtered subgraph          (Read)
	OpContribute                          // merge claims onto a branch         (Contribute; branch-admin is C on the branch table)
	OpUpdate                              // overlay claims with newer versions (Update)
	OpDelete                              // delete claims                      (Delete)
	OpHead                                // a branch's current head id         (Read)
	OpClaim                               // one claim within a branch          (Read)
	OpContent                             // one content blob within a branch   (Read)
	OpUniverseClaim                       // one claim by id, privileged        (Read on $universe)
	OpUniverseContent                     // one blob by hash, privileged       (Read on $universe)
	OpHealth                              // liveness                           (no grant)
	OpLayers                              // list storage layers                (no grant)
	OpStartVerification                   // start a verification run           (no grant — verification needs none)
	OpVerifications                       // list verification runs             (no grant)
	OpVerification                        // one verification run               (no grant)
	OpCancelVerification                  // cancel a run                       (no grant)
	OpDeleteVerification                  // delete a run                       (no grant)
)

// Right is the access right this operation requires, or 0 when it needs no grant.
// The reads all require R; contribute/update/delete their CRUD letter; the
// operational and verification ops need none (verification's independence from
// grants is a core-access invariant).
func (o Operation) Right() access.Right {
	switch o {
	case OpQuery, OpHead, OpClaim, OpContent, OpUniverseClaim, OpUniverseContent:
		return access.Read
	case OpContribute:
		return access.Contribute
	case OpUpdate:
		return access.Update
	case OpDelete:
		return access.Delete
	}
	return 0
}

// Request is the single object that flows through the server, enriched at each
// stage. Its ingress fields are op-specific — an operation reads only the ones it
// needs — and the endpoint fills them from the wire.
type Request struct {
	// --- ingress: filled by the endpoint from the wire ---

	// Credential is the one auth token the transport extracted, or the zero value
	// if none was presented (→ NoAuth). An endpoint that finds more than one scheme
	// rejects the request itself, before building this.
	Credential auth.Credential
	// Op is what the caller wants; it fixes the required access right.
	Op Operation
	// Branch is the target branch, or access.Universe for a privileged by-id read.
	Branch string
	// Query is the read AST (OpQuery).
	Query *Query
	// Body is the signed-CBOR claims to merge (OpContribute).
	Body io.Reader
	// ClaimID targets one claim (OpClaim, OpUniverseClaim).
	ClaimID ranke.Id
	// Hash targets one content blob (OpContent, OpUniverseContent).
	Hash ranke.Id
	// VerConfig parameters a run (OpStartVerification).
	VerConfig *VerificationConfig
	// VerID targets one run (OpVerification, OpCancelVerification, OpDeleteVerification).
	VerID string

	// --- enrichment: filled by core as the request flows ---

	// Principal is who the request authenticated as (set by the authenticate stage).
	Principal access.Principal
	// Result is the operation's output, rendered back by the endpoint (set by
	// the execution stage).
	Result json.RawMessage
	// Report is the running execution trace, accumulated across stages.
	Report Report
}

// Report is the execution record that travels with the request: a human-readable
// trace now, and the home for the witnessed transaction window and per-stage
// counts as those stages are built.
type Report struct {
	Steps []string
}

// step appends one line to the running trace.
func (r *Report) step(format string, args ...any) {
	r.Steps = append(r.Steps, fmt.Sprintf(format, args...))
}
