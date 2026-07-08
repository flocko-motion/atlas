// package: sequencer / coordination
// type:    interface
// job:     the Sequencer port — rankedb's mechanism for advancing the branch-table head (B_h) under concurrency, keeping its history for rollback
// limits:  contract only; backends live in sub-packages. Wraps ranke-go's BranchTableHead, which only *holds* B_h (-> adapters/sequencer/...)
//
// Package sequencer defines rankedb's Sequencer. The foundation paper fixes what
// a branch-table head IS but leaves its MANAGEMENT open: ranke-go's
// BranchTableHead only loads, saves, and closes the single mutable Id. The
// Sequencer is what rankedb adds on top — the thing that *handles* B_h: it
// serialises concurrent contributions onto one authoritative head, and keeps the
// history of past heads so a claim that failed to persist can be rolled back to
// the last working state (paper 2 §Sequencer). It is deliberately a rankedb
// concern, not a library one: the library knows the mechanics of having a head,
// the server owns the policy of advancing it.
package sequencer

import (
	"context"

	"github.com/flocko-motion/ranke-go"
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
