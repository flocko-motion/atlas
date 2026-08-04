// package: main / cmd
// type:    logic
// job:     the fixture identity and the graph shapes it signs
// limits:  builds claims only; delivering them is the client's (-> client.go)
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/flocko-motion/ranke-go"
)

// batches is one slice of claims per contribution, merged in order, so the branch
// table grows a revision per batch.
type batches [][]ranke.Claim

// identityDomain separates derived fixture keys from anything else derived from a name.
const identityDomain = "ranke-db/generator/contributor/"

// epoch is the fixture clock's start. Pinned, not time.Now(): an id covers created_at,
// so a fixed clock makes the same command yield the same ids.
var epoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

// clockStep advances the fixture clock per claim, so created_at orders the build.
const clockStep = time.Minute

// grower signs claims for one identity, remembering them so later claims can cite them.
type grower struct {
	self      ranke.Contributor
	selfClaim ranke.Claim
	rnd       *rand.Rand
	at        time.Time
	made      []made
}

// made pairs a signed claim with the height a claim citing it must climb past.
type made struct {
	claim  ranke.Claim
	height uint64
}

// newGrower derives the fixture contributor named by `as` and binds its signing key.
func newGrower(ctx context.Context, as string) (*grower, error) {
	claim, priv, err := identity(as)
	if err != nil {
		return nil, err
	}
	// No Universe: this contributor's pubkey is inline, so nothing needs fetching.
	self, err := claim.AsContributor(ctx, nil, priv)
	if err != nil {
		return nil, fmt.Errorf("bind contributor %q: %w", as, err)
	}
	digest := sha256.Sum256([]byte(as))
	return &grower{
		self:      self,
		selfClaim: claim,
		rnd:       rand.New(rand.NewPCG(uint64(digest[0]), uint64(digest[1]))),
		at:        epoch,
	}, nil
}

// identity derives a contributor from a name, so a fixture is reproducible. The key
// comes from a public string and is worth nothing: never an application's.
func identity(as string) (ranke.Claim, ed25519.PrivateKey, error) {
	seed := sha256.Sum256([]byte(identityDomain + as))
	priv := ed25519.NewKeyFromSeed(seed[:])
	pub, err := ranke.EncodePublicKey(priv.Public())
	if err != nil {
		return nil, nil, fmt.Errorf("encode public key: %w", err)
	}
	// The root contributor claim names no contributor: it is its own (§4.3).
	claim, err := ranke.NewClaim(ranke.NodeTypeContributor, nil).
		WithInlineContent(pub).
		WithEncoding(ranke.EncodingOctetStream).
		WithCreatedAt(epoch).
		Sign(priv)
	if err != nil {
		return nil, nil, fmt.Errorf("sign contributor claim: %w", err)
	}
	return claim, priv, nil
}

// example is the smallest graph with real provenance: two sources, a derivation citing
// both, an entity from that derivation.
func (g *grower) example() (batches, error) {
	first, err := g.claim("source/note", "a first note")
	if err != nil {
		return nil, err
	}
	second, err := g.claim("source/note", "a second note")
	if err != nil {
		return nil, err
	}
	derivation, err := g.claim("derivation/extraction", "Ada Lovelace", first, second)
	if err != nil {
		return nil, err
	}
	entity, err := g.claim("entity/person", "Ada Lovelace", derivation)
	if err != nil {
		return nil, err
	}
	return batches{{first.claim, second.claim, derivation.claim, entity.claim}}, nil
}

// chainTypes is the mix a grown archive gets — sources dominate, as they do in life.
var chainTypes = []string{
	"source/note",
	"source/note",
	"source/record",
	"derivation/extraction",
	"entity/person",
	"relation/mention",
}

// chain grows an archive contribution by contribution, each citing what came before:
// climbing heights, a branch table of many revisions, references that reach back.
func (g *grower) chain(contributions, per int) (batches, error) {
	out := make(batches, 0, contributions)
	for i := range contributions {
		batch := make([]ranke.Claim, 0, per)
		for j := range per {
			typ := chainTypes[(i*per+j)%len(chainTypes)]
			// A source enters the archive, so it cites nothing; the rest draw on what is there.
			var refs []made
			if !isSource(typ) {
				refs = g.pick(maxReferences)
			}
			m, err := g.claim(typ, fmt.Sprintf("%s %d.%d", typ, i+1, j+1), refs...)
			if err != nil {
				return nil, err
			}
			batch = append(batch, m.claim)
		}
		out = append(out, batch)
	}
	return out, nil
}

// isSource reports whether a type is in the source class, whose claims are roots.
func isSource(typ string) bool {
	return strings.HasPrefix(typ, string(ranke.NodeClassSource)+"/")
}

// How far back "recent" reaches, how often a reference skips that window for anywhere
// in the archive, and how many one claim makes.
const (
	recentWindow  = 40
	farReachOdds  = 8
	maxReferences = 3
)

// pick chooses up to n earlier claims to cite, biased to the recent, occasionally
// reaching right back. Duplicates drop: the same edge twice is one edge.
func (g *grower) pick(n int) []made {
	if len(g.made) == 0 || n < 1 {
		return nil
	}
	seen := make(map[string]bool, n)
	refs := make([]made, 0, n)
	for range g.rnd.IntN(n) + 1 {
		i := len(g.made) - 1 - g.rnd.IntN(min(len(g.made), recentWindow))
		if g.rnd.IntN(farReachOdds) == 0 {
			i = g.rnd.IntN(len(g.made))
		}
		if key := g.made[i].claim.ID().String(); !seen[key] {
			seen[key] = true
			refs = append(refs, g.made[i])
		}
	}
	return refs
}

// claim signs one inline-text claim citing refs, and remembers it. Height is explicit
// because a referencing claim must declare it (§4.1), and every claim cites its
// contributor — an initial node at 0, so citing only it sits at 1.
func (g *grower) claim(typ, text string, refs ...made) (made, error) {
	edges := make([]ranke.Edge, 0, len(refs))
	var height uint64
	for _, ref := range refs {
		edge, err := ranke.NewEdge(ranke.EdgeConfig{Reference: ref.claim.ID(), Type: "derivation/input"})
		if err != nil {
			return made{}, fmt.Errorf("edge to %s: %w", ref.claim.ID(), err)
		}
		edges = append(edges, edge)
		height = max(height, ref.height)
	}
	height++

	claim, err := ranke.NewClaim(typ, g.self).
		WithInlineContent([]byte(text)).
		WithEncoding("text/plain").
		WithCreatedAt(g.at).
		WithHeight(height).
		WithEdges(edges...).
		Sign()
	if err != nil {
		return made{}, fmt.Errorf("sign %s: %w", typ, err)
	}
	g.at = g.at.Add(clockStep)
	m := made{claim: claim, height: height}
	g.made = append(g.made, m)
	return m, nil
}
