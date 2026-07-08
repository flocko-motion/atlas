# Cypher conformance: goraphdb vs the openCypher TCK

A test harness that measures how much of the **Cypher / GQL** language
[`github.com/mstrYoda/goraphdb`](https://github.com/mstrYoda/goraphdb) actually
implements, by running the official **openCypher Technology Compatibility Kit
(TCK)** against it.

## Background — is there a standard?

- **openCypher** is Neo4j's open specification of Cypher. It ships a real
  conformance suite, the **TCK**: ~220 `.feature` files (Gherkin) describing
  queries and their expected results. That suite is vendored here under
  [`tck/features/`](tck/features) (Apache-2.0, attribution in
  `tck/LICENSE-openCypher`).
- **GQL — ISO/IEC 39075:2024** is the formal ISO standard (published Apr 2024,
  same committee as SQL). It has **no** official conformance test suite —
  conformance is vendor-self-asserted. The openCypher TCK is the closest
  runnable proxy, and openCypher now tracks the GQL editions.

So: the TCK is the thing you can actually *run*. That's what this does.

## What's here

```
adapter/      Tiny app wrapping goraphdb behind a Run(query) interface.
              Each scenario gets a fresh on-disk DB in a temp dir.
tck/
  features/   Vendored openCypher TCK (~220 features, ~3900 scenarios).
  value.go    Canonical value model + goraphdb→canonical conversion.
  parse.go    Recursive-descent parser for TCK expected-value strings.
  compare.go  Result-table and side-effect comparison.
  steps.go    godog step definitions (the TCK Gherkin vocabulary).
  tck_test.go Test entry: runs godog, prints the conformance report, writes CSV.
Makefile      Run targets.
```

The "tiny Go app that runs a gdb" is [`adapter/adapter.go`](adapter/adapter.go):
it opens goraphdb, runs a Cypher string (with optional parameters), and returns
columns + rows. The harness drives that adapter for every TCK scenario.

## Running

```bash
make smoke        # sanity check goraphdb opens and runs basic queries
make conformance  # full TCK run, prints just the conformance report
make test         # full run, verbose (per-scenario godog output too)
make subset SUBSET=features/clauses/create   # narrow to one area
make report       # outcome breakdown from the last run's CSV
```

Requires Go 1.26+. First run downloads goraphdb, godog, and bbolt.

## How a scenario is judged

Each TCK scenario is classified into one outcome:

| outcome            | meaning |
|--------------------|---------|
| `pass`             | query ran and results (+ checkable side effects) matched |
| `error-raised-ok`  | TCK expected an error; goraphdb raised *some* error |
| `wrong-result`     | query ran but results or side effects disagreed |
| `unsupported`      | goraphdb rejected a query the TCK expects to succeed (parser/lexer error, or setup failed) |
| `error-not-raised` | TCK expected an error; goraphdb returned a result instead |
| `skipped-harness`  | needs a fixture this harness doesn't provide (named `binary-tree` graphs, user procedures) |

"Conformance" = `pass` + `error-raised-ok` over total.

### Honest caveats

- **`error-raised-ok` is lenient.** The TCK names a specific error type/detail
  (e.g. `SyntaxError: UndefinedVariable`); goraphdb's error strings won't match
  Neo4j's, so we only check that *an* error was raised. Many of these are
  goraphdb's parser bailing on a query that happens to be one the TCK also
  expects to fail — so the *positive* signal is really the `pass` count.
- **Side effects are approximate.** goraphdb's result type exposes no
  statistics, so `+nodes/-nodes/+relationships/+labels/+properties` are derived
  by snapshotting graph state before/after (`adapter.Snapshot`). Net property
  diffs can disagree with the TCK's assignment-counting on overwrites.
- **Named-graph and procedure scenarios are skipped**, not failed (~18).

## Results (goraphdb @ `v0.0.0-20260220134623`)

```
Total scenarios: 3897
  pass                  55  (  1.4%)
  error-raised-ok      646  ( 16.6%)
  wrong-result          29  (  0.7%)
  unsupported         3126  ( 80.2%)
  error-not-raised      23  (  0.6%)
  skipped-harness       18  (  0.5%)

  Conformance (pass + correct-error): 701/3897 = 18.0%
```

**~80% of TCK queries don't even parse.** goraphdb implements a small,
pragmatic slice of Cypher, not the standard language. The biggest gaps, by
scenario count:

1. **No standalone `RETURN`** — every query must begin with `MATCH`. This alone
   fails ~1505 scenarios (the whole `expressions/` tree is written as
   `RETURN <expr>`). This is the single largest gap by far.
2. **No comma-separated patterns** — `MATCH (a), (b)` and `CREATE (), ()` are
   rejected.
3. **No relationship properties in patterns** — `[:T {k: v}]` fails to parse
   (`expected ] but got {`).
4. **Limited lexer** — operators/characters like `|`, `+`, `%`, `/` in
   expressions are unrecognised tokens.
5. **Limited clause support** — `OPTIONAL MATCH` at statement start, `WITH`
   chaining, `UNWIND`, list/path predicates, etc.

What *does* work is the documented happy path: `MATCH` with label/property
filters, `WHERE` comparisons, single-hop and bounded variable-length
traversal, `CREATE`/`MERGE` of nodes, `ORDER BY`, and parameters.

### Coverage vs. correctness — two different axes

The headline 18% conflates two things. Separating them is more useful:

- **Coverage** — goraphdb's parser accepts only ~20% of TCK queries.
- **Correctness within what it accepts** — of the **84** positive scenarios
  whose query goraphdb actually executed, only **55 were correct (65%)**.

That second number is the alarming one: among queries goraphdb *accepts and
runs without error*, roughly a third return the wrong answer **silently** — no
error, just incorrect results. A few examples confirmed by direct probing:

| query | expected | goraphdb |
|-------|----------|----------|
| `MATCH (n:A) SET n.prop = 5 RETURN n.prop` | `5` | `null` — `SET` is a no-op |
| `MATCH (n) RETURN n LIMIT 0` | 0 rows | all rows — `LIMIT 0` ignored |
| `MATCH (n) RETURN n.v SKIP 2` | last row | all rows — `SKIP` ignored |
| `... WHERE 4611686018427387905 = 4611686018427387907` | 0 rows | 1 row — large ints collapse (precision) |
| `RETURN labels(n)` | `['Foo','Bar']` | error — `labels()` unimplemented |
| `collect()` over a null | `[]` | `null` |

Silent wrong answers on `SET`/`SKIP`/`LIMIT` are worse than an honest "not
supported" parse error — an application can't tell it got bad data.

Full per-scenario detail is written to `tck/tck-results.csv` after each run.

## Reusing this against another database

The harness is engine-agnostic above [`adapter`](adapter/adapter.go). To test a
different embedded graph DB, implement the same `New() / Run(ctx, query,
params) (*Result, error) / Snapshot() / Close()` surface and extend
`FromActual` in `tck/value.go` to map that engine's node/relationship types.
