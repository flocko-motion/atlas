// package: core / orchestration
// type:    struct
// job:     the one request object that flows through the server, enriched at each stage
// limits:  transport-neutral; endpoints fill the ingress, core fills the rest (-> core.go, adapters/endpoints)
//
// Package core is the hexagon's center: it drives the driven ports (storage,
// sequencer, signer) and is driven by the driving ports (auth, endpoints). Its
// spine is one Request value that flows endpoint → auth → access → execution; the
// response comes back as a Stream (see stream.go, core.go). The endpoint fills the
// ingress fields from the wire; core fills the enrichment.
package core

import (
	"fmt"
	"io"

	"github.com/flocko-motion/ranke-go"

	"github.com/flocko-motion/rankedb/adapters/auth"
	"github.com/flocko-motion/rankedb/internal/core/access"
)

// The reserved pseudo-branches, re-exported from access so a request targets them
// through core alone (Branch = core.Universe / core.Branches). Universe is the
// privileged by-id read across the whole graph; Branches is the branch table.
const (
	Universe = access.Universe
	Branches = access.Branches
)

// Operation is what a request asks the server to do, named Subject+Verb so it
// sorts hierarchically and disambiguates (OpClaimDelete vs OpVerificationDelete).
// It fixes the access right the authorize stage checks — 0 means no grant is
// required (health, verification) — and selects the execute branch. A read of a
// claim by id is one op whether the branch is a name or $universe; the pseudo-
// branch generalises the scope, so there is no separate universe op.
type Operation int

const (
	OpClaimQuery         Operation = iota // query claims                           (Read)
	OpClaimGet                            // one claim by id (branch or $universe)   (Read)
	OpClaimContent                        // one claim's content                    (Read)
	OpClaimContribute                     // merge claims onto a branch             (Contribute; branch-admin is C on the branch table)
	OpClaimDelete                         // purge claims — physical removal, not a mutation (Delete)
	OpBranchHead                          // a branch's current head id             (Read)
	OpBranchList                          // list the branch table's branches       (Read on $branches)
	OpLayerList                           // list storage layers (name + type)      (no grant)
	OpLayerInfo                           // runtime info on one storage layer      (no grant)
	OpHealthGet                           // liveness                               (no grant)
	OpVerificationStart                   // start a verification run               (no grant — verification needs none)
	OpVerificationList                    // list verification runs                 (no grant)
	OpVerificationGet                     // one verification run                   (no grant)
	OpVerificationCancel                  // cancel a run                           (no grant)
	OpVerificationDelete                  // delete a run                           (no grant)
)

// Right is the access right this operation requires, or 0 when it needs no grant.
// The reads require R; contribute/update/delete their CRUD letter; the operational
// and verification ops need none (verification's independence from grants is a
// core-access invariant).
func (o Operation) Right() access.Right {
	switch o {
	case OpClaimQuery, OpClaimGet, OpClaimContent, OpBranchHead, OpBranchList:
		return access.Read
	case OpClaimContribute:
		return access.Contribute
	case OpClaimDelete:
		return access.Delete
	}
	return 0
}

// Request is the single object that flows through the server, enriched at each
// stage. Its ingress fields are op-specific — an operation reads only the ones it
// needs — and the endpoint fills them from the wire. The response is not a field:
// Handle returns a Stream.
type Request struct {
	// --- ingress: filled by the endpoint from the wire ---

	// Credential is the one auth token the transport extracted, or the zero value
	// if none was presented (→ NoAuth). An endpoint that finds more than one scheme
	// rejects the request itself, before building this.
	Credential auth.Credential
	// Op is what the caller wants; it fixes the required access right.
	Op Operation
	// Branch is the target branch, or Universe for a privileged by-id read.
	Branch string
	// Query is the read AST (OpClaimQuery).
	Query *Query
	// Body is the signed-CBOR claims to merge (OpClaimContribute).
	Body io.Reader
	// ClaimID targets one claim (OpClaimGet, OpClaimContent) — content is addressed
	// by the claim that holds it, so it inherits the claim's branch/access scope and
	// hides whether the bytes are inline or an external blob.
	ClaimID ranke.Id
	// VerConfig parameters a run (OpVerificationStart).
	VerConfig *VerificationConfig
	// VerID targets one run (OpVerificationGet, OpVerificationCancel, OpVerificationDelete).
	VerID string

	// --- enrichment: filled by core as the request flows ---

	// Principal is who the request authenticated as (set by the authenticate stage).
	Principal access.Principal
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
