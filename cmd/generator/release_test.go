package main

import (
	"context"
	"strings"
	"testing"

	"github.com/rankegraph/ranke-go"
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
	// Five: the four actors, plus the root, which signs the records and entities that
	// introduce them — nobody else exists yet when those are written.
	if len(signers) != 5 {
		t.Fatalf("claims signed by %d identities, want the four actors and the root", len(signers))
	}
	if signers[g.selfClaim.ID().String()] == 0 {
		t.Error("the root signed nothing, so nothing introduced the actors")
	}
	for id, n := range signers {
		if _, ok := contributors[id]; !ok && id != g.selfClaim.ID().String() {
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
	// How many scans reach each CVE. One is reached by every scan and one by a single
	// package's, which is the arrangement the drawing is deliberate about: a claim two
	// provenance paths arrive at, beside one only reached by a single path.
	reached := map[string]int{}
	for _, scan := range scans {
		for _, id := range edgesOfType(scan, edgeMentions) {
			reached[findName(t, bs, id)]++
		}
	}
	if len(reached) != len(vulnerabilities) {
		t.Fatalf("scans reached %d distinct CVEs, want %d", len(reached), len(vulnerabilities))
	}
	for _, v := range vulnerabilities {
		want := len(scans)
		if len(v.affects) > 0 {
			want = len(scans) / len(packages) * len(v.affects)
		}
		if reached[v.id] != want {
			t.Errorf("%s reached by %d of %d scans, want %d", v.id, reached[v.id], len(scans), want)
		}
	}
	if reached[vulnerabilities[0].id] <= reached[vulnerabilities[1].id] {
		t.Error("no CVE is shared more widely than another, so the graph shows no merge")
	}
}

// TestReleaseNamesWhoDecided pins that an actor is named rather than derived from, and named
// as a PERSON rather than as a key: a contributor is operational and an entity semantic, and
// the two never share a node (foundation paper §Taxonomy). A decision reaches its candidate as
// an input and the deciders as a relation.
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
		named := map[string]bool{}
		for _, id := range deciders {
			who := findClaim(t, bs, id)
			if who.Node().Type() != typePerson {
				t.Errorf("decided_by reaches a %s, want a %s", who.Node().Type(), typePerson)
			}
			named[fieldOf(t, who, "name")] = true
		}
		// The release manager and the security expert decided; the test executor and the CI
		// runner did not, which the drawing is explicit about.
		for _, who := range []string{whoRelease, whoSecurity} {
			if !named[who] {
				t.Errorf("decision does not name %s", who)
			}
		}
		for _, who := range []string{whoTests, whoCI} {
			if named[who] {
				t.Errorf("decision names %s, who does not decide", who)
			}
		}
		// A relation edge states its direction (V-REL), so a reader can tell which way it runs.
		for _, edge := range decision.Edges() {
			if edge.Type() == edgeDecidedBy && edge.RelationDirection() != ranke.RelationTo {
				t.Error("decided_by does not mark the person as the object of the relation")
			}
		}
	}
}

