// package: sequencer / coordination
// type:    factory
// job:     the Sequencer port — build the ranke-go sequencer backend named in a config section
// limits:  wiring only; advancing the head, the six merge steps and the head history are ranke-go's (-> github.com/flocko-motion/ranke-go)
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

// New builds the backend named by the section's "type".
func New(ctx context.Context, cfg scope.Section, storage ranke.Universe, sig signer.Signer) (Sequencer, error) {
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

	switch t {
	case "dev":
		return dev.NewSequencer(ctx, storage, hist, self, systemClock{})
	case "concurrent":
		return concurrent.NewSequencer(ctx, storage, hist, self, systemClock{})
	default:
		return nil, fmt.Errorf("sequencer: unknown type %q (want dev or concurrent)", t)
	}
}

// buildHistory builds the head timeline. A file history is what lets a restart reopen
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
// epoch so the id follows from the key alone; a launch time would mint a new identity
// every start.
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

// systemClock stamps the claims the sequencer mints with wall-clock time.
type systemClock struct{}

// Tick returns now, in UTC.
func (systemClock) Tick() time.Time { return time.Now().UTC() }
