## 1. The generated client reaches the bundle

- [ ] 1.1 Emit a copy of `openapi.gen.ts` into `frontend/src/core/data/` from the root `make generate`, committed like `explorer.html` is, so `make -C frontend` needs nothing from the root target.
- [ ] 1.2 Add `@flocko-motion/ranke` to `frontend/package.json`, pinned to the version whose `Claim` the code is written against.
- [ ] 1.3 Check that the generated file type-checks under `frontend/tsconfig.json`, which is stricter than the generator assumes — it carries `verbatimModuleSyntax`, `noUnusedLocals` and `allowImportingTsExtensions`.

## 2. Transport through the client

- [ ] 2.1 Construct an `Api` per connection, taking `baseUrl` from the connection and headers from `authHeaders()` via `baseApiParams`.
- [ ] 2.2 Replace `RestSource.branches()`'s two hand-built reads with `listBranches()` and `getArchiveInfo()`, and `headOf()` with `getBranchHead()`.
- [ ] 2.3 Replace the content read with `getBranchClaimContent()` and `getArchiveClaimContent()`, dropping the path-building conditional.
- [ ] 2.4 Delete `getJSON()` and every literal route string.
- [ ] 2.5 Call `POST /query` through the generated route with the response format left unset, so the client answers with the `Response` whose body the library then reads. Error bodies arrive as a thrown response rather than as text, so the failure paths change with it.

## 3. The claim comes from the library

- [ ] 3.1 Replace `MockClaim` and `MockEdge` with the library's `Claim` and `Edge` throughout `core/`. Keep `contribution`, `branch` and `label` beside the claim, not inside a type restating it.
- [ ] 3.2 Delete `WireClaim` and `claimsFromSequence`; decode with `codec_json` through the library's reader.
- [ ] 3.3 Replace `claimsFromStream` with `readClaims` over the response body, or `SeqReader` where the push loop reads better, and take the progress count from `bytesRead` rather than from `content-length`.
- [ ] 3.4 Delete `idsFromSequence`; an ids-only read decodes through the same reader.
- [ ] 3.5 Replace `classOf()` with `typeClass`, and drop the `Date.parse` in favour of `createdAtMs`.
- [ ] 3.6 Replace the `startsWith` edge filter in `graph/build.ts:184` with `matchTypeList`, which implements the contract's glob rules including a leading `-`.
- [ ] 3.7 Delete `NODE_CLASSES`' class list in favour of `NodeClasses`; keep the generator's invented subtypes, which are open vocabulary.

## 4. The query comes from the standard

- [ ] 4.1 Build the query body as the library's `Query`, serialised by `query_codec`.
- [ ] 4.2 Keep `core/query.ts`'s own type as the view's state. It carries `classes`, which is a view selection rather than a query field, so the two are separate shapes and one translates into the other at request time.

## 5. The generator builds real claims

- [ ] 5.1 Build generated claims through `claim_builder`, so mock and server data are one type on one code path.
- [ ] 5.2 Re-measure the benches. Generation now encodes where it previously assembled object literals, and `results/*.json` is committed on purpose.

## 6. Fix what the comparison found

- [ ] 6.1 Make `overflow` required on `OutputContent` in `openapi/openapi.yaml`, matching `rql.schema.json`, and regenerate.
- [ ] 6.2 Correct `frontend/README.md:44`, `NotWiredError` (`source.ts:70`) and `RestSource`'s docstring (`source.ts:178`): reading claim bodies from a connection has worked since `add-branch-selection`.
- [ ] 6.3 Correct `frontend/README.md:421-427`, which lists `src/mock/graph.ts` and `src/explorer/main.ts` against a tree that holds neither.

## 7. Tests

- [ ] 7.1 A claim decoded from a server response and one from the generator are the same type, read by one code path.
- [ ] 7.2 A progress count climbs before the body completes, driven by a chunked stream.
- [ ] 7.3 Edge filtering matches the contract's glob rules, exclusions included, against the library's matcher rather than a local one.
- [ ] 7.4 No explorer source file holds a route path or a claim field name as a literal — the check that keeps this from growing back.
- [ ] 7.5 `make -C frontend check` and `make -C frontend test` pass, and `make verify` passes at the root after the spec fix.
