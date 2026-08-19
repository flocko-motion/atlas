// package: sequencer / coordination
// type:    factory
// job:     the Sequencer port — build the ranke-go sequencer backend named in a config section
// limits:  wiring only; the head, merges and history are ranke-go's (-> ranke-go)
//
// The single writer advances the head k → k′ and keeps past heads for rollback (paper 2
// §Sequencer). That mechanism is ranke-go's; this package only picks a backend.
package sequencer

import (
	"context"
	"crypto"
	"fmt"
	"io"
	"time"

	"github.com/flocko-motion/ranke-go"
	historyfile "github.com/flocko-motion/ranke-go/adapter/history/file"
	historymem "github.com/flocko-motion/ranke-go/adapter/history/mem"
	"github.com/flocko-motion/ranke-go/adapter/sequencer/concurrent"
	"github.com/flocko-motion/ranke-go/adapter/sequencer/dev"

	"github.com/flocko-motion/rankedb/adapters/signer"
	"github.com/flocko-motion/rankedb/config/scope"
)

// Sequencer is the sequencer port's product: ranke-go's Sequencer contract. The
// server reaches the archive through it — immutable snapshots to read from, and
// merges that advance the head to write.
type Sequencer = ranke.Sequencer

// New builds the backend named by the section's "type". now is the time source it mints
// claims from; nil defaults to the wall clock. Both dev.NewSequencer and
// concurrent.NewSequencer already accept any Clock — real time is this package's own
// choice, not something either backend requires — so a caller wanting a different source
// (a steerable one, for --dev) supplies now instead of leaving it nil.
func New(ctx context.Context, cfg scope.Section, storage ranke.Universe, sig signer.Signer, now func() time.Time) (Sequencer, error) {
	if !cfg.HasValue("type") {
		return nil, fmt.Errorf("sequencer: missing type")
	}
	t, err := cfg.Get(ctx, "type")
	if err != nil {
		return nil, err
	}

	hist, err := buildHistory(ctx, cfg)
	if err != nil {
		return nil, err
	}
	self, err := contributor(ctx, storage, sig)
	if err != nil {
		return nil, err
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	clock := clockFunc(now)

	switch t {
	case "dev":
		return dev.NewSequencer(ctx, storage, hist, self, clock)
	case "concurrent":
		return concurrent.NewSequencer(ctx, storage, hist, self, clock)
	default:
		return nil, fmt.Errorf("sequencer: unknown type %q (want dev or concurrent)", t)
	}
}

// buildHistory builds the head timeline. A file timeline is what lets a restart reopen
// an archive rather than bootstrap a fresh one.
func buildHistory(ctx context.Context, cfg scope.Section) (ranke.History, error) {
	if !cfg.HasSection("history") {
		return historymem.New(), nil
	}
	sec := cfg.GetSection("history")
	t, err := sec.Get(ctx, "type")
	if err != nil {
		return nil, fmt.Errorf("sequencer: history: %w", err)
	}
	switch t {
	case "mem":
		return historymem.New(), nil
	case "file":
		path, err := sec.Get(ctx, "path")
		if err != nil {
			return nil, fmt.Errorf("sequencer: history: %w", err)
		}
		return historyfile.New(path)
	default:
		return nil, fmt.Errorf("sequencer: history: unknown type %q (want mem or file)", t)
	}
}

// contributor mints the identity merges are signed as. created_at is pinned to the
// epoch, always — not just for id stability across restarts, but because this identity
// is minted once at boot, before any --dev caller can possibly steer the clock (the
// HTTP server isn't listening yet); it must precede whatever the earliest merge it will
// ever sign turns out to be, and epoch is the one instant guaranteed to.
func contributor(ctx context.Context, u ranke.Universe, sig signer.Signer) (ranke.Contributor, error) {
	pub, err := sig.Public(ctx)
	if err != nil {
		return nil, fmt.Errorf("sequencer: signer public key: %w", err)
	}
	encoded, err := ranke.EncodePublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("sequencer: encode public key: %w", err)
	}
	key := portKey{ctx: ctx, sig: sig, pub: pub}
	claim, err := ranke.NewClaim(ranke.NodeContributor, nil).
		WithInlineContent(encoded).
		WithEncoding(ranke.EncodingOctetStream).
		WithCreatedAt(time.Unix(0, 0).UTC()).
		Sign(key)
	if err != nil {
		return nil, fmt.Errorf("sequencer: sign contributor claim: %w", err)
	}
	return claim.AsContributor(ctx, u, key)
}

// portKey adapts the signer port to crypto.Signer, keeping signing a call: a Transit
// key never leaves the vault. It carries a context because crypto.Signer takes none.
type portKey struct {
	ctx context.Context
	sig signer.Signer
	pub crypto.PublicKey
}

// Public returns the public half of the identity.
func (k portKey) Public() crypto.PublicKey { return k.pub }

// Sign signs the digest through the port; the backends hold their own entropy.
func (k portKey) Sign(_ io.Reader, digest []byte, _ crypto.SignerOpts) ([]byte, error) {
	return k.sig.Sign(k.ctx, digest)
}

// clockFunc adapts a bare function to dev.Clock/concurrent.Clock — identically shaped
// (Tick() time.Time) in both, so one adapter serves either backend.
type clockFunc func() time.Time

// Tick reads the next timestamp.
func (f clockFunc) Tick() time.Time { return f() }
