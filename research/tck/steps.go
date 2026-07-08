package tck

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/cucumber/godog"
	"github.com/flocko-motion/rankedb/research/cypher-tck/adapter"
)

// Outcome categorises how a scenario ended, for the conformance summary.
type Outcome string

const (
	OutPass        Outcome = "pass"             // everything checked matched
	OutWrongResult Outcome = "wrong-result"     // query ran, result/side-effects wrong
	OutUnsupported Outcome = "unsupported"      // engine rejected a query the TCK expects to run
	OutErrorOK     Outcome = "error-raised-ok"  // TCK expected an error; engine raised one
	OutErrorMiss   Outcome = "error-not-raised" // TCK expected an error; engine produced a result
	OutSkipped     Outcome = "skipped-harness"  // harness can't set up (named graph, procedure)
)

// Tally accumulates per-scenario outcomes across the whole run.
type Tally struct {
	Counts  map[Outcome]int
	Details []ScenarioResult
}

// ScenarioResult is one scenario's verdict.
type ScenarioResult struct {
	Feature  string
	Scenario string
	Outcome  Outcome
	Detail   string
}

func NewTally() *Tally { return &Tally{Counts: map[Outcome]int{}} }

func (t *Tally) record(feat, name string, o Outcome, detail string) {
	t.Counts[o]++
	t.Details = append(t.Details, ScenarioResult{feat, name, o, detail})
}

// state holds per-scenario execution context.
type state struct {
	tally    *Tally
	feature  string
	name     string
	eng      *adapter.Engine
	params   map[string]any
	before   adapter.Snapshot
	result   *adapter.Result
	execErr  error
	executed bool
	outcome  Outcome
	detail   string
	skip     bool
}

func (s *state) ensureEngine() error {
	if s.eng != nil {
		return nil
	}
	e, err := adapter.New()
	if err != nil {
		return err
	}
	s.eng = e
	s.params = map[string]any{}
	return nil
}

func (s *state) set(o Outcome, detail string) error {
	s.outcome = o
	s.detail = detail
	if o == OutPass || o == OutErrorOK || o == OutSkipped {
		return nil
	}
	return fmt.Errorf("%s: %s", o, detail)
}

// --- Given steps -----------------------------------------------------------

func (s *state) givenEmptyGraph() error { return s.ensureEngine() }

func (s *state) givenNamedGraph(name string) error {
	// binary-tree-1 / binary-tree-2 fixtures are not loaded by this harness.
	s.skip = true
	return s.set(OutSkipped, "requires named graph fixture: "+name)
}

// --- setup / params --------------------------------------------------------

func (s *state) havingExecuted(doc *godog.DocString) error {
	if err := s.ensureEngine(); err != nil {
		return err
	}
	if _, err := s.eng.Run(context.Background(), doc.Content, s.params); err != nil {
		// Setup failing means the scenario cannot run on this engine.
		s.skip = true
		return s.set(OutUnsupported, "setup query failed: "+oneLine(err.Error()))
	}
	return nil
}

func (s *state) parametersAre(t *godog.Table) error {
	if err := s.ensureEngine(); err != nil {
		return err
	}
	for _, row := range t.Rows {
		if len(row.Cells) < 2 {
			continue
		}
		name := row.Cells[0].Value
		v, err := ParseValue(row.Cells[1].Value)
		if err != nil {
			return s.set(OutSkipped, "unparseable parameter "+name+": "+err.Error())
		}
		s.params[name] = v.ToGo()
	}
	return nil
}

// --- When ------------------------------------------------------------------

func (s *state) executingQuery(doc *godog.DocString) error {
	if err := s.ensureEngine(); err != nil {
		return err
	}
	s.before = s.eng.Snapshot()
	s.result, s.execErr = s.eng.Run(context.Background(), doc.Content, s.params)
	s.executed = true
	return nil
}

// --- Then: results ---------------------------------------------------------

func (s *state) resultShouldBe(t *godog.Table, inOrder, ignoreListOrder bool) error {
	if s.execErr != nil {
		return s.set(OutUnsupported, "query errored: "+oneLine(s.execErr.Error()))
	}
	exp, err := tableToExpected(t)
	if err != nil {
		return s.set(OutSkipped, "bad expected table: "+err.Error())
	}
	if err := CompareResult(s.result, exp, inOrder, ignoreListOrder); err != nil {
		return s.set(OutWrongResult, oneLine(err.Error()))
	}
	return s.set(OutPass, "")
}

func (s *state) resultShouldBeEmpty() error {
	if s.execErr != nil {
		return s.set(OutUnsupported, "query errored: "+oneLine(s.execErr.Error()))
	}
	if s.result != nil && len(s.result.Rows) != 0 {
		return s.set(OutWrongResult, fmt.Sprintf("expected empty result, got %d rows", len(s.result.Rows)))
	}
	return s.set(OutPass, "")
}

// --- Then: errors ----------------------------------------------------------

