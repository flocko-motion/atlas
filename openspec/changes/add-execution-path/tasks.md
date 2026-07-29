## 1. The rendering unit

First, because all fifteen operations render through it. Growing it out of whichever
arm is written first is how it ends up wrong.

- [ ] 1.1 Define the item contract: one result serialising itself into a writer under the request's output axes. Core-internal — it must not surface in `Stream`, which the endpoint sees.
- [ ] 1.2 Make the item kind open from the start, so a stream can carry a result and an execution report and keep them distinguishable by type. Retrofitting this means a special case in every framing.
- [ ] 1.3 Define the framing contract and implement four: `application/json-seq` (RFC 7464, RS-prefixed), `application/cbor-seq` (RFC 8742, concatenated), single JSON object, raw blob passthrough.
- [ ] 1.4 Check the axes against `ranke.Output` — `Shape{single,path}`, `Detail{id,graph,claims}`, `Form{original,materialized}`, `Content{Max,Overflow}`, `ResultEncoding{json,cbor}` — so the wire, the library and the renderer share one vocabulary. Note that Paper 02 §Filtered Reads conflates encoding with framing; the normative spec separates them and the library follows the spec.
- [ ] 1.5 Test that `detail: claims` + `form: original` + `encoding: cbor` passes the library's canonical bytes through unaltered and still verifies against the id.
- [ ] 1.6 Test laziness: a large result set streams without being buffered whole.

## 2. Settle the read-only question

- [ ] 2.1 Decide between opening the archive directly from the history's latest head when no sequencer is configured, and making the sequencer mandatory. `design.md` recommends the former.
- [ ] 2.2 Implement it in `config.build`, so a stack without a sequencer either works read-only or fails at launch with a clear message — never builds cleanly and then fails at the first read.

## 3. Dispatch and the read arms

Each arm is one library call plus rendering. If an arm grows logic, it is in the wrong
repo.

- [ ] 3.1 Resolve the archive snapshot once per request and hold it for that request.
- [ ] 3.2 `OpHealthGet` — liveness plus the signing identity, sourced so it still answers when the archive cannot be opened (see the open question in `design.md`).
- [ ] 3.3 `OpBranchHead` → `GetBranch`; `OpBranchList` → `GetBranches`.
- [ ] 3.4 `OpClaimGet` → `GetClaim`; `OpClaimContent` → `GetClaimContent`, streamed as raw bytes. Includes the `$universe` privileged form.
- [ ] 3.5 `OpClaimQuery` → `Query`, rendering the `ResultStream` through the item/framing pair and emitting the execution report when `Execution.Report` is set.
- [ ] 3.6 Map library failures to the sentinels; absent and out-of-closure both resolve to not-found.
- [ ] 3.7 Integration-test the read arms against a real stack, not a double.

## 4. Layer introspection

- [ ] 4.1 Retain the layer names and types `storage.build()` already parses and discards. Config-side; no port change and no library call — `Universe.Capabilities()` answers a different question.
- [ ] 4.2 Implement `OpLayerList` and `OpLayerInfo` against it, reporting name and type only.
- [ ] 4.3 Test that no connection string, credential or path is reachable through it.

## 5. Raise the contribute blockers upstream

Both are ranke-go's by the razor. Do not work around them here.

- [ ] 5.1 Propose a wire format for a multi-claim contribution body: a self-framing CBOR sequence of `[id, claim-bytes]`. The id must be explicit, since it is a signature over the hash and not recomputable from the payload. Every client needs it, so it belongs in the library.
- [ ] 5.2 Propose a per-claim read check on `CompleteAndVerify`, which today takes `(ctx)` and offers no hook. Without it the paper's step 3 — "read access to all branch-external claims is required" — cannot be enforced from this repo.
- [ ] 5.3 Report the two smaller findings alongside: `admissible()` rejects `NodeBranches` but not limiting claims, though the paper reserves both to the Sequencer; and step 6's `expires_after_request` → `contribution/expiry` mint is absent from `concurrent`, leaving `core-limiting-claims` unsatisfiable by that backend.

## 6. The contribute arm

Blocked on group 5. Nothing above depends on it.

- [ ] 6.1 Decode the request body into claims once the format from 5.1 exists.
- [ ] 6.2 Drive the six steps in order, delegating each to the library, and return the new head id with the contributed claim ids.
- [ ] 6.3 Wire the read check from 5.2 into the closure walk.
- [ ] 6.4 Report a head conflict only for a genuine irreconcilable clash. `concurrent` folds step 6 against the branch's live head, so two contributions opened at one base normally both succeed; reporting a conflict there would have clients retry a condition that did not occur.

## 7. The verification subsystem

Last: a subsystem with its own state, not another dispatch arm.

- [ ] 7.1 Build the run registry — allocate an id, hold the report, track `running → complete | stopped | error`.
- [ ] 7.2 Adapt `Archive.Verify`'s live handle onto that model, mapping `Verified`/`Failures`/`Done`/`Err` onto the retained report.
- [ ] 7.3 Implement start, list, get, cancel and delete. Cancellation goes through context; a cancelled run reports stopped and keeps its partial findings.
- [ ] 7.4 Enforce the active-run limit, refusing as busy beyond it.
- [ ] 7.5 Test that a completed run's report survives retrieval and a cancelled run keeps what it found.
