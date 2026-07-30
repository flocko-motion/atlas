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

### There is nothing to render — the engine serves the result already shaped

ranke-go owns query execution entirely, and as of **v0.4.0** that includes
serialisation. Every output axis is answered by the engine:

| Axis | Honoured at |
|---|---|
| `Shape` | `query_default.go:45,87`, `archive.go:148`, `neo4j/query.go:22` |
| `Detail` | `query_default.go:88`, `neo4j/query.go:43,49` |
| `Form` | `stack/query.go:27` |
| `Content` | `query_default.go:98`, `stack/query.go:80` |
| `Encoding` | `query_encode.go` — `EncodeResults`, called from `query_default.go:108` and `stack/query.go:56` |

`QueryResult` is now a tagged union: a `Kind` plus `ClaimId` / `PathId` /
`ClaimNative` / `PathNative` / `ClaimEncoded` / `PathEncoded`, with
`Claim.EncodeCBOR(form)` and `EncodeJSON(form)` producing the bytes. `ResultNative`
(also what `""` means) returns the Go objects; `ResultCBOR` and `ResultJSON` fill the
encoded fields.

So the server's job is a switch, and the library says so itself:

> `QueryResult` is one reached claim, shaped per `Output`. **Kind names the one field
> carrying the payload, so a caller switches once and streams that field.**

Switch once on `Kind`, stream that field, add the separator the media type requires —
RFC 7464 prefixes each record with RS, RFC 8742 concatenates — and set the content
type. A single object and a raw blob are degenerate cases of the same loop. No
abstraction earns its keep, and building one would put the canonical form in the
server's hands: `detail: claims` + `form: original` + `encoding: cbor` *is* the
serialization a claim's id is computed over, so a server that re-encodes it is a
server that decides identity.

The same applies to the execution report: `Report()` returns a `*QueryReport`, and the
spec requires it to reach the reader *"typed distinctly from result claims"*. The
typing is the library's to preserve, not the server's to invent.

v0.4.0 also drops `QueryResult.Content []byte`, which was a second copy of bytes the
claim already carried — inline content is inside the claim by definition. `Output.Content`
still caps it and still decides overflow; only the duplicate field is gone.

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

Both are ranke-go's by the razor, and both block the contribute arm only — the read
arms are unblocked by v0.4.0. Neither should be worked around here. Tracked as
`td-0da051`.

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

- **The v0.4.0 bump reshapes `QueryResult`** → it becomes a tagged union and loses its
  `Content` field. Small, but it lands before any arm is written, so the bump is task 1
  with `make verify` green as its gate rather than a step inside the read work.
- **`concurrent` holds its committed-id set in memory** and it "grows with the archive"
  (its own doc) → unbounded growth on a long-lived server. Upstream.
- **The contributor identity is minted, never looked up** → a fresh contributor claim
  each launch, its id reproducing only because `created_at` is pinned to the epoch. A
  key rotation therefore changes the merge identity silently instead of being rejected.

## Resolved while planning

Two questions were open through drafting and are settled here, so nothing is left for
whoever executes to guess at.

**Read-only opens the archive directly.** Not the alternative of making the sequencer
mandatory. `adapters/sequencer` already documents the promise — *"omit the section to
launch read-only"* — so honouring it is finishing something started, not adding a
feature. `ranke.NewArchive(ctx, u, k)` is public and `History.Latest()` supplies `k`,
so the cost is small, and storage-plus-history with no writer is a real deployment: a
replica, or a frozen bundle served for inspection. The consequence to accept is that
history becomes required in every configuration, read-only included — consistent with
it being the only source of `k`.

**`/health` takes the signing identity from the signer port.** Not from the
sequencer's `GetContributor`. Health exists to answer when things are broken, and
routing it through the sequencer means a stack that failed to assemble one cannot
report why — the single case where the endpoint matters most. The signer is also the
authority the identity derives from: the sequencer's contributor is minted *from* the
signer's public key, so asking the signer is asking the source rather than a
downstream copy.
