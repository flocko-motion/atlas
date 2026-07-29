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
- The rendering unit: library values → framed bytes with a content type.
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

### The rendering unit is the only real design decision

`stream.go` defers it deliberately — "that rendering unit is core's own; it lands with
the execute-stage engines". It is the spine: every operation renders through it.

The normative spec fixes the algebra, so this is constrained rather than open:

```
Result  = Element | [Element]        // per output.shape: single | path
Element = Id | Graph | Claim         // per output.detail
```

with `form` (original | materialized) and `content` (max + overflow) as value
properties, and `encoding` (json | cbor) orthogonal to all of it. The *framing* is
separate again — `application/json-seq` (RS-prefixed), `application/cbor-seq`
(concatenated), a single JSON object, a raw blob.

One type per combination is combinatorial. Two orthogonal abstractions instead: an
**item** renders itself into a writer under the output axes; a **framing** writes a
sequence of items with its separators and declares the content type. A by-id read
yields one item; `/health` yields one item; a query yields many from a `ResultStream`.
Same machinery throughout.

The constraint that is easy to miss and expensive to retrofit — the spec requires:

> After the last element — and only if `execution.report` was set — the stream carries
> a final *report* record... **It is typed distinctly from result claims so a reader
> never mistakes it for data.**

So a stream carries items *of more than one kind*. An abstraction where "item" quietly
means "claim" cannot carry the report without a special case in every framing.

Vocabulary note, so nobody trips: Paper 02 §Filtered Reads describes `output.encoding`
as `json-seq`/`cbor-seq`, conflating serialization with framing. The normative spec
separates them — `encoding` is `json`|`cbor`, the media type frames — and both
`ranke.Output` and `openapi.yaml` follow the spec. Follow the spec; the paper section
predates the split.

### Only the canonical combination is verifiable, and the renderer must not touch it

`detail: claims` + `form: original` + `encoding: cbor` reproduces the canonical
serialization the id was computed over. That is the library's output; the renderer's
job is to pass it through unaltered rather than re-encode it. Every other combination
is a convenience projection and must not be presented as verifiable.

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

## Upstream asks — the contribute arm depends on these

Both are ranke-go's by the razor. Neither should be worked around here.

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

- **The rendering unit has the widest blast radius** → all fifteen operations render
  through it; if it is wrong they all change. Hence deciding it, including the
  heterogeneous-item constraint, before the first arm is written.
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
