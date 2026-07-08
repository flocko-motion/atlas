// package: sequencer / coordination
// type:    factory
// job:     build the sequencer from its config section, wired to storage and the signing identity
// limits:  stub — construction not yet implemented (-> adapters/sequencer)
//
// This file is the sequencer port's composition seam. In the bootstrap order the
// sequencer is built after its dependencies: the storage Universe it persists
// branch tables into, and the signing identity it attests each merge with.
package sequencer

import (
	"context"
	"crypto"
	"fmt"

	"github.com/flocko-motion/ranke-go"
	"github.com/flocko-motion/rankedb/config/scope"
)

// New builds the sequencer over the branch-table-head backend named in cfg,
// wired to the storage Universe and the signing identity. Stub: not yet
// implemented.
func New(ctx context.Context, cfg scope.Section, storage ranke.Universe, sig crypto.Signer) (Sequencer, error) {
	return nil, fmt.Errorf("sequencer: backend not yet implemented")
}
