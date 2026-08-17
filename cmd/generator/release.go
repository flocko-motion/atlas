// package: main / cmd
// type:    logic
// job:     the release-process scenario — a fixed script, not a grown shape
// limits:  builds claims only; delivering them is the client's (-> client.go)
//
// A hand-written archive of a release process: four signing identities, two packages
// travelling from a git snapshot to a signed-off release, and the CVEs a scan mentions.
// Where `chain` grows an archive by rule, this one says what happens, in order, because
// what it exists to show is a shape a reader recognises.
//
// It is also a claim about the model: everything the slide draws is built here for real,
// so a picture that cannot be built cannot be presented either.
package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/flocko-motion/ranke-go"
)

// The scenario's vocabulary. Only `contribution/*` subtypes are reserved (§4.3); the rest is
// open, so these name what each claim *is* rather than which drawer it belongs in — a reader
// filtering for `derivation/triage` wants triage decisions, not classifications in general.
const (
	typeGitSnapshot   = "source/git_snapshot"
	typeBuildLog      = "source/build_log"
	typeTestReport    = "source/test_report"
	typeAdvisory      = "source/advisory"
	typeVulnerability = "entity/vulnerability"
	typeScan          = "derivation/vulnerability_scan"
	typeCandidate     = "derivation/release_candidate"
	typeTriage        = "derivation/triage"
	typeRelease       = "derivation/release"
)

// The identities the process runs on. Each signs what it would sign in life, which is what
// makes several identities worth having: whose claim a thing is says how far to trust it.
const (
	whoRelease  = "release-manager"
	whoSecurity = "security-expert"
	whoTests    = "test-executor"
	whoCI       = "ci-runner"
)

// packages are the two that go out together. Two in one release, rather than one release
// twice, is the whole point of the picture: the artifact is where they meet.
var packages = []string{"ranke-go", "ranke-db"}

// vulnerabilities are the advisories a scan finds. Global and public: a CVE belongs to no
// project, so both packages' scans reach the same claim rather than a copy each — which is
// also the case an explorer's union store exists for.
var vulnerabilities = []struct{ id, summary string }{
	{"CVE-2024-31337", "unbounded allocation in a length-prefixed decoder"},
	{"CVE-2024-40001", "signature check skipped when the algorithm field is absent"},
}

