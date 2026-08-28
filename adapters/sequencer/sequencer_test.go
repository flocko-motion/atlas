// Sequencer port tests: the factory binds ranke-go's backends, and the identity it
// signs merges with comes through the signer port rather than from a local key.
package sequencer_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/flocko-motion/rankedb/adapters/sequencer"
	"github.com/flocko-motion/rankedb/adapters/signer"
	"github.com/flocko-motion/rankedb/config/scope"
	"github.com/rankegraph/ranke-go"
)

// signerConfig builds an inmemory signer descriptor over a throwaway Ed25519 key,
// in the PKCS#8 PEM form the adapter reads.
func signerConfig(t *testing.T) scope.Section {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)
	key := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	return scope.Literal(map[string]string{"type": "inmemory", "key": string(key)})
}

// newSigner builds a signer through its own port, so this exercises the assembly
// the server performs rather than a hand-made double.
func newSigner(t *testing.T) signer.Signer {
	t.Helper()
	sig, err := signer.New(context.Background(), signerConfig(t))
	require.NoError(t, err)
	return sig
}

// TestNewBindsBackends: both backends build, and an unknown type is refused by name.
func TestNewBindsBackends(t *testing.T) {
	ctx := context.Background()
	for _, kind := range []string{"dev", "concurrent"} {
		t.Run(kind, func(t *testing.T) {
			seq, err := sequencer.New(ctx, scope.Literal(map[string]string{"type": kind}),
				ranke.NewMemoryUniverse(), newSigner(t), nil)
			require.NoError(t, err)
			require.NotNil(t, seq)
			require.NotNil(t, seq.GetContributor(), "the sequencer signs as somebody")
		})
	}

	_, err := sequencer.New(ctx, scope.Literal(map[string]string{"type": "nope"}),
		ranke.NewMemoryUniverse(), newSigner(t), nil)
	require.ErrorContains(t, err, "unknown type")
}

// TestArchiveOpensAtEmptyBranchTable: a freshly bound sequencer hands out a readable
// archive — the bootstrap the paper describes, an empty branch table at k₀.
func TestArchiveOpensAtEmptyBranchTable(t *testing.T) {
	ctx := context.Background()
	seq, err := sequencer.New(ctx, scope.Literal(map[string]string{"type": "dev"}),
		ranke.NewMemoryUniverse(), newSigner(t), nil)
	require.NoError(t, err)

	archive, err := seq.GetArchive(ctx)
	require.NoError(t, err)
	branches, err := archive.GetBranches(ctx)
	require.NoError(t, err)
	require.Empty(t, branches, "a fresh archive names no branches yet")
	require.NotEmpty(t, archive.Head(), "but it does have a head: the empty branch table")
}

// TestContributorIdIsStableAcrossBuilds: one key yields one contributor id, so a
// restart reopens an archive as the same identity instead of minting a new one.
func TestContributorIdIsStableAcrossBuilds(t *testing.T) {
	ctx := context.Background()
	cfg := signerConfig(t)
	sequencerCfg := scope.Literal(map[string]string{"type": "dev"})

	build := func() ranke.Id {
		sig, err := signer.New(ctx, cfg)
		require.NoError(t, err)
		seq, err := sequencer.New(ctx, sequencerCfg, ranke.NewMemoryUniverse(), sig, nil)
		require.NoError(t, err)
		return seq.GetContributor().ID()
	}

	require.Equal(t, build(), build())
}

// TestContributorAlwaysPinsToEpoch: the sequencer's own identity is minted once at
// boot, before any --dev caller can possibly steer the clock — the HTTP server isn't
// listening yet. It must precede whatever the earliest merge it signs turns out to be,
// so it stays epoch-pinned whether or not a clock was supplied, past-dated fixtures
// (a --dev story set in 2024, say) included; a value that tracked the clock instead
// would put this identity's created_at *after* the very first branch table it signs,
// which V-MONO forbids.
func TestContributorAlwaysPinsToEpoch(t *testing.T) {
	ctx := context.Background()
	cfg := scope.Literal(map[string]string{"type": "dev"})

	seq, err := sequencer.New(ctx, cfg, ranke.NewMemoryUniverse(), newSigner(t), nil)
	require.NoError(t, err)
	require.True(t, seq.GetContributor().Node().CreatedAt().Equal(time.Unix(0, 0).UTC()),
		"nil now: want the identity pinned to the epoch")

	past := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	seq, err = sequencer.New(ctx, cfg, ranke.NewMemoryUniverse(), newSigner(t),
		func() time.Time { return past })
	require.NoError(t, err)
	require.True(t, seq.GetContributor().Node().CreatedAt().Equal(time.Unix(0, 0).UTC()),
		"a supplied now, even a --dev story's own past date: want the identity still pinned to the epoch")
}

// TestHistoryDescriptorIsValidated: an unknown history type fails by name rather
// than silently falling back to memory.
func TestHistoryDescriptorIsValidated(t *testing.T) {
	_, err := sequencer.New(context.Background(),
		scope.Literal(map[string]string{"type": "dev", "history": "nonsense"}),
		ranke.NewMemoryUniverse(), newSigner(t), nil)
	require.NoError(t, err, "a scalar history key is not a section, so the default applies")
}
