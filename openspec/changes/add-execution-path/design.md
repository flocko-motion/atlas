## Context

`Core.Handle` is `authenticate → authorize → execute`. The first two are written;
`execute` records one trace line and returns `ErrNotImplemented`.

Read against `ranke-go@v0.3.2` (fetched from the module proxy — no module cache here)
and Paper 02 §Sequencer / §Access Control / §Filtered Reads plus the normative spec
§RQL (fetched from the paper repo's `main`, since `/workspace` is read-only and
`make docs` cannot write).

The controlling fact: **ranke-go is finished for everything this stage does.** The
archive answers `Query`, `GetClaim`, `GetClaimContent`, `GetBranch`, `GetBranches` and
`Verify`; the contribution type implements the six steps; `concurrent` runs steps 2–5
off the sequencing thread with step 6 a serialised group commit. `execute` is a seam,
not an engine — anything here that reads like a description of graph behaviour is
misplaced and belongs upstream.

## Goals / Non-Goals

**Goals:**

- Dispatch: `Operation` → the library call that answers it.
- Serving: the library's result bytes onto the wire, framed and content-typed.
- Error mapping onto the existing sentinels.
- The verification-run registry, which is operational state rather than graph state.

**Non-Goals:**

- **Anything ranke-go already does.** The query engine, filters, ordering, closure
  walk, admission rules and merge steps. `concurrent.admissible()` already enforces
  the paper's step 2 (`errFutureDated`, `errReservedType`); the storage stack already
  reports `Capabilities()`. Restating these here would specify the library from
  inside the server.
- **The contribute arm's blockers.** Two upstream asks, below — recorded, not designed
  around.
- **The MCP endpoint.** A stub driving the same core.

## Decisions

### There is nothing to render — the server serves what the library shapes

The library already shapes the result set to the query's output axes. Four of the five
are honoured today:

| Axis | Honoured at |
|---|---|
| `Shape` | `query_default.go:45,87`, `archive.go:148`, `neo4j/query.go:22` |
| `Detail` | `query_default.go:88`, `neo4j/query.go:43,49` |
| `Form` | `stack/query.go:20` |
| `Content` | `query_default.go:98`, `stack/query.go:80` |

The fifth is not. **`Output.Encoding` is declared at `query.go:121-126` and consumed
nowhere** — `ResultJSON` and `ResultCBOR` are defined and never referenced — so
`QueryResult` comes back as Go values (`Id`, `Claim`, `[]Claim`, `[]byte`) even when
the client's query asked for cbor.

That is an upstream gap, not a server concern. `Output.Encoding` sits in the query AST
the client sends, so the engine is what should answer in it — exactly as it already
answers `Shape`, `Detail`, `Form` and `Content`. Building a serialisation layer here
to compensate would put the graph's canonical form in the server's hands, which is the
one place the razor says it must not be: get it wrong and a claim no longer verifies
against its id.

So once the library honours the axis, the server's job is a loop — write each result's
bytes, add the separator the media type requires, set the content type. RFC 7464
prefixes each record with RS; RFC 8742 concatenates. A single object and a raw blob are
degenerate cases of the same loop. No abstraction earns its keep here.

The same applies to the execution report: `Report()` returns a `*QueryReport`, and the
spec requires it to reach the reader *"typed distinctly from result claims"*. The
typing is the library's to preserve, not the server's to invent.

Vocabulary note, so nobody trips: Paper 02 §Filtered Reads describes `output.encoding`
as `json-seq`/`cbor-seq`, conflating serialization with framing. The normative spec
separates them — `encoding` is `json`|`cbor`, the media type frames the sequence — and
both `ranke.Output` and `openapi.yaml` follow the spec. Follow the spec; the paper
section predates the split.

### One snapshot per request

`GetArchive` returns an immutable `RA_k`. Resolving it once per request and answering
everything from it is what keeps a streaming query consistent while a merge advances
the head. Cheap, and the alternative is subtly wrong.

### A stack with no sequencer answers nothing

