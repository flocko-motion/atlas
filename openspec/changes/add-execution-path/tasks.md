## 1. Bump ranke-go to v0.4.0

- [x] 1.1 `go get github.com/flocko-motion/ranke-go@v0.4.0`, tidy, `make verify` green. v0.4.0 honours `Output.Encoding`: `EncodeResults` (`query_encode.go`) fills `ClaimEncoded`/`PathEncoded` via `Claim.EncodeCBOR(form)` / `EncodeJSON(form)`, called from `query_default.go:108` and `stack/query.go:56`.
- [x] 1.2 Absorb the `QueryResult` reshape: it is now a tagged union — `Kind` plus `ClaimId`/`PathId`/`ClaimNative`/`PathNative`/`ClaimEncoded`/`PathEncoded`. The duplicate `Content []byte` field is gone (inline content is in the claim; `Output.Content` still caps it). Check `endpoints_read.go`'s query mapping still holds, and that `ResultNative` — the new default, and what `""` means — is never what the endpoint asks for.

## 2. Serve the result set

A switch, not a component. The library's own words: *"Kind names the one field carrying
the payload, so a caller switches once and streams that field."*

- [x] 2.1 Switch once on `Result().Kind` and stream that field's bytes through unaltered — no re-encoding, no re-shaping.
- [x] 2.2 Add only what the media type requires: RS-prefixed per record for `application/json-seq` (RFC 7464), concatenated for `application/cbor-seq` (RFC 8742). A single JSON object and a raw blob are the degenerate cases.
- [x] 2.3 Append the library's execution report as the final record when the query asked for one, leaving it typed as the library typed it. Emit nothing when it did not.
- [x] 2.4 Pull and write incrementally, so neither a large result set nor a large blob is buffered whole.
- [x] 2.5 Test that `detail: claims` + `form: original` + `encoding: cbor` reaches the wire byte-identical and still verifies against the id.

## 3. A sequencer is required

Corrected during execution. The plan's premise — that `adapters/sequencer` documents
"omit the section to launch read-only" — is not in the code; no such promise exists. The
sequencer holds `k` and hands back every snapshot, so a stack without one answers
nothing. See `design.md` § Resolved while planning.

- [x] 3.1 Require a sequencer to serve: `requireServing` names it alongside signer, storage and endpoints, so the failure is at launch with a clear message.
- [x] 3.2 No second place configures the head timeline; it stays inside the sequencer section, which is the only thing that advances or tracks a head.
- [x] 3.3 Read-only, if ever wanted, is a sequencer backend that refuses to advance — behind the port, never around it.

## 4. Dispatch and the read arms

Each arm is one library call plus the serve loop from group 2. If an arm grows logic,
it is in the wrong repo.

- [x] 4.1 Resolve the archive snapshot once per request and hold it for that request.
- [x] 4.2 `OpHealthGet` — liveness plus the signing identity, taken from the signer port and not from the sequencer, so it still answers when no archive or sequencer could be assembled. Test that it answers on a deliberately broken stack.
- [x] 4.3 `OpBranchHead` → `GetBranch`; `OpBranchList` → `GetBranches`.
- [x] 4.4 `OpClaimGet` → `GetClaim`; `OpClaimContent` → `GetClaimContent`, streamed as raw bytes. Includes the `$universe` privileged form.
- [x] 4.5 `OpClaimQuery` → `Query`, serving the `ResultStream` through group 2 and appending the execution report when `Execution.Report` is set.
- [x] 4.6 Map library failures to the sentinels; absent and out-of-closure both resolve to not-found.
- [x] 4.7 Integration-test the read arms against a real stack, not a double.

## 5. Layer introspection

- [x] 5.1 Retain the layer names and types `storage.build()` already parses and discards. Config-side; no port change and no library call — `Universe.Capabilities()` answers a different question.
- [x] 5.2 Implement `OpLayerList` and `OpLayerInfo` against it, reporting name and type only.
- [x] 5.3 Test that no connection string, credential or path is reachable through it.

## 6. Raise the contribute blockers upstream

Both are ranke-go's by the razor. Do not work around them here. Recorded in
`design.md` § Upstream asks; two further findings from executing the read arms are added
below.

- [x] 6.1 Propose a wire format for a multi-claim contribution body: a self-framing CBOR sequence of `[id, claim-bytes]`. The id must be explicit, since it is a signature over the hash and not recomputable from the payload. Every client needs it, so it belongs in the library.
- [x] 6.2 Propose a per-claim read check on `CompleteAndVerify`, which today takes `(ctx)` and offers no hook. Without it the paper's step 3 — "read access to all branch-external claims is required" — cannot be enforced from this repo.
- [x] 6.3 Report the two smaller findings alongside: `admissible()` rejects `NodeBranches` but not limiting claims, though the paper reserves both to the Sequencer; and step 6's `expires_after_request` → `contribution/expiry` mint is absent from `concurrent`, leaving `core-limiting-claims` unsatisfiable by that backend.
- [x] 6.4 Found while building the read arms: `Archive.GetBranch`'s not-found is `errBranchNotFound`, which is unexported and wraps no exported sentinel, so no caller can classify it. `execute` asks `HasBranch` on the error path to tell "no such branch" from a real failure; wrapping `ErrNotFound` upstream would remove the extra call.
- [x] 6.5 Found while building the report record: `QueryReport` carries no JSON tags and no marshaller, so serving it verbatim puts Go field names (`StartedAt`, `Elapsed`, `Events`) on the wire, where `openapi.yaml` declares `startedAt`/`elapsedMs`/`events`. The report is served as the library types it, per the spec, so the contract's schema and the library's struct should be reconciled upstream rather than remapped here.

## 7. The contribute arm

Blocked on group 6, which is unresolved upstream: 7.1 has no wire format to decode
and 7.3 has no hook to wire. Left unbuilt rather than worked around.

- [ ] 7.1 Decode the request body into claims once the format from 6.1 exists.
- [ ] 7.2 Drive the six steps in order, delegating each to the library, and return the new head id with the contributed claim ids.
- [ ] 7.3 Wire the read check from 6.2 into the closure walk.
- [ ] 7.4 Report a head conflict only for a genuine irreconcilable clash. `concurrent` folds step 6 against the branch's live head, so two contributions opened at one base normally both succeed; reporting a conflict there would have clients retry a condition that did not occur.

## 8. The verification subsystem

Last: a subsystem with its own state, not another dispatch arm.

- [x] 8.1 Build the run registry — allocate an id, hold the report, track `running → complete | stopped | error`.
- [x] 8.2 Adapt `Archive.Verify`'s live handle onto that model, mapping `Verified`/`Failures`/`Done`/`Err` onto the retained report.
- [x] 8.3 Implement start, list, get, cancel and delete. Cancellation goes through context; a cancelled run reports stopped and keeps its partial findings.
- [x] 8.4 Enforce the active-run limit, refusing as busy beyond it.
- [x] 8.5 Test that a completed run's report survives retrieval and a cancelled run keeps what it found.
