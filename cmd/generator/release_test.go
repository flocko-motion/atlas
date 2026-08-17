package main

import (
	"context"
	"strings"
	"testing"

	"github.com/flocko-motion/ranke-go"
)

// TestReleaseSignsAsFourIdentities pins the addition the scenario exists for: claims that
// several identities signed, each attested by the root, and each claim attributed to the one
// that would have made it.
func TestReleaseSignsAsFourIdentities(t *testing.T) {
	g := newTestGrower(t)
	bs, err := g.release(context.Background(), "main", 1)
	if err != nil {
		t.Fatalf("release: %v", err)
	}

	// Every contributor claim beyond the root, and who each names as its own contributor.
	contributors := map[string]ranke.Id{}
	for _, b := range bs {
		for _, claim := range b.claims {
			if claim.Node().Type() == string(ranke.NodeTypeContributor) {
				contributors[claim.ID().String()] = contributorOf(t, claim)
			}
		}
	}
	if len(contributors) != 4 {
		t.Fatalf("attested %d identities, want the four the process runs on", len(contributors))
	}
	for id, under := range contributors {
		if !under.Equal(g.selfClaim.ID()) {
			t.Errorf("contributor %s sits under %s, want the root %s", id, under, g.selfClaim.ID())
		}
	}

	// And the work is spread across them: a single-identity archive would say nothing about
	// who did what, which is what the slide's actor column is for.
	signers := map[string]int{}
	for _, b := range bs {
		for _, claim := range b.claims {
			if claim.Node().Type() == string(ranke.NodeTypeContributor) {
				continue
			}
			signers[contributorOf(t, claim).String()]++
		}
	}
	if len(signers) != 4 {
		t.Fatalf("claims signed by %d identities, want all four to have signed something", len(signers))
	}
	for id, n := range signers {
		if _, ok := contributors[id]; !ok {
			t.Errorf("%d claim(s) attributed to %s, which no contributor claim attests", n, id)
		}
	}
}

// TestReleaseFansInAtTheArtifact is the picture's point: two packages in one release, meeting
// at a claim that cites both triage decisions.
func TestReleaseFansInAtTheArtifact(t *testing.T) {
	g := newTestGrower(t)
	bs, err := g.release(context.Background(), "main", 2)
	if err != nil {
		t.Fatalf("release: %v", err)
	}

	byType := claimsByType(bs)
	if got := len(byType[typeRelease]); got != 2 {
		t.Fatalf("releases = %d, want one per iteration", got)
	}
	if got := len(byType[typeTriage]); got != 4 {
		t.Fatalf("triage decisions = %d, want one per package per release", got)
	}

	for _, release := range byType[typeRelease] {
		cited := edgesOfType(release, edgeInput)
		if len(cited) != 2 {
			t.Fatalf("release cites %d decisions, want both packages", len(cited))
		}
		// The two it cites are triage decisions for different packages, which is what makes
		// this a fan-in rather than two releases that happen to share a name.
		named := map[string]bool{}
		for _, id := range cited {
			decision := findClaim(t, bs, id)
			if decision.Node().Type() != typeTriage {
				t.Errorf("release cites a %s, want a %s", decision.Node().Type(), typeTriage)
			}
			named[fieldOf(t, decision, "name")] = true
		}
		if len(named) != len(packages) {
			t.Errorf("release cites decisions for %d package(s), want %d", len(named), len(packages))
		}
	}
}

// TestReleaseSharesTheVulnerabilities pins that a CVE is one claim both scans reach. A
// vulnerability is public and belongs to no package, so a copy per package would be a
// different claim about the same thing — and the merge is what an explorer's union is for.
func TestReleaseSharesTheVulnerabilities(t *testing.T) {
	g := newTestGrower(t)
	bs, err := g.release(context.Background(), "main", 2)
	if err != nil {
		t.Fatalf("release: %v", err)
	}

	byType := claimsByType(bs)
	if got := len(byType[typeVulnerability]); got != len(vulnerabilities) {
		t.Fatalf("vulnerability entities = %d, want %d — one per CVE, shared",
			got, len(vulnerabilities))
	}

	scans := byType[typeScan]
	if len(scans) != len(packages)*2 {
		t.Fatalf("scans = %d, want one per package per release", len(scans))
	}
	shared := map[string]int{}
	for _, scan := range scans {
		mentioned := edgesOfType(scan, edgeMentions)
		if len(mentioned) != len(vulnerabilities) {
			t.Errorf("scan mentions %d CVEs, want %d", len(mentioned), len(vulnerabilities))
		}
		for _, id := range mentioned {
			shared[id.String()]++
		}
	}
	if len(shared) != len(vulnerabilities) {
		t.Fatalf("scans reached %d distinct CVE claims, want %d", len(shared), len(vulnerabilities))
	}
	for id, times := range shared {
		if times != len(scans) {
			t.Errorf("CVE %s reached by %d of %d scans, want every one", id, times, len(scans))
		}
	}
}

