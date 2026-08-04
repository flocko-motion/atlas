## Context

The trigger was a missing feature: no route lists an archive's branches, so a client
must be told a branch name out of band before it can read anything. `OpBranchList`
already exists in core and is orphaned at the transport.

Placing that route surfaced the structural problem. `/{branch}/…` occupies the root, so
the first path segment is a wildcard. Everything else has been arranged around that.

## Goals / Non-Goals

**Goals:**

- A root namespace that is not a wildcard, so new resources need no defensive prefix.
- A URL layout a reader can predict without knowing the library.
- Branch discovery.

**Non-Goals:**

- **Anything in `internal/core/`.** Every operation, right and payload exists.
- **RankeQL.** `$universe` / `$archive` stay as scope values in a query body; that is
  the library's vocabulary and the body is its type, transcribed field for field.
- **The grants language.** `$`-targets stay in `core-access`. Policy identifiers and URL
  paths are different namespaces that happen to correspond.

## Decisions

### Branches become a collection, which is what frees the root

```
GET /branches                                   list
GET /branches/{branch}/head
GET /branches/{branch}/claims/{id}
GET /branches/{branch}/claims/{id}/content
```

With `{branch}` in the second segment, no branch name can collide with a fixed route,
whatever it is called. The contract's whole "reserved top-level segments" requirement —
the rule that pushed operational routes under `/system/` because *"a root-level `/x/{id}`
and `/{branch}/head` match the same path with neither more specific"* — stops being a
constraint the design has to satisfy and becomes a property it has.

This is the substance of the change. The sentinel and the listing both follow from it.

### Every claim read is a scope followed by a claim

```
GET /branches/{branch}/claims/{id}
GET /universe/claims/{id}                (was /$universe/claim/{id})
GET /universe/claims/{id}/content
```

`$universe` was doing one job in the path: telling "a claim, anywhere" apart from "a
claim in the branch called universe". Once branch reads live under `/branches/`, that
ambiguity is gone — so the sentinel could simply be dropped, leaving `GET /claims/{id}`
to mean archive-wide by the *absence* of a branch.

That was the first shape here and it is worse, for a reason that only shows up on the
third scope. The access model reserves **three**: `$universe`, `$archive` and
`$branches`, and RankeQL carries `BranchUniverse` and `BranchArchive` as distinct scopes
— the Universe being everything the store holds, the archive being the closure of the
current head. Spending the bare `/claims/{id}` on one of them leaves the other with
nowhere to live, and an inference ("no scope means universe") that quietly becomes wrong
the day a second scopeless meaning is wanted.

Naming the scope keeps every read the same shape — `<scope>/claims/{id}` — and gives
`$archive` a route it has never had. It also drops the `$` without dropping what the `$`
was attached to: `universe` is a plain segment in a position where a scope is expected,
so nothing needs to mark it as "not a branch".

### The `$` names are documented, because a client already meets them

The first instinct was to keep `$archive` and `$universe` out of the documentation
entirely — plain routes, no mention of the reserved names. That hides nothing useful.
An operator writes `R $universe` in a grant; a client puts `select.branch: "$archive"`
in a RankeQL body. The names are already in the vocabulary, so omitting them from the
route documentation only leaves `GET /archive/claims/{id}` looking arbitrarily related
to the grant that gates it.

So the contract says what each collection reads: `/archive/…` is the `$archive` scope,
`/universe/…` is `$universe`. One concept across three surfaces — route, query language,
access policy — instead of three that a reader has to notice correspond.

What that must not become is a second path. Documenting the identity is explanation;
offering `/branches/$archive/claims/{id}` as well would be two URLs for one resource.
`{branch}` means an ordinary branch, so a reserved name there is a branch that does not
exist, and not-found is the honest answer — consistent with the contract already making
unknown and out-of-scope indistinguishable.

The grants are unchanged: `/archive/…` needs **R** on `$archive`, `/universe/…` needs
**R** on `$universe`. `matchBranch` already refuses to let an ordinary glob reach either
— a `$`-prefixed target requires an exact grant, so `R *` confers neither.

### `$` was never a URL convention

It is a device of the grants language: illegal in branch names, therefore unmatched by
any glob, therefore safe as a reserved target. That reasoning is entirely internal, and a
REST client should not need it to read a URL.

It also breaks in the place these routes are meant to be used. `$` is legal unescaped in
a path, but a shell expands it:

```
curl http://host/$universe/claim/abc     # bash sends http://host/claim/abc
```

The GET subset exists to be *"reachable from a browser or `curl` without JSON"*. A path
that silently becomes a different path when typed is not that.

### `/system/` stays, for a different reason than it was created for

It was created to dodge ambiguity with `/{branch}/…`. That reason is gone. The grouping
survives on its own merit: `rest-api` marks these as *"ranke-db extensions beyond the
paper"* — layers, verification runs — and separating operational surface from archive
content is worth a segment. `/system/verification` becomes `/system/verifications`, since
it is a collection.

`POST /system/verifications/{id}/cancel` keeps its verb. Cancellation is a state
transition with no natural noun, the form is widely used, and inventing
`/cancellation` to satisfy a rule would trade a familiar path for a puzzling one.

### Plural members

`claim/{id}` → `claims/{id}`, so every collection reads the same way. Small, but the
inconsistency would be permanent and visible in every generated client method name.

## Risks / Trade-offs

- **Operation ids change**, so generated client method names change with them
  (`getUniverseClaim` → `getClaim`). That is the intended readability gain surfacing in
  the client; it is still a rename to absorb. The API is in design with no released
  consumer, so this is bookkeeping rather than migration — `frontend/` touches one call
  site (`core/data/source.ts:101`).
- **`/branches` reads like a branch named "branches"** to someone skimming. It resolves
  correctly — that branch is `/branches/branches/head` — but the collection name and a
  possible member name coincide, which is inherent to REST collections and worth a test.