func (s *state) errorShouldBeRaised(errType, phase, detail string) error {
	if s.execErr != nil {
		return s.set(OutErrorOK, fmt.Sprintf("engine raised: %s", oneLine(s.execErr.Error())))
	}
	return s.set(OutErrorMiss, fmt.Sprintf("expected %s (%s/%s), but query succeeded", errType, phase, detail))
}

// --- And: side effects -----------------------------------------------------

func (s *state) noSideEffects() error {
	return s.checkSideEffects(map[string]int{})
}

func (s *state) sideEffectsShouldBe(t *godog.Table) error {
	want := map[string]int{}
	for _, row := range t.Rows {
		if len(row.Cells) < 2 {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(row.Cells[1].Value))
		if err != nil {
			return s.set(OutSkipped, "bad side-effect count: "+row.Cells[1].Value)
		}
		want[strings.TrimSpace(row.Cells[0].Value)] = n
	}
	return s.checkSideEffects(want)
}

func (s *state) checkSideEffects(want map[string]int) error {
	// Only meaningful after a successful mutating query.
	if !s.executed || s.execErr != nil {
		return nil // result/error step already classified this scenario
	}
	if s.outcome == OutWrongResult {
		return nil // already failed on results; don't override
	}
	after := s.eng.Snapshot()
	if err := CompareSideEffects(s.before, after, want); err != nil {
		return s.set(OutWrongResult, oneLine(err.Error()))
	}
	return nil
}

// --- registration ----------------------------------------------------------

// Register wires all TCK step definitions into a godog scenario, recording the
// outcome of each scenario into tally.
func Register(ctx *godog.ScenarioContext, tally *Tally) {
	s := &state{tally: tally}

	ctx.Before(func(c context.Context, sc *godog.Scenario) (context.Context, error) {
		s.reset(tally, sc)
		return c, nil
	})
	ctx.After(func(c context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		out := s.outcome
		detail := s.detail
		if err != nil && out == OutPass {
			// A step failed for a reason we didn't classify.
			out = OutWrongResult
			detail = oneLine(err.Error())
		}
		if out == "" {
			out = OutSkipped
			detail = "no terminal assertion"
		}
		tally.record(s.feature, s.name, out, detail)
		if s.eng != nil {
			s.eng.Close()
			s.eng = nil
		}
		return c, nil
	})

	ctx.Given(`^any graph$`, s.givenEmptyGraph)
	ctx.Given(`^an empty graph$`, s.givenEmptyGraph)
	ctx.Given(`^the (binary-tree-\d) graph$`, s.givenNamedGraph)
	ctx.Step(`^there exists a procedure (.+)$`, func(_ string) error {
		s.skip = true
		return s.set(OutSkipped, "requires user-defined procedure")
	})

	ctx.Step(`^having executed:$`, s.havingExecuted)
	ctx.Step(`^parameters are:$`, s.parametersAre)

	ctx.When(`^executing query:$`, s.executingQuery)
	ctx.When(`^executing control query:$`, s.executingQuery)

	ctx.Then(`^the result should be, in any order:$`, func(t *godog.Table) error {
		return s.resultShouldBe(t, false, false)
	})
	ctx.Then(`^the result should be, in order:$`, func(t *godog.Table) error {
		return s.resultShouldBe(t, true, false)
	})
	ctx.Then(`^the result should be \(ignoring element order for lists\):$`, func(t *godog.Table) error {
		return s.resultShouldBe(t, false, true)
	})
	ctx.Then(`^the result should be, in any order \(ignoring element order for lists\):$`, func(t *godog.Table) error {
		return s.resultShouldBe(t, false, true)
	})
	ctx.Then(`^the result should be empty$`, s.resultShouldBeEmpty)

	ctx.Then(`^an? (\w+) should be raised at (compile time|runtime): (\w+)$`, s.errorShouldBeRaised)

	ctx.Step(`^no side effects$`, s.noSideEffects)
	ctx.Step(`^the side effects should be:$`, s.sideEffectsShouldBe)
}

func (s *state) reset(tally *Tally, sc *godog.Scenario) {
	if s.eng != nil {
		s.eng.Close()
	}
	*s = state{tally: tally, name: sc.Name, feature: sc.Uri}
}

func tableToExpected(t *godog.Table) (ExpectedTable, error) {
	if len(t.Rows) == 0 {
		return ExpectedTable{}, fmt.Errorf("empty table")
	}
	exp := ExpectedTable{}
	for _, c := range t.Rows[0].Cells {
		exp.Columns = append(exp.Columns, c.Value)
	}
	for _, row := range t.Rows[1:] {
		cells := make([]string, len(row.Cells))
		for i, c := range row.Cells {
			cells[i] = c.Value
		}
		if len(cells) != len(exp.Columns) {
			return ExpectedTable{}, fmt.Errorf("row width %d != header width %d", len(cells), len(exp.Columns))
		}
		exp.Rows = append(exp.Rows, cells)
	}
	return exp, nil
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 220 {
		s = s[:220] + "…"
	}
	return s
}
