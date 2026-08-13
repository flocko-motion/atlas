## Why

The Ranke Explorer hand-writes both the graph model and the contract it talks to.
`frontend/src/core/mock/model.ts` declares the five node classes, `MockClaim`, `MockEdge`
and `classOf()`; `frontend/src/core/data/source.ts` declares `WireClaim` for the JSON a
query returns, frames the RFC 7464 sequence itself in `claimsFromStream`, and builds each
`POST /query` body from an object literal. None of it is generated, and nothing checks it
against a source.

Until now there was no alternative. `ranke-go` had no browser counterpart, so the explorer
stood in for the ADT as well as for the client. `openapi/openapi.gen.ts` has existed since
`add-rest-api` and nothing imports it; `frontend/tsconfig.json` includes only `src`, so the
file is not reachable from the bundle at all.

Both gaps have closed. `@flocko-motion/ranke` 0.2.0 publishes the ADT for TypeScript:
`Claim` and `Edge` carrying `id` as a string, `typeClass` and `typeSub` already split,
`createdAtMs` beside the RFC 3339 `createdAt`, `fields`, and a `ContentRef`. It reads a
framed sequence two ways, through `readClaims` over a `ReadableStream` or a `SeqReader`
push parser reporting `bytesRead`, and it carries `codec_json` for the JSON projection,
`claim_builder` for constructing claims, and `matchTypeList` for the class globs the query
contract specifies. Its `Query` comes from `rql.schema.json`, the schema the specification
releases.

So the duplication now has an owner, and keeping it costs what duplication costs. Comparing
the three copies of the query type showed what the cost looks like: `OutputContent` required
`overflow` in `rql.schema.json` and in the library, and left it optional in `openapi.yaml`,
so a client could send `content: {max: 1024}` with no `overflow` and the generated server
would accept a body the standard rejects. That divergence has since gone the right way —
`openapi.yaml` now `$ref`s `rql.schema.json` and holds no second copy of the language, with
`make pull-rql-schema` taking a newer release and `openapi.gen.yaml` bundling the reference
for generators that cannot follow it. The explorer is the copy that remains.

> **Note, 2026-08-06 — the divergence went the other way, twice.** The premise above did not
> survive contact: `openapi.yaml` holds no copy of the query type at all, `$ref`ing
> `rql.schema.json` for the whole read language, so there was nothing to make required (task
> 6.1). Then `R-QCONTENT` made `overflow` *optional* — absent is `omit` — and both libraries
> followed, leaving this repo's vendored schema the only copy still demanding it, because it
> was pulled from the latest ranke-graph **release** rather than from the spec. Fixed by
> vendoring the spec: `RQL_SCHEMA_URL` now points at `spec/rql.schema.json` on `main`, and
> `check-rql-schema` asks whether we match the spec rather than the last snapshot of it.
>
> The reasoning the proposal rests on holds — three hand-maintained copies of one type drift,
> and this is what drift looks like. Only the direction and the owner were the reverse of the
> guess.

## What Changes

- **The claim becomes the library's `Claim`.** `MockClaim`, `MockEdge`, `NODE_CLASSES`'
  class list and `classOf()` go. `typeClass` and `typeSub` arrive precomputed, which removes
  a string split per claim from the 300k path, and `createdAtMs` removes the `Date.parse`
  per record in `claimsFromSequence`.
- **The wire becomes the library's codec.** `claimsFromStream`, `claimsFromSequence` and
  `idsFromSequence` go in favour of `readClaims` or `SeqReader`, whose `bytesRead` is the
  progress signal `source.ts` currently derives from `content-length` by hand.
- **The query body becomes the library's `Query`.** Typed by the standard rather than by one
  binding of it, and serialised by `query_codec`.
- **The transport becomes the generated client.** `make generate` emits a committed copy of
  `openapi.gen.ts` into `frontend/src`, so `frontend/` stays buildable on its own. An `Api`
  is constructed per connection and receives the headers `authHeaders()` already builds. The
  hand-written `getJSON()` helper and every literal route path go.
- **The mock generator builds real claims** through `claim_builder`, so generated data and
  server data are one type read by one code path.
- **`OutputContent.overflow`** — see the note above: nothing to make required, and the rule
  has since gone the other way. What changed instead is where the schema is vendored from.
- **The stale claims about wiring go.** `frontend/README.md:44`, `NotWiredError`
  (`source.ts:70`) and `RestSource`'s docstring (`source.ts:178`) all say reading claim
  bodies from a connection is unwired, and `RestSource.fetch` has been doing it since
  `add-branch-selection`. `README.md:421-427` lists `src/mock/graph.ts` and
  `src/explorer/main.ts`, neither of which exists.

## Capabilities

### Modified Capabilities

- `graph-explorer`: gains the dependency boundary. Which types the explorer may declare and
  which it must take from a published library is a rule about the client's shape, and the
  hand-written model this change deletes is what its absence produced.

## Impact

- **`frontend/package.json`**: `@flocko-motion/ranke` as a dependency.
- **`frontend/src/core/mock/model.ts`**: the ADT half goes; the generator's invented
  subtypes (`email`, `person`, `knows`) stay, being open vocabulary rather than ADT.
- **`frontend/src/core/data/source.ts`**: the largest change. `WireClaim`, the three
  sequence readers, `getJSON()` and the literal paths go; `RestSource` drives the generated
  `Api` and the library's codec.
- **`frontend/src/core/graph/build.ts`**: `classOf()` becomes `typeClass`, and the
  `startsWith` edge filter at line 184 becomes `matchTypeList`.
- **`frontend/src/core/mock/generate.ts`**: builds claims through `claim_builder`.
- **`frontend/tsconfig.json`, `Makefile`, root `Makefile`**: the generated copy into
  `frontend/src`.
- **`openapi/rql.schema.json`, root `Makefile`**: vendored from the spec on `main` rather
  than from the latest release (see the note above), and `openapi.yaml`'s account of the
  canonical form now states `content: {max: 0}`, which `R-QCANON` requires and `R-QCONTENT`
  no longer supplies by default.
- **Not affected**: the server's behaviour. Every change here is to what the contract
  *states*; the endpoint enforces the library's rules either way.
