package tck

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
)

// TestTCK runs the vendored openCypher Technology Compatibility Kit against
// goraphdb and prints a conformance summary.
//
// The TCK feature files live in ./features. Set TCK_FEATURES to point at a
// subset (e.g. features/clauses/create) to narrow the run.
func TestTCK(t *testing.T) {
	path := os.Getenv("TCK_FEATURES")
	if path == "" {
		path = "features"
	}

	tally := NewTally()
	suite := godog.TestSuite{
		Name: "openCypher-TCK-vs-goraphdb",
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			Register(ctx, tally)
		},
		Options: &godog.Options{
			Format:        "progress",
			Paths:         []string{path},
			Output:        colors.Colored(os.Stdout),
			Concurrency:   1, // shared tally + per-scenario engine
			StopOnFailure: false,
			Strict:        false,
		},
	}

	// godog returns non-zero when scenarios fail; for a conformance survey we
	// expect failures, so we don't fail the Go test on that — we report.
	suite.Run()
	printReport(tally)

	if total := len(tally.Details); total == 0 {
		t.Fatalf("no scenarios ran for path %q", path)
	}
}

func printReport(t *Tally) {
	total := len(t.Details)
	order := []Outcome{OutPass, OutErrorOK, OutWrongResult, OutUnsupported, OutErrorMiss, OutSkipped}

	fmt.Printf("\n==================== openCypher TCK — goraphdb conformance ====================\n")
	fmt.Printf("Total scenarios: %d\n\n", total)
	for _, o := range order {
		n := t.Counts[o]
		pct := 0.0
		if total > 0 {
			pct = 100 * float64(n) / float64(total)
		}
		fmt.Printf("  %-18s %5d  (%5.1f%%)\n", o, n, pct)
	}
	passLike := t.Counts[OutPass] + t.Counts[OutErrorOK]
	fmt.Printf("\n  Conformance (pass + correct-error): %d/%d = %.1f%%\n",
		passLike, total, 100*float64(passLike)/float64(total))

	// Top failure reasons among non-passing, non-skipped scenarios.
	reasons := map[string]int{}
	for _, d := range t.Details {
		if d.Outcome == OutUnsupported || d.Outcome == OutWrongResult {
			reasons[bucket(d.Detail)]++
		}
	}
	if len(reasons) > 0 {
		type kv struct {
			k string
			n int
		}
		var rs []kv
		for k, n := range reasons {
			rs = append(rs, kv{k, n})
		}
		sort.Slice(rs, func(i, j int) bool { return rs[i].n > rs[j].n })
		fmt.Printf("\n  Most common failure reasons:\n")
		for i, r := range rs {
			if i >= 15 {
				break
			}
			fmt.Printf("    %4d  %s\n", r.n, r.k)
		}
	}
	fmt.Printf("===============================================================================\n")

	// Machine-readable dump for further analysis.
	writeCSV(t)
}

// bucket collapses a detailed failure message into a coarse reason so the
// summary groups similar failures.
func bucket(detail string) string {
	switch {
	case detail == "":
		return "(none)"
	case contains(detail, "cypher parser:"):
		return "parser: " + after(detail, "cypher parser:")
	case contains(detail, "columns mismatch"):
		return "columns mismatch"
	case contains(detail, "row count"):
		return "wrong row count"
	case contains(detail, "row ") && contains(detail, "mismatch"):
		return "row value mismatch"
	case contains(detail, "missing expected row"):
		return "missing/extra rows"
	case contains(detail, "side effect"):
		return "side-effect mismatch"
	case contains(detail, "unsupported") || contains(detail, "not supported"):
		return "engine: unsupported feature"
	default:
		if len(detail) > 60 {
			return detail[:60]
		}
		return detail
	}
}

func writeCSV(t *Tally) {
	f, err := os.Create("tck-results.csv")
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintln(f, "outcome,feature,scenario,detail")
	for _, d := range t.Details {
		fmt.Fprintf(f, "%s,%q,%q,%q\n", d.Outcome, d.Feature, d.Scenario, d.Detail)
	}
	fmt.Printf("  Per-scenario results written to tck/tck-results.csv\n")
}

func contains(s, sub string) bool {
	return len(sub) <= len(s) && indexOf(s, sub) >= 0
}

func after(s, sub string) string {
	i := indexOf(s, sub)
	if i < 0 {
		return s
	}
	r := s[i+len(sub):]
	if len(r) > 50 {
		r = r[:50]
	}
	return r
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