// TestReleaseLinksKeysToPeople pins the one claim that joins the operational side to the
// semantic one: each key claim cites the actor it belongs to, and each actor rests on the
// record that introduced it.
func TestReleaseLinksKeysToPeople(t *testing.T) {
	g := newTestGrower(t)
	bs, err := g.release(context.Background(), "main", 1)
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	byType := claimsByType(bs)

	keys := byType[string(ranke.NodeTypeContributor)]
	if len(keys) != len(participants) {
		t.Fatalf("attested %d keys, want %d", len(keys), len(participants))
	}
	for _, key := range keys {
		of := edgesOfType(key, edgeIdentity)
		if len(of) != 1 {
			t.Fatalf("key claim cites %d actors, want exactly one", len(of))
		}
		actor := findClaim(t, bs, of[0])
		if typ := actor.Node().Type(); typ != typePerson && typ != typeCIInstance {
			t.Errorf("key claim cites a %s, want a person or a machine", typ)
		}
		// The actor has a path back to a source, which a contributor edge alone would not
		// give it (D1).
		if got := len(edgesOfType(actor, edgeInput)); got != 1 {
			t.Errorf("%s cites %d records, want the one that introduced it",
				actor.Node().Type(), got)
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
		// A source cites nothing but its contributor — but that key claim is four deep now:
		// the record introducing the actor sits at 1, the actor entity at 2, the key at 3, so
		// anything the key signs starts at 4. Getting this wrong is how a fixture ends up
		// with heights that contradict its own edges.
		if got := claim.Node().Height(); got != 4 {
			t.Errorf("snapshot height = %d, want 4: record 1, actor 2, key 3, this 4", got)
		}
	}
}

// TestReleaseContributionsAreOneEventEach pins the addition that fixes the archive's own
// history: a build finishing and a test suite finishing are different moments, so each
// lands as its own contribution — the branch table advances once per real event, not once
// for everything that happened since the last check-in.
func TestReleaseContributionsAreOneEventEach(t *testing.T) {
	g := newTestGrower(t)
	bs, err := g.release(context.Background(), "main", 1)
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	for i, b := range bs {
		if i == 0 {
			continue // setup: provisioning the fixture, not a lifecycle event
		}
		if len(b.claims) != 1 {
			t.Errorf("contribution %d carries %d claims, want one event each", i, len(b.claims))
		}
	}
}

// TestReleaseTimestampsAreMonotonic pins V-MONO across the whole scenario: no claim's
// created_at may sit before that of a claim it cites. A schedule built by counting
// backward from a later moment (triage, minus days of dev work) can otherwise walk a
// claim past what it's attributed to — as it once did past the setup phase itself.
func TestReleaseTimestampsAreMonotonic(t *testing.T) {
	g := newTestGrower(t)
	bs, err := g.release(context.Background(), "main", 2)
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	all := map[string]ranke.Claim{}
	for _, b := range bs {
		for _, c := range b.claims {
			all[c.ID().String()] = c
		}
	}
	for _, c := range all {
		for _, edge := range c.Edges() {
			ref, ok := all[edge.Reference().String()]
			if !ok {
				continue
			}
			if c.Node().CreatedAt().Before(ref.Node().CreatedAt()) {
				t.Errorf("%s %s (%s) predates its %s reference %s (%s)",
					c.Node().Type(), c.ID(), c.Node().CreatedAt(),
					edge.Type(), ref.Node().Type(), ref.Node().CreatedAt())
			}
		}
	}
}

// TestReleaseTimelineSpansDays pins the addition this scenario exists for: the archive
// goes quiet for days between code being ready and the release window, and the
// vulnerability scan lands close to triage rather than at build time.
func TestReleaseTimelineSpansDays(t *testing.T) {
	g := newTestGrower(t)
	bs, err := g.release(context.Background(), "main", 1)
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	byType := claimsByType(bs)
	for _, pkg := range packages {
		snapshot := findByName(t, byType[typeGitSnapshot], pkg)
		scan := findByName(t, byType[typeScan], pkg)
		decision := findByName(t, byType[typeTriage], pkg)

		if gap := scan.Node().CreatedAt().Sub(snapshot.Node().CreatedAt()); gap < devToTriageMin {
			t.Errorf("%s: snapshot to scan = %s, want at least %s", pkg, gap, devToTriageMin)
		}
		if gap := decision.Node().CreatedAt().Sub(scan.Node().CreatedAt()); gap < 0 || gap > scanLeadMax+candidateDelayMax {
			t.Errorf("%s: scan to triage = %s, want the scan landing close to the decision", pkg, gap)
		}
	}
}

// --- helpers --------------------------------------------------------------

// findByName returns the one claim in cs whose "name" field is name.
func findByName(t *testing.T, cs []ranke.Claim, name string) ranke.Claim {
	t.Helper()
	for _, c := range cs {
		if fieldOf(t, c, "name") == name {
			return c
		}
	}
	t.Fatalf("no claim named %q among %d", name, len(cs))
	return nil
}

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

// findName is the `name` field of the claim an id names.
func findName(t *testing.T, bs batches, id ranke.Id) string {
	t.Helper()
	return fieldOf(t, findClaim(t, bs, id), "name")
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
