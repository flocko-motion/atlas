## 1. Raise `Output.Encoding` upstream — gates every read arm

- [ ] 1.1 Report it: `Output.Encoding` is declared at `query.go:121-126` and consumed nowhere — `ResultJSON`/`ResultCBOR` are never referenced — while `Shape`, `Detail`, `Form` and `Content` are all honoured by the engines. So a client asking for cbor gets Go values. The engine should answer the axis it accepts.
- [ ] 1.2 Do not compensate here. Serialising the canonical form outside the library is how a claim stops verifying against its id. If the fix cannot land soon, any temporary server-side encode is marked as debt the day it is written.

## 2. Serve the result set

Once 1.1 lands this is a loop, not a component.

- [ ] 2.1 Write each result's bytes through unaltered, adding only the separator the media type requires: RS-prefixed per record for `application/json-seq` (RFC 7464), concatenated for `application/cbor-seq` (RFC 8742). A single JSON object and a raw blob are the degenerate cases.
- [ ] 2.2 Append the library's execution report as the final record when the query asked for one, leaving it typed as the library typed it. Emit nothing when it did not.
- [ ] 2.3 Pull and write incrementally, so neither a large result set nor a large blob is buffered whole.
- [ ] 2.4 Test that `detail: claims` + `form: original` + `encoding: cbor` reaches the wire byte-identical and still verifies against the id.

## 3. Settle the read-only question

- [ ] 3.1 Decide between opening the archive directly from the history's latest head when no sequencer is configured, and making the sequencer mandatory. `design.md` recommends the former.
- [ ] 3.2 Implement it in `config.build`, so a stack without a sequencer either works read-only or fails at launch with a clear message — never builds cleanly and then fails at the first read.

## 4. Dispatch and the read arms

Each arm is one library call plus the serve loop from group 2. If an arm grows logic,
it is in the wrong repo.

- [ ] 4.1 Resolve the archive snapshot once per request and hold it for that request.
- [ ] 4.2 `OpHealthGet` — liveness plus the signing identity, sourced so it still answers when the archive cannot be opened (see the open question in `design.md`).
- [ ] 4.3 `OpBranchHead` → `GetBranch`; `OpBranchList` → `GetBranches`.
- [ ] 4.4 `OpClaimGet` → `GetClaim`; `OpClaimContent` → `GetClaimContent`, streamed as raw bytes. Includes the `$universe` privileged form.
- [ ] 4.5 `OpClaimQuery` → `Query`, serving the `ResultStream` through group 2 and appending the execution report when `Execution.Report` is set.
- [ ] 4.6 Map library failures to the sentinels; absent and out-of-closure both resolve to not-found.
- [ ] 4.7 Integration-test the read arms against a real stack, not a double.

## 5. Layer introspection

- [ ] 5.1 Retain the layer names and types `storage.build()` already parses and discards. Config-side; no port change and no library call — `Universe.Capabilities()` answers a different question.
- [ ] 5.2 Implement `OpLayerList` and `OpLayerInfo` against it, reporting name and type only.
- [ ] 5.3 Test that no connection string, credential or path is reachable through it.

## 6. Raise the contribute blockers upstream

Both are ranke-go's by the razor. Do not work around them here.

- [ ] 6.1 Propose a wire format for a multi-claim contribution body: a self-framing CBOR sequence of `[id, claim-bytes]`. The id must be explicit, since it is a signature over the hash and not recomputable from the payload. Every client needs it, so it belongs in the library.
- [ ] 6.2 Propose a per-claim read check on `CompleteAndVerify`, which today takes `(ctx)` and offers no hook. Without it the paper's step 3 — "read access to all branch-external claims is required" — cannot be enforced from this repo.
- [ ] 6.3 Report the two smaller findings alongside: `admissible()` rejects `NodeBranches` but not limiting claims, though the paper reserves both to the Sequencer; and step 6's `expires_after_request` → `contribution/expiry` mint is absent from `concurrent`, leaving `core-limiting-claims` unsatisfiable by that backend.

## 7. The contribute arm

Blocked on group 6. Nothing above depends on it.

- [ ] 7.1 Decode the request body into claims once the format from 6.1 exists.
- [ ] 7.2 Drive the six steps in order, delegating each to the library, and return the new head id with the contributed claim ids.
- [ ] 7.3 Wire the read check from 6.2 into the closure walk.
- [ ] 7.4 Report a head conflict only for a genuine irreconcilable clash. `concurrent` folds step 6 against the branch's live head, so two contributions opened at one base normally both succeed; reporting a conflict there would have clients retry a condition that did not occur.

## 8. The verification subsystem

Last: a subsystem with its own state, not another dispatch arm.

- [ ] 8.1 Build the run registry — allocate an id, hold the report, track `running → complete | stopped | error`.
- [ ] 8.2 Adapt `Archive.Verify`'s live handle onto that model, mapping `Verified`/`Failures`/`Done`/`Err` onto the retained report.
- [ ] 8.3 Implement start, list, get, cancel and delete. Cancellation goes through context; a cancelled run reports stopped and keeps its partial findings.
- [ ] 8.4 Enforce the active-run limit, refusing as busy beyond it.
- [ ] 8.5 Test that a completed run's report survives retrieval and a cancelled run keeps what it found.
