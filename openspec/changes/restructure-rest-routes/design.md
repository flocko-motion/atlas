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

### The archive-wide read loses its sentinel because it no longer needs one

```
GET /claims/{id}                 (was /$universe/claim/{id})
GET /claims/{id}/content
```

`$universe` was doing one job in the path: distinguishing "a claim, anywhere in the
archive" from "a claim in the branch called universe". Once branch reads live under
`/branches/`, that ambiguity is gone and the distinction the sentinel encoded is carried
by structure instead — a claim read **scoped by a branch** is under that branch; a claim
read with **no branch** is archive-wide. Which is exactly what the privilege means:
`core-access` calls it *"reading a graph by head id directly — bypassing the branch
table"*. Bypassing the branch table now looks like bypassing `/branches`.

The grant is unchanged: `GET /claims/{id}` requires **R** on `$universe`.

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