`GetArchive` is a `Sequencer` method and storage has no head, so `RA_k = (𝒰, k)` cannot
be opened from storage alone. Config builds the sequencer only
`if len(c.Sequencer) > 0`, so such a stack builds cleanly and then fails at the first
read. Either open the archive directly via `ranke.NewArchive(ctx, u, k)` with `k` from
the history's latest — the constructor exists, and storage-plus-history with no writer
is a useful read-only replica — or make the sequencer mandatory. Either way the silent
nil must go.

### Layer introspection needs no port change

`storage.build()` already parses each layer's name and type in order to compose the
stack, then discards them. Retaining that value answers `OpLayerList`/`OpLayerInfo`
outright. The information is config's, not the library's — `Universe.Capabilities()`
reports the *composed* abilities, which is a different question.

### The verification registry is ours because it is operational, not graph

ranke-go's `VerificationRun` is `Verified/Failures/Done/Err/Wait` — live, in-process,
no id, no persistence, no cancel beyond ctx. `domain.go` already models `ID`, `Status`,
`StartedAt`, retained `Failures`. Run identity, listing, cancellation and the
active-run limit are server state. Last in the order, being a subsystem rather than a
dispatch arm.

## Upstream asks

All three are ranke-go's by the razor. None should be worked around here.

**`Output.Encoding` must be honoured.** Declared at `query.go:121-126`, consumed
nowhere; `ResultJSON`/`ResultCBOR` are never referenced. The client asks for a
serialized form in the query and the engine ignores it, so a `QueryResult` arrives as
Go values. Every read arm waits on this: without it the server would have to serialise,
and serialising the canonical form outside the library is how a claim stops verifying
against its id.

**A wire format for a multi-claim body.** `openapi.yaml` promises "signed CBOR in
ranke's bundle format"; no such format exists. ranke-go has per-claim `Encode()` and
`DecodeClaim(id, b)`, and "bundle" upstream means a filesystem layout. Since an id is
`Sign(H(S(v)))` — a signature, not recomputable from the payload — a body must carry
explicit `(id, bytes)` pairs. Proposed: a self-framing CBOR sequence of
`[id, claim-bytes]`. Every client must produce it, so it belongs in the library.

**A per-claim read check during closure.** Paper 02 §Sequencer step 3 requires that
closing a contribution draw in referenced claims from other branches or the wider
Universe, and that *"read access to all branch-external claims is required"*. There is
no seam for it:

```go
// contribution.go:26
CompleteAndVerify(ctx context.Context) (VerifiedContribution, error)
```

No options, no hook. Steps 3 and 4 run as one traversal inside, and the only extension
point in the path is `g.Verify(ranke.WithTrusted(c.s.isCommitted))` — the sequencer's
own pruning, not the caller's. So ranke-db cannot authorize what the walk pulls in, and
contribute cannot satisfy the paper until `CompleteAndVerify` accepts a read check.

Two smaller observations from the same read, for whoever raises the above:
`admissible()` rejects `NodeBranches` but not limiting claims, though the paper reserves
both to the Sequencer; and step 6's `expires_after_request` → `contribution/expiry`
mint does not exist in `concurrent` at all, which also leaves `core-limiting-claims`
unsatisfiable by this backend.

## Risks / Trade-offs

- **Every read arm waits on `Output.Encoding`** → the read path is otherwise trivial,
  so this single upstream fix gates the bulk of the change. If it cannot land soon,
  the fallback is a temporary server-side encode, which puts canonical bytes in the
  wrong repo and should be marked as debt the day it is written.
- **`concurrent` holds its committed-id set in memory** and it "grows with the archive"
  (its own doc) → unbounded growth on a long-lived server. Upstream.
- **The contributor identity is minted, never looked up** → a fresh contributor claim
  each launch, its id reproducing only because `created_at` is pinned to the epoch. A
  key rotation therefore changes the merge identity silently instead of being rejected.

## Open Questions

- **Read-only mode: direct archive open, or mandatory sequencer?** Recommending the
  former.
- **Does `/health` take the signing identity from the sequencer's `GetContributor` or
  from the signer port?** Via the sequencer, a broken stack makes `/health` fail —
  precisely when it is wanted.
