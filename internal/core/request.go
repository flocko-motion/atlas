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

// Operation is what a request asks the server to do. Each operation maps to the
// single CRUD right the access checker must confirm before it runs.
type Operation int

const (
	OpQuery      Operation = iota // read a filtered subgraph
	OpContribute                  // merge new claims onto a branch (creating/hiding branches too — a C on the branch table)
	OpUpdate                      // overlay existing claims with newer versions
	OpDelete                      // delete claims
)

// Right is the access right this operation requires. Branch admin (create/hide)
// is not distinct: it is an OpContribute whose target is the branch table, so it
// maps to Contribute like any other write.
func (o Operation) Right() access.Right {
	switch o {
	case OpQuery:
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
// stage. Fields are grouped by the stage that fills them.
type Request struct {
	// --- ingress: filled by the endpoint from the wire ---

	// Credential is the one auth token the transport extracted, or the zero value
	// if none was presented (→ NoAuth). An endpoint that finds more than one scheme
	// rejects the request itself, before building this.
	Credential auth.Credential
	// Op is what the caller wants; it fixes the required access right.
	Op Operation
	// Branch is the target branch (or access.Universe for a privileged head read).
	Branch string
	// Payload is the operation's opaque body — a query tree or the claims to
	// contribute — parsed by the execution stage, not here.
	Payload json.RawMessage

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
