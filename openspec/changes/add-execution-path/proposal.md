## Why

`core.execute` is a stub returning `ErrNotImplemented` for all fifteen operations.
Everything around it is built: the REST endpoints are complete, `Request` carries every
ingress field, `Stream` fixes the response contract, and `adapters/sequencer` binds
`dev`/`concurrent` against `ranke-go v0.3.2`.

The stage is small, because ranke-go does the work. Each read is one call on the
archive — `Query(ctx, q) → ResultStream`, `GetClaim`, `GetClaimContent`, `GetBranch`,
`GetBranches`, `Verify` — and a contribution is the paper's six steps, each of them a
library call. The query engine, the walk, the filters, the ordering, the execution
report, the merge and the verification pass all exist upstream and are finished.

What is left in `execute` is the seam: turning an `Operation` into the call that
answers it, and turning what comes back into bytes on the wire. That is four things.

## What Changes

- Pin **dispatch**: one path per operation, resolving the archive snapshot once per
  request so results stay consistent while the head advances.
- Pin that the server **serves the result set rather than shaping it**. The library
  already honours four output axes — `Shape`, `Detail`, `Form`, `Content`. The fifth,
  `Output.Encoding`, is declared at `query.go:121-126` and consumed nowhere, so a
  `QueryResult` arrives as Go values even when the query asked for cbor. That is an
  upstream fix, not a server abstraction: the canonical form is the library's, and
  serialising it here is how a claim stops verifying against its id. Once the axis is
  honoured, serving is a loop — write the bytes, add the separator the media type
  requires, set the content type.
- Pin **error mapping**: library errors resolve to the sentinels `Categorize` already
  translates, and an out-of-closure read is reported as not-found, indistinguishable
  from absent.
- Pin the **verification-run registry**. ranke-go's `VerificationRun` is a live
  in-process handle — `Verified/Failures/Done/Err/Wait`, no id, no persistence, no
  cancel — while `domain.go` already models ids, status and retained reports. So run
  identity, listing, cancellation and the active-run limit are the server's
  operational state, not graph behaviour.

## Capabilities

### New Capabilities

- `core-execution`: the execute stage — dispatch and the archive snapshot seam,
  serving the library's result set to the wire, error mapping, and the
  verification-run registry.

## Impact

- **`internal/core/`**: `execute` grows from a stub into dispatch plus the rendering
  unit; the verification registry is new.
- **`config/`**: settles what a stack without a sequencer section means. `GetArchive`
  is a `Sequencer` method and storage has no head, so such a stack currently builds
  cleanly and then answers nothing.
- **`adapters/storage/`**: `build()` already parses each layer's name and type and
  then discards them. Retaining that value answers `OpLayerList`/`OpLayerInfo`; it is
  a config-side detail, not a port change.
- **Not affected**: `core-query`, `core-contribution`, `core-verification`,
  `core-access`, `adapter-storage` and `adapter-sequencer`. Their semantics are
  ranke-go's or already specified; this change restates none of them.
- **Blocked upstream, not planned here**: three asks, recorded in `design.md` with
  tasks to raise them. `Output.Encoding` must be honoured before any read arm can
  serve bytes; and the contribute arm needs a wire format for a multi-claim body and a
  per-claim read check during closure, neither of which ranke-go offers.
