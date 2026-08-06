## 1. The generated client reaches the bundle

- [x] 1.1 Emit a copy of `openapi.gen.ts` into `frontend/src/core/data/` from the root `make generate`, committed like `explorer.html` is, so `make -C frontend` needs nothing from the root target.
- [x] 1.2 Add `@flocko-motion/ranke` to `frontend/package.json`, pinned to the version whose `Claim` the code is written against.
- [x] 1.3 Check that the generated file type-checks under `frontend/tsconfig.json`, which is stricter than the generator assumes — it carries `verbatimModuleSyntax`, `noUnusedLocals` and `allowImportingTsExtensions`.

## 2. Transport through the client

- [x] 2.1 Construct an `Api` per connection, taking `baseUrl` from the connection and headers from `authHeaders()` via `baseApiParams`.
- [x] 2.2 Replace `RestSource.branches()`'s two hand-built reads with `listBranches()` and `getArchiveInfo()`, and `headOf()` with `getBranchHead()`.
- [x] 2.3 Replace the content read with `getBranchClaimContent()` and `getArchiveClaimContent()`, dropping the path-building conditional.
- [x] 2.4 Delete `getJSON()` and every literal route string.
- [x] 2.5 Call `POST /query` through the generated route with the response format left unset, so the client answers with the `Response` whose body the library then reads. Error bodies arrive as a thrown response rather than as text, so the failure paths change with it.

## 3. The claim comes from the library

- [x] 3.1 Replace `MockClaim` and `MockEdge` with the library's `Claim` and `Edge` throughout `core/`. Keep `contribution`, `branch` and `label` beside the claim, not inside a type restating it.
- [x] 3.2 Delete `WireClaim` and `claimsFromSequence`; decode with `codec_json` through the library's reader.
- [x] 3.3 Replace `claimsFromStream` with `readClaims` over the response body, or `SeqReader` where the push loop reads better, and take the progress count from `bytesRead` rather than from `content-length`.
- [x] 3.4 Delete `idsFromSequence`; an ids-only read decodes through the same reader.
      Was blocked upstream — the library's reader decoded claim records only, so an
      `output.detail: id` sequence of bare strings had no reader and `idsFromSequence` stood in
      as a stopgap naming its owner. ranke-ts 0.3.0 publishes `readIds` (plus `readRecords`,
      `decodeResultRecord` and a raw framing reader), so the stopgap is gone and no framing rule
      is left in this repo.
- [x] 3.5 Replace `classOf()` with `typeClass`, and drop the `Date.parse` in favour of `createdAtMs`.
- [x] 3.6 Replace the `startsWith` edge filter in `graph/build.ts:184` with `matchTypeList`, which implements the contract's glob rules including a leading `-`.
- [x] 3.7 Delete `NODE_CLASSES`' class list in favour of `NodeClasses`; keep the generator's invented subtypes, which are open vocabulary.

## 4. The query comes from the standard

- [x] 4.1 Build the query body as the library's `Query`, serialised by `query_codec`.
- [x] 4.2 Keep `core/query.ts`'s own type as the view's state. It carries `classes`, which is a view selection rather than a query field, so the two are separate shapes and one translates into the other at request time.

## 5. The generator builds real claims

- [x] 5.1 Build generated claims through `claim_builder`, so mock and server data are one type on one code path.
- [x] 5.2 Re-measure the benches. Generation now encodes where it previously assembled object literals, and `results/*.json` is committed on purpose.
      Measured at ranke-ts 0.2.1, generation was ~25× slower (30.3 s per 100k) and a decoded
      claim ~8× the heap (770 MiB per 100k, was 97 MiB) — enough that the container's 2 GiB cap
      bound. Taking both apart put the causes upstream rather than in the model: about half the
      build time was `claim_builder` encoding each record three times and decoding it back, and
      ~74 % of the claim heap was ConsString ropes from `base32Encode`'s `out += char` loop.
      Both are fixed in **0.3.0**; this change pins **0.4.0**, which measures the same: generation is 8 400 claims/s
      (~0.12 ms each) and a claim is 1 813 B — ~1.4× the old hand-rolled shape, which is what
      holding the model costs. Every table in `frontend/README.md` re-measured at the committed
      shape, `results/graph-bench-300k.json` restored, the granularity sweep back at 100k.
      Build and layout land on the pre-library figures within this host's ~28 % run-to-run
      noise, which was measured rather than assumed.

## 6. Fix what the comparison found

- [x] 6.1 Make `overflow` required on `OutputContent` in `openapi/openapi.yaml`, matching `rql.schema.json`, and regenerate.
      **Nothing to change.** `openapi.yaml` holds no copy of the query language: `Query` is a
      `$ref` to `./rql.schema.json#/$defs/Query`, which requires `[max, overflow]`, and every
      generated artifact agrees (`openapi.gen.yaml`, `openapi.gen.go`, `openapi.gen.ts`).
      `make check-generated` reports the artifacts current, so the drift is already gone.
- [x] 6.2 Correct `frontend/README.md:44`, `NotWiredError` (`source.ts:70`) and `RestSource`'s docstring (`source.ts:178`): reading claim bodies from a connection has worked since `add-branch-selection`.
- [x] 6.3 Correct `frontend/README.md:421-427`, which lists `src/mock/graph.ts` and `src/explorer/main.ts` against a tree that holds neither.

## 7. Tests

- [x] 7.1 A claim decoded from a server response and one from the generator are the same type, read by one code path.
- [x] 7.2 A progress count climbs before the body completes, driven by a chunked stream.
- [x] 7.3 Edge filtering matches the contract's glob rules, exclusions included, against the library's matcher rather than a local one.
- [x] 7.4 No explorer source file holds a route path or a claim field name as a literal — the check that keeps this from growing back.
- [x] 7.5 `make -C frontend check` and `make -C frontend test` pass, and `make verify` passes at the root after the spec fix.