// TestReleaseNamesWhoDecided pins that an actor is named rather than derived from: a triage
// decision reaches its candidate as an input and the deciders as a relation.
func TestReleaseNamesWhoDecided(t *testing.T) {
	g := newTestGrower(t)
	bs, err := g.release(context.Background(), "main", 1)
	if err != nil {
		t.Fatalf("release: %v", err)
	}

	for _, decision := range claimsByType(bs)[typeTriage] {
		if got := len(edgesOfType(decision, edgeInput)); got != 1 {
			t.Errorf("decision cites %d inputs, want the candidate alone", got)
		}
		deciders := edgesOfType(decision, edgeDecidedBy)
		if len(deciders) != 2 {
			t.Fatalf("decision names %d deciders, want two", len(deciders))
		}
		for _, id := range deciders {
			who := findClaim(t, bs, id)
			if who.Node().Type() != string(ranke.NodeTypeContributor) {
				t.Errorf("decided_by reaches a %s, want a contributor claim", who.Node().Type())
			}
		}
		// A relation edge states its direction (§4.7), so a reader can tell which way it runs.
		for _, edge := range decision.Edges() {
			if edge.Type() == edgeDecidedBy && edge.RelationDirection() == 0 {
				t.Error("decided_by edge states no direction")
			}
		}
	}
}

// TestReleaseCarriesContentWorthReading pins the third addition: logs big enough that a
// content cap decides something, and a title a reader recognises.
func TestReleaseCarriesContentWorthReading(t *testing.T) {
	g := newTestGrower(t)
	bs, err := g.release(context.Background(), "main", 1)
	if err != nil {
		t.Fatalf("release: %v", err)
	}

	byType := claimsByType(bs)
	for _, typ := range []string{typeBuildLog, typeTestReport} {
		claims := byType[typ]
		if len(claims) == 0 {
			t.Fatalf("no %s claims", typ)
		}
		for _, claim := range claims {
			size := claim.Node().GetContentSize()
			if size < 1024 {
				t.Errorf("%s carries %d bytes, want a log a content cap would bite on", typ, size)
			}
			body := content(t, claim)
			if !strings.Contains(body, fieldOf(t, claim, "name")) {
				t.Errorf("%s content names no package, so a reader cannot tell whose log it is", typ)
			}
		}
	}
}

// TestReleaseIsDeterministic pins reproducibility: re-seeding writes the claims already
// there rather than a second archive beside them.
func TestReleaseIsDeterministic(t *testing.T) {
	first, err := newTestGrower(t).release(context.Background(), "main", 2)
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	again, err := newTestGrower(t).release(context.Background(), "main", 2)
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	if len(first) != len(again) {
		t.Fatalf("contributions = %d then %d", len(first), len(again))
	}
	for i := range first {
		for j := range first[i].claims {
			if !first[i].claims[j].ID().Equal(again[i].claims[j].ID()) {
				t.Fatalf("claim %d.%d differs between runs: %s vs %s",
					i, j, first[i].claims[j].ID(), again[i].claims[j].ID())
			}
		}
	}
}

// TestReleaseHeightsCountTheContributorEdge pins the arithmetic an attested identity changes:
// its own claim sits at 1, so everything it signs sits above that, not at 1 like the root's.
func TestReleaseHeightsCountTheContributorEdge(t *testing.T) {
	g := newTestGrower(t)
	bs, err := g.release(context.Background(), "main", 1)
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	for _, claim := range claimsByType(bs)[typeGitSnapshot] {
		// A source cites nothing but its contributor, which is itself at 1 — so 2.
		if got := claim.Node().Height(); got != 2 {
			t.Errorf("snapshot height = %d, want 2: its signer's claim is not an initial node", got)
		}
	}
}

// --- helpers --------------------------------------------------------------

// claimsByType indexes every claim in a scenario by node type.
func claimsByType(bs batches) map[string][]ranke.Claim {
	out := map[string][]ranke.Claim{}
	for _, b := range bs {
		for _, claim := range b.claims {
			typ := claim.Node().Type()
			out[typ] = append(out[typ], claim)
		}
	}
	return out
}

// edgesOfType returns what a claim reaches through edges of one type.
func edgesOfType(claim ranke.Claim, typ string) []ranke.Id {
	var ids []ranke.Id
	for _, edge := range claim.Edges() {
		if edge.Type() == typ {
			ids = append(ids, edge.Reference())
		}
	}
	return ids
}

// contributorOf returns the id of the contributor claim a claim is attributed to.
func contributorOf(t *testing.T, claim ranke.Claim) ranke.Id {
	t.Helper()
	for _, edge := range claim.Edges() {
		if edge.Type() == string(ranke.EdgeTypeContributor) {
			return edge.Reference()
		}
	}
	t.Fatalf("%s carries no contributor edge", claim.Node().Type())
	return nil
}

// findClaim looks a claim up by id across every contribution.
func findClaim(t *testing.T, bs batches, id ranke.Id) ranke.Claim {
	t.Helper()
	for _, b := range bs {
		for _, claim := range b.claims {
			if claim.ID().Equal(id) {
				return claim
			}
		}
	}
	t.Fatalf("no claim in the scenario has id %s", id)
	return nil
}

// content reads a claim's inline bytes, which is where this scenario keeps its logs.
func content(t *testing.T, claim ranke.Claim) string {
	t.Helper()
	body, err := claim.Node().GetInlineContent()
	if err != nil {
		t.Fatalf("read content: %v", err)
	}
	return string(body)
}

// fieldOf reads a claim field, which is what a reader filters on.
func fieldOf(t *testing.T, claim ranke.Claim, name string) string {
	t.Helper()
	value, err := claim.Node().GetField(name)
	if err != nil {
		t.Fatalf("field %q: %v", name, err)
	}
	return value
}
