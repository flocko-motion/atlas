## 1. Bump ranke-go (v0.4.0, v0.5.0 for the wire codec, v0.6.0 for the declared header)

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

## 6. The wire format, and what is left upstream

Corrected twice during execution. 6.1 was framed as an upstream blocker; it was not. The
format was then defined here — and ranke-go v0.5.0 shipped it, so the codec is the
library's and this repo holds none of it.

- [x] 6.1 ranke-go v0.5.0 provides the contribution codec (`codec_wire.go`): `WireWriter`/`WireReader` over a CBOR sequence of `[0, id, claim-bytes, branch]` and `[1, hash, blob]`, with `WireMediaType`. The server reads it and defines no framing of its own; `openapi.yaml` documents it as the binding.
- [x] 6.2 A shared codec was the ask, and v0.5.0 answers it — a client produces a contribution body with the library.
- [ ] 6.3 Ask ranke-go for a per-claim read check on `CompleteAndVerify`, which takes `(ctx)` and offers no hook. Without it the paper's step 3 — "read access to all branch-external claims is required" — cannot be enforced from this repo. The one part of contribute still unsatisfiable here.
- [x] 6.8 Answered by ranke-go v0.6.0: a stream declares its branches in a leading `[2, [branch, …]]` header, `WireReader.Branches()` reads it without draining, and a claim naming an undeclared branch is refused — so the declaration binds. `AddWire` takes the reader, so the caller that checked the header hands on the same one. This arm is three steps: read the declaration, authorize it, `AddWire`. Nothing is buffered, so a contribution of any size streams.
- [x] 6.4 Report alongside: `admissible()` rejects `NodeBranches` but not limiting claims, though the paper reserves both to the Sequencer; and step 6's `expires_after_request` → `contribution/expiry` mint is absent from `concurrent`, leaving `core-limiting-claims` unsatisfiable by that backend.
- [x] 6.5 `Archive.GetBranch`'s not-found is `errBranchNotFound`, unexported and wrapping no exported sentinel, so no caller can classify it. `execute` asks `HasBranch` on the error path; wrapping `ErrNotFound` upstream would remove the extra call.
- [x] 6.6 `QueryReport` carries no JSON tags, so serving it as the library types it puts Go field names on a wire whose contract declares `startedAt`/`elapsedMs`/`events`. Reconcile upstream rather than remapping here.
- [x] 6.7 The height of an initial node diverges: the normative spec §R-HEIGHT puts it at 1 ("every height is therefore ≥ 1, which leaves 0 free to mean unbounded"), while the verifier requires 0 and rejects 1. One of the two is wrong.

## 7. The contribute arm

Built on the library's codec. Only the step-3 read check (6.3) is missing, and it has no
seam upstream.

- [x] 7.1 Read the body with `ranke.NewWireReader`, grouping claims by the branch each record names. A record that does not decode fails the whole body, since a contribution is atomic.
- [x] 7.2 Drive the steps in order — `PutContents`, `NewContribution`, `AddClaims` per branch, `CompleteAndVerify`, `Persist`, `Merge` — and return the new head with the contributed ids.
- [ ] 7.3 Wire the read check from 6.3 into the closure walk. Blocked: `CompleteAndVerify(ctx)` offers no hook.
- [x] 7.4 Report a head conflict only for a genuine irreconcilable clash. `concurrent` folds step 6 against the branch's live head, so nothing here reports a conflict for two contributions opened at one base.
- [x] 7.5 A body names the branch per claim, so one contribution may advance several. The **C** right is checked on every branch the body writes to, before anything lands — the shape `core-access` already gives the cross-branch delete rule.
- [x] 7.6 Use `AddWire` over the reader whose header was already checked, so the arm streams and holds nothing. Retired the record loop that existed only to learn the branches.

## 8. The verification subsystem

Last: a subsystem with its own state, not another dispatch arm.

- [x] 8.1 Build the run registry — allocate an id, hold the report, track `running → complete | stopped | error`.
- [x] 8.2 Adapt `Archive.Verify`'s live handle onto that model, mapping `Verified`/`Failures`/`Done`/`Err` onto the retained report.
- [x] 8.3 Implement start, list, get, cancel and delete. Cancellation goes through context; a cancelled run reports stopped and keeps its partial findings.
- [x] 8.4 Enforce the active-run limit, refusing as busy beyond it.
- [x] 8.5 Test that a completed run's report survives retrieval and a cancelled run keeps what it found.
