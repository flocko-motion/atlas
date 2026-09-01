#import "vocabulary.typ": *

= Accounts and access <ch:access>

Who may read and who may contribute is a property of this server, not of the
archive it serves. It belongs in the configuration, and it is never written
into the graph as a #gls("claim"): copy an archive to a second deployment and
none of the policy below travels with it, which is the point. The graph holds
what happened; the file holds who may act here.

== The roster <sec:roster>

`accounts` maps an account name to the grants it holds.

#example[
#listing[
```json
"accounts": {
  "reader":      {"grants": ["R *", "R $branches", "R $archive"]},
  "contributor": {"grants": ["CR proj-*", "R $branches"]},
  "ops":         {"grants": ["CR *", "CR $branches", "R $archive", "R $universe"]}
}
```
]
]

An account is a name and a list of rights, and it carries no credential of its
own. What proves a request is that account belongs to an endpoint's `auth`
section (@sec:auth), and which endpoints can reach it at all is `admit`
(@sec:admit).

== Grants <sec:grants>

A grant is one string, `"RIGHTS TARGET"`: the rights as letters run together,
then what they apply to. The rights are CRUD.

#item("C")[
  Contribute #glspl("claim") to a #gls("branch"), creating the branch if it is
  new.
]

#item("R")[
  Read it.
]

#item("U")[
  Overlay an existing claim with a newer version.
]

#item("D")[
  Delete claims, which is physical removal rather than a mutation.
]

`"CR proj-alpha"` is one grant conferring two rights. Rights are additive
across an account's grants: a request is allowed when any one grant carries the
right and matches the target.

The target is either a branch pattern — lowercase letters, digits and `-`, with
`*` and `?` as wildcards — or one of four reserved names.

#item("$branches")[
  The #gls("branch-table") itself. `R` lists the branches; `C` is what creates
  one, since the table is a claim and adding a branch contributes to it.
]

#item("$archive")[
  The #gls("ranke-archive") as a whole — its head and shape.
]

#item("$universe")[
  The unconfined read across everything stored, whatever branch it belongs to.
  Only `R` applies here; any other letter is refused.
]

#item("$sequencer")[
  Reserved for the #gls("sequencer"). A grant may name it and no operation asks
  for it yet.
]

#warning[A wildcard never reaches a reserved name. `"CR *"` covers every
ordinary branch and no `$`-name, so each reserved target a deployment wants
must be granted literally — which is why the roster above lists them one by
one.]

Grants are checked when the file is parsed, so an unknown right letter, a
branch pattern with an illegal character, or `"C $universe"` fails
`verify --level syntax` rather than at the first request.

== What each operation requires <sec:operations>

#table(
  columns: (auto, auto),
  align: left,
  table.header([*The caller asks for*], [*The grant it needs*]),
  [Query claims], [`R` on the queried scope],
  [One claim, or its content, by id], [`R` on the scope it is read through],
  [A branch's head, or its head and height], [`R` on that branch],
  [The branch list], [`R $branches`],
  [The archive's head and shape], [`R $archive`],
  [Contribute claims to a branch], [`C` on that branch],
  [Create a branch], [`C $branches`],
  [Purge claims], [`D` on every branch holding them],
  [Health, storage layers, verification runs], [none],
)

#note[Verification needs no grant of its own: it reports findings rather than
contents, so a run over a #gls("closure") tells a caller nothing the closure
would not.]

`U` and `D` are accepted in a grant, and no route exercises them yet: nothing
served overlays or purges a claim, so an account holding them can do what a
`CR` account can and nothing more.

== Attenuation <sec:attenuation>

A macaroon carries first-party caveats, and each is read as a grant of the
opposite polarity: the account's own grants say what it may ever do, the
caveats narrow what this particular token may do, and a request must satisfy
both. Caveats accumulate — every one of them must allow the request, so passing
a token on after adding a caveat can only ever hand over less.

Nothing else moves rights at run time. There is no call that edits the roster,
and no claim in the graph that confers one: to change who may do what, stop the
instance, edit the file, and start it again.
