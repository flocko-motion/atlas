// package: sequencer / coordination
// type:    factory
// job:     the Sequencer port — build the ranke-go sequencer backend named in a config section
// limits:  wiring only; advancing the head, the six merge steps and the head history are ranke-go's (-> github.com/flocko-motion/ranke-go)
//
// Package sequencer is ranke-db's sequencer port. The Sequencer is a Ranke-
// Archive's single writer: it advances the head k → k′ and keeps the history of
// past heads so a claim that failed to persist can be rolled back (paper 2
// §Sequencer). That mechanism now lives in ranke-go — together with the narrow
// head-history port the paper describes (ranke.History, with in-memory and
// file backends) — so the port type here IS the library's contract and this
// package only selects a backend from the configuration and hands it its
// dependencies: the storage Universe it persists branch tables into, and the
// signing identity it attests each merge with.
package sequencer

import (
	"context"
	"fmt"

	"github.com/flocko-motion/ranke-go"

	"github.com/flocko-motion/rankedb/adapters/signer"
	"github.com/flocko-motion/rankedb/config/scope"
)

// Sequencer is the sequencer port's product: ranke-go's Sequencer contract. The
// server reaches the archive through it — immutable snapshots to read from, and
// merges that advance the head to write.
type Sequencer = ranke.Sequencer

// New builds the sequencer backend named by the section's "type", wired to the
// storage Universe it persists branch tables into and the signing identity it
// attests each merge with.
//
// No backend binds yet. ranke-go's writers today are adapter/sequencer/dev — a
// deliberately serial reference writer that mints a fresh k₀ on every
// construction, so it cannot reopen an existing archive — and
// adapter/sequencer/simple, an empty placeholder; neither implements
// ranke.Sequencer, whose write path (NewContribution + Merge) is still marked a
// draft upstream. The full sequencer is in the works there, and this factory is
// where it binds: one case, no core changes.
func New(ctx context.Context, cfg scope.Section, storage ranke.Universe, sig signer.Signer) (Sequencer, error) {
	if !cfg.HasValue("type") {
		return nil, fmt.Errorf("sequencer: missing type")
	}
	t, err := cfg.Get(ctx, "type")
	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("sequencer: no backend yet for type %q — ranke-go's sequencer is being completed upstream; omit the section to launch read-only", t)
}
