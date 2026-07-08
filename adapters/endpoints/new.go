// package: endpoints / transport
// type:    factory
// job:     build one endpoint from its config section, wired to the sequencer and the authenticators
// limits:  stub — construction not yet implemented (-> adapters/endpoints)
//
// This file is the endpoint port's composition seam. In the bootstrap order an
// endpoint is built last: it is wired to the sequencer it drives and the full
// set of authenticators it may accept (a request may carry a JWT here, a
// macaroon there).
package endpoints

import (
	"context"
	"fmt"

	"github.com/flocko-motion/rankedb/adapters/auth"
	"github.com/flocko-motion/rankedb/adapters/sequencer"
	"github.com/flocko-motion/rankedb/config/scope"
)

// New builds an endpoint of the transport named in cfg, wired to the sequencer
// it drives and the authenticators it may accept. Stub: not yet implemented.
func New(ctx context.Context, cfg scope.Section, seq sequencer.Sequencer, auths []auth.Auth) (Endpoints, error) {
	return nil, fmt.Errorf("endpoints: transport not yet implemented")
}
