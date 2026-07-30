package core

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	ranke "github.com/flocko-motion/ranke-go"
)

// newContributor mints a client-side contributor: its own key, its own signed
// contributor claim. This is the application's identity, not the server's signing
// identity — the client signs the claims, the server attests only the merge.
func newContributor(t *testing.T, u ranke.Universe) (ranke.Contributor, ed25519.PrivateKey, ranke.Claim) {
	t.Helper()
	ctx := context.Background()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	encoded, err := ranke.EncodePublicKey(pub)
	if err != nil {
		t.Fatalf("encode public key: %v", err)
	}
	claim, err := ranke.NewClaim(ranke.NodeContributor, nil).
		WithInlineContent(encoded).
		WithEncoding(ranke.EncodingOctetStream).
		WithCreatedAt(time.Unix(0, 0).UTC()).
		Sign(priv)
	if err != nil {
		t.Fatalf("sign contributor claim: %v", err)
	}
	self, err := claim.AsContributor(ctx, u, priv)
	if err != nil {
		t.Fatalf("AsContributor: %v", err)
	}
	// The contributor claim travels in the contribution too: every claim it signs
	// references it, so an archive that has never seen it cannot resolve the closure.
	return self, priv, claim
}

// contributorHeight is where a contributor claim sits: it references nothing, so the
// verifier treats it as an initial node at 0, and a claim citing it sits one above.
// (The normative spec §R-HEIGHT puts an initial node at 1; the library is the oracle for
// what verifies, so these follow it. Reported as a divergence.)
const contributorHeight = 0

// TestContributeRejectsAPartlyReadableBody pins that a body only partly understood is
// refused whole: a contribution is atomic, so guessing at one record would merge
// something the client did not send.
func TestContributeRejectsAPartlyReadableBody(t *testing.T) {
	c := newStack(t)
	for _, tc := range []struct{ name, body string }{
		{"empty", ""},
		{"not cbor at all", "this is not a contribution"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := &Request{Op: OpClaimContribute, Body: strings.NewReader(tc.body)}
			_, err := c.Handle(context.Background(), req)
			if !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("err = %v, want ErrInvalidRequest", err)
			}
			if got := Categorize(err); got != CatInvalid {
				t.Fatalf("category = %q, want %q", got, CatInvalid)
			}
		})
	}
}

// TestContributeMerges drives the whole arm: a client-signed claim crosses the wire
// format, the sequencer merges it, and the new head plus the contributed ids come back.
func TestContributeMerges(t *testing.T) {
	c := newStack(t)
	self, priv, selfClaim := newContributor(t, c.store)

	claim, err := ranke.NewClaim("source/letter", self).
		WithInlineContent([]byte("a claim the client signed")).
		WithEncoding(ranke.EncodingOctetStream).
		WithHeight(contributorHeight + 1).
		Sign(priv)
	if err != nil {
		t.Fatalf("sign claim: %v", err)
	}

	body := writeContribution(t, "main", selfClaim, claim)
	req := &Request{Op: OpClaimContribute, Body: bytes.NewReader(body)}
	out, mediaType := serve(t, c, req)
	if mediaType != mediaJSON {
		t.Fatalf("content type = %q, want %q", mediaType, mediaJSON)
	}

	var got Contribution
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v (body %s)", err, out)
	}
	if got.Head == "" {
		t.Fatal("no new head returned")
	}
	if len(got.Ids) == 0 {
		t.Fatal("no contributed ids returned")
	}

	// And the claim is now readable on the branch it was merged onto.
	read, _ := serve(t, c, &Request{Op: OpClaimGet, Branch: "main", ClaimID: claim.ID()})
	want, err := claim.EncodeCBOR(ranke.FormOriginal)
	if err != nil {
		t.Fatalf("EncodeCBOR: %v", err)
	}
	if !bytes.Equal(read, want) {
		t.Fatal("the merged claim does not read back as the bytes the client signed")
	}
}

// TestContributeSpansSeveralBranches pins what the body naming its own branches buys: one
// contribution advances more than one branch, and each needs the C right.
func TestContributeSpansSeveralBranches(t *testing.T) {
	c := newStack(t)
	self, priv, selfClaim := newContributor(t, c.store)

	var body bytes.Buffer
	w := ranke.NewWireWriter(&body)
	if err := w.WriteClaim("main", selfClaim); err != nil {
		t.Fatalf("WriteClaim: %v", err)
	}
	for _, branch := range []string{"main", "notes"} {
		claim, err := ranke.NewClaim("source/letter", self).
			WithInlineContent([]byte("a claim for " + branch)).
			WithEncoding(ranke.EncodingOctetStream).
			WithHeight(contributorHeight + 1).
			Sign(priv)
		if err != nil {
			t.Fatalf("sign claim: %v", err)
		}
		if err := w.WriteClaim(branch, claim); err != nil {
			t.Fatalf("WriteClaim: %v", err)
		}
	}

	out, _ := serve(t, c, &Request{Op: OpClaimContribute, Body: bytes.NewReader(body.Bytes())})
	var got Contribution
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v (body %s)", err, out)
	}
	if got.Head == "" {
		t.Fatal("no new head returned")
	}

	// Both branches now resolve.
	for _, branch := range []string{"main", "notes"} {
		if _, err := c.Handle(context.Background(), &Request{Op: OpBranchHead, Branch: branch}); err != nil {
			t.Fatalf("branch %q did not advance: %v", branch, err)
		}
	}
}

// writeContribution writes claims onto one branch, as a client would.
func writeContribution(t *testing.T, branch string, claims ...ranke.Claim) []byte {
	t.Helper()
	var body bytes.Buffer
	w := ranke.NewWireWriter(&body)
	for _, claim := range claims {
		if err := w.WriteClaim(branch, claim); err != nil {
			t.Fatalf("WriteClaim: %v", err)
		}
	}
	return body.Bytes()
}
