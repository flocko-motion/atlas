// package: sequencer / coordination
// type:    interface + factory
// job:     the Sequencer port — rankedb's mechanism for advancing the branch-table head (B_h) — plus the factory that builds a backend
// limits:  contract + dispatch; wraps ranke-go's BranchTableHead, which only *holds* B_h (-> adapters/sequencer/...)
//
// Package sequencer defines rankedb's Sequencer and builds it from config. The
// foundation paper fixes what a branch-table head IS but leaves its MANAGEMENT
// open: ranke-go's BranchTableHead only loads, saves, and closes the single
// mutable Id. The Sequencer is what rankedb adds on top — the thing that
// *handles* B_h: it serialises concurrent contributions onto one authoritative
// head, and keeps the history of past heads so a claim that failed to persist can
// be rolled back to the last working state (paper 2 §Sequencer). It is
// deliberately a rankedb concern, not a library one: the library knows the
// mechanics of having a head, the server owns the policy of advancing it.
package sequencer

import (
	"context"
	"fmt"

	"github.com/flocko-motion/ranke-go"

	"github.com/flocko-motion/rankedb/adapters/signer"
	"github.com/flocko-motion/rankedb/config/scope"
)

// Sequencer advances and serves the branch-table head. After a contribution's
// claims are verified and persisted, it mints a new branch-table claim
// referencing the addition and the prior table, then advances B_h to it —
// retaining the previous heads so a failed write can be rolled back.
type Sequencer interface {
	// BTH returns the id of the latest branch table (n == 0) or a historical one
	// (n < 0), or a nil Id when the archive has no branches yet.
	BTH(ctx context.Context, n int) (ranke.Id, error)

	// BTHLen reports the length of the branch-table-head history.
	BTHLen(ctx context.Context) (int, error)

	// Add appends a new branch-table id to the top of the history, advancing B_h.
	Add(ctx context.Context, id ranke.Id) error
}

// New builds the sequencer over the branch-table-head backend named in cfg, wired
// to the storage Universe it persists branch tables into and the signing identity
// it attests each merge with (its bootstrap dependencies). Stub: not yet
// implemented.
func New(ctx context.Context, cfg scope.Section, storage ranke.Universe, sig signer.Signer) (Sequencer, error) {
	return nil, fmt.Errorf("sequencer: backend not yet implemented")
}
