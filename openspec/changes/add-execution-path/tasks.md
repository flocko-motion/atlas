## 1. Bump ranke-go to v0.4.0

- [ ] 1.1 `go get github.com/flocko-motion/ranke-go@v0.4.0`, tidy, `make verify` green. v0.4.0 honours `Output.Encoding`: `EncodeResults` (`query_encode.go`) fills `ClaimEncoded`/`PathEncoded` via `Claim.EncodeCBOR(form)` / `EncodeJSON(form)`, called from `query_default.go:108` and `stack/query.go:56`.
- [ ] 1.2 Absorb the `QueryResult` reshape: it is now a tagged union — `Kind` plus `ClaimId`/`PathId`/`ClaimNative`/`PathNative`/`ClaimEncoded`/`PathEncoded`. The duplicate `Content []byte` field is gone (inline content is in the claim; `Output.Content` still caps it). Check `endpoints_read.go`'s query mapping still holds, and that `ResultNative` — the new default, and what `""` means — is never what the endpoint asks for.

## 2. Serve the result set

A switch, not a component. The library's own words: *"Kind names the one field carrying
the payload, so a caller switches once and streams that field."*

- [ ] 2.1 Switch once on `Result().Kind` and stream that field's bytes through unaltered — no re-encoding, no re-shaping.
- [ ] 2.2 Add only what the media type requires: RS-prefixed per record for `application/json-seq` (RFC 7464), concatenated for `application/cbor-seq` (RFC 8742). A single JSON object and a raw blob are the degenerate cases.
- [ ] 2.3 Append the library's execution report as the final record when the query asked for one, leaving it typed as the library typed it. Emit nothing when it did not.
- [ ] 2.4 Pull and write incrementally, so neither a large result set nor a large blob is buffered whole.
- [ ] 2.5 Test that `detail: claims` + `form: original` + `encoding: cbor` reaches the wire byte-identical and still verifies against the id.

## 3. Read-only launch

Decided: a stack with no sequencer section serves read-only rather than failing. See
`design.md` § Resolved while planning.

- [ ] 3.1 When no sequencer is configured, open the archive directly — `ranke.NewArchive(ctx, u, k)` with `k` from `History.Latest()` — so storage-plus-history serves reads with no writer.
- [ ] 3.2 Build the history whether or not a sequencer is configured; it is the only source of `k`. A configuration with neither sequencer nor history fails at launch with a clear message.
- [ ] 3.3 Refuse every write operation on such a stack at the dispatch boundary, so the refusal is one explicit check rather than a nil dereference somewhere downstream.

## 4. Dispatch and the read arms

Each arm is one library call plus the serve loop from group 2. If an arm grows logic,
it is in the wrong repo.

- [ ] 4.1 Resolve the archive snapshot once per request and hold it for that request.
- [ ] 4.2 `OpHealthGet` — liveness plus the signing identity, taken from the signer port and not from the sequencer, so it still answers when no archive or sequencer could be assembled. Test that it answers on a deliberately broken stack.
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