// release builds the scenario over `releases` iterations, each carrying both packages from
// snapshot to signed-off release. One branch: a release process is one line, and spreading
// it over several would say something about branches rather than about releases.
func (g *grower) release(ctx context.Context, branch string, releases int) (batches, error) {
	if releases < 1 {
		return nil, fmt.Errorf("release: --releases must be at least 1")
	}
	out := make(batches, 0, releases*5+1)

	// The actors, and the CVEs they will refer to, before anything cites them. The root
	// contributor claim rides in ahead of these, added by the caller: everything here is
	// attributed to it, so a branch that has not seen it cannot resolve the closure.
	actors, err := g.actors(ctx)
	if err != nil {
		return nil, err
	}
	setup := claimsOf(actors[whoRelease].claim, actors[whoSecurity].claim,
		actors[whoTests].claim, actors[whoCI].claim)

	found, advisories, err := g.vulnerabilities(branch, actors[whoSecurity])
	if err != nil {
		return nil, err
	}
	setup = append(setup, advisories...)
	out = append(out, batch{branch: branch, claims: setup})

	for i := range releases {
		version := fmt.Sprintf("v0.%d.0", i+1)
		var triaged []made
		for _, pkg := range packages {
			bs, decision, err := g.releaseOnePackage(branch, pkg, version, actors, found)
			if err != nil {
				return nil, err
			}
			out = append(out, bs...)
			triaged = append(triaged, decision)
		}

		// The fan-in: one release citing both triage decisions. A reader following either
		// package upward arrives here, which is what a release *is*.
		artifact, err := g.write(spec{
			by:      actors[whoRelease],
			branch:  branch,
			typ:     typeRelease,
			fields:  map[string]string{"name": version},
			content: []byte(releaseNotes(version, packages)),
			cites:   asInputs(triaged...),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, batch{branch: branch, claims: claimsOf(artifact.claim)})
	}
	return out, nil
}

// actors derives the four signing identities the root vouches for.
func (g *grower) actors(ctx context.Context) (map[string]*signer, error) {
	out := make(map[string]*signer, 4)
	for _, name := range []string{whoRelease, whoSecurity, whoTests, whoCI} {
		s, err := g.attest(ctx, name)
		if err != nil {
			return nil, err
		}
		out[name] = s
	}
	return out, nil
}

// vulnerabilities writes each advisory as it arrived and the entity a scan will mention. Two
// claims per CVE because an entity rests on stated provenance (§3.5): the advisory is what we
// were told, the entity is the thing an archive then links to.
func (g *grower) vulnerabilities(branch string, by *signer) ([]made, []ranke.Claim, error) {
	found := make([]made, 0, len(vulnerabilities))
	claims := make([]ranke.Claim, 0, len(vulnerabilities)*2)
	for _, v := range vulnerabilities {
		advisory, err := g.write(spec{
			by:      by,
			branch:  branch,
			typ:     typeAdvisory,
			fields:  map[string]string{"name": v.id},
			content: []byte(logText(v.id+" — "+v.summary, 2)),
		})
		if err != nil {
			return nil, nil, err
		}
		entity, err := g.write(spec{
			by:      by,
			branch:  branch,
			typ:     typeVulnerability,
			fields:  map[string]string{"name": v.id},
			content: []byte(v.id + ": " + v.summary),
			cites:   asInputs(advisory),
		})
		if err != nil {
			return nil, nil, err
		}
		found = append(found, entity)
		claims = append(claims, advisory.claim, entity.claim)
	}
	return found, claims, nil
}

// releaseOnePackage takes one package from snapshot to triage decision, a contribution per
// step so the branch table records the process advancing rather than appearing done.
func (g *grower) releaseOnePackage(
	branch, pkg, version string,
	actors map[string]*signer,
	found []made,
) (batches, made, error) {
	name := map[string]string{"name": pkg, "version": version}

	// What arrives from outside: the code, and the logs of what was run against it. Sources
	// rest on nothing, which is what makes them sources.
	snapshot, err := g.write(spec{
		by: actors[whoCI], branch: branch, typ: typeGitSnapshot, fields: name,
		content: []byte(snapshotText(pkg, version)),
	})
	if err != nil {
		return nil, made{}, err
	}
	buildLog, err := g.write(spec{
		by: actors[whoCI], branch: branch, typ: typeBuildLog, fields: name,
		content: []byte(logText(fmt.Sprintf("build %s %s", pkg, version), 6)),
	})
	if err != nil {
		return nil, made{}, err
	}
	testReport, err := g.write(spec{
		by: actors[whoTests], branch: branch, typ: typeTestReport, fields: name,
		content: []byte(logText(fmt.Sprintf("test %s %s", pkg, version), 8)),
	})
	if err != nil {
		return nil, made{}, err
	}

	// The scan derives from the snapshot and *mentions* the CVEs: it did not produce them,
	// and a mention is a different relation from an input (§4.7).
	cites := asInputs(snapshot)
	for _, v := range found {
		cites = append(cites, cite{to: v, typ: edgeMentions, dir: ranke.RelationFrom})
	}
	scan, err := g.write(spec{
		by: actors[whoSecurity], branch: branch, typ: typeScan, fields: name,
		content: []byte(scanText(pkg, version, vulnerabilities)),
		cites:   cites,
	})
	if err != nil {
		return nil, made{}, err
	}

	// The candidate cites all four, which is what makes it a candidate rather than an opinion.
	candidate, err := g.write(spec{
		by: actors[whoCI], branch: branch, typ: typeCandidate, fields: name,
		content: []byte(fmt.Sprintf("release candidate %s %s", pkg, version)),
		cites:   asInputs(snapshot, buildLog, testReport, scan),
	})
	if err != nil {
		return nil, made{}, err
	}

	// The decision cites the candidate and names who made it. Naming an actor is not an
	// input edge: the release manager is not something the decision was derived from.
	decided := asInputs(candidate)
	for _, who := range []string{whoRelease, whoSecurity} {
		decided = append(decided, cite{
			to:  made{claim: actors[who].claim, height: actors[who].height, branch: branch},
			typ: edgeDecidedBy,
			dir: ranke.RelationFrom,
		})
	}
	decision, err := g.write(spec{
		by: actors[whoRelease], branch: branch, typ: typeTriage, fields: name,
		content: []byte(triageText(pkg, version, vulnerabilities)),
		cites:   decided,
	})
	if err != nil {
		return nil, made{}, err
	}

	return batches{
		{branch: branch, claims: claimsOf(snapshot.claim, buildLog.claim, testReport.claim)},
		{branch: branch, claims: claimsOf(scan.claim)},
		{branch: branch, claims: claimsOf(candidate.claim)},
		{branch: branch, claims: claimsOf(decision.claim)},
	}, decision, nil
}

// claimsOf collects claims for a batch, which is what a contribution carries.
func claimsOf(claims ...ranke.Claim) []ranke.Claim {
	return claims
}

// logParagraph is the filler a fake log is padded with. Content this size is the point: it
// makes `output.content.max` and its overflow decide something, and gives the content
// routes bytes worth fetching.
const logParagraph = "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do " +
	"eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, " +
	"quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. " +
	"Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu " +
	"fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa " +
	"qui officia deserunt mollit anim id est laborum."

// logText is a fake log: a title, then filler. Deterministic, so the same seed writes the
// same ids.
func logText(title string, paragraphs int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n%s\n\n", title, strings.Repeat("=", len(title)))
	for i := range paragraphs {
		fmt.Fprintf(&b, "[%02d:%02d:%02d] %s\n\n", 9, i*7%60, i*13%60, logParagraph)
	}
	return b.String()
}

// snapshotText is what a git snapshot claim carries: enough to recognise the commit.
func snapshotText(pkg, version string) string {
	return fmt.Sprintf("%s %s\ncommit %s\nchore(release): prepare %s\n",
		pkg, version, strings.Repeat("0", 8)+pkg[:3], version)
}

// scanText reports what the scan found, in the order the CVEs are mentioned.
func scanText(pkg, version string, found []struct{ id, summary string }) string {
	var b strings.Builder
	fmt.Fprintf(&b, "vulnerability scan — %s %s\n\n", pkg, version)
	for _, v := range found {
		fmt.Fprintf(&b, "  %s  %s\n", v.id, v.summary)
	}
	fmt.Fprintf(&b, "\n%d advisory match(es), transitive dependencies included\n", len(found))
	return b.String()
}

// triageText is the decision a reader wants to read: what was found, and what was decided
// about it.
func triageText(pkg, version string, found []struct{ id, summary string }) string {
	var b strings.Builder
	fmt.Fprintf(&b, "triage — %s %s\n\n", pkg, version)
	for i, v := range found {
		verdict := "accepted: not reachable from any exported path"
		if i == 0 {
			verdict = "mitigated: decoder bounded in this release"
		}
		fmt.Fprintf(&b, "  %s  %s\n", v.id, verdict)
	}
	fmt.Fprint(&b, "\ndecision: ship\n")
	return b.String()
}

// releaseNotes is what the release itself says, naming what went out together.
func releaseNotes(version string, of []string) string {
	return fmt.Sprintf("release %s\n\npackages: %s\ntriage: both packages signed off\n",
		version, strings.Join(of, ", "))
}
