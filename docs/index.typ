// docs/index.typ — the root of ranke-db's own docs tree.
//
// Authored and committed, like the chapters it includes. `vocabulary.typ` and
// `handbook.typ` are placed by the build and gitignored — `make docs-papers`
// (via ranke-graph's fetch-ranke-docs.sh) puts the print backend there.
//
// Build:  make docs-pdf   (dist/docs.pdf, through shared/handbook.typ)

#import "handbook.typ": *
#import "vocabulary.typ": *

#show: handbook.with(
  title: "The RankeDB Operator's Manual",
  subtitle: [Configuring and running an instance: the launch artifact
             field by field, and the access policy it carries],
  date: "2026-08-31",
)

RankeDB serves one #gls("ranke-archive") from one configuration file. The file
is the whole of the deployment: it names where #glspl("claim") are kept, what
advances the head, the identity merges are signed under, where secrets come
from, which addresses are served, and who may act on what. Nothing else
configures an instance, and nothing reconfigures one while it runs.

This manual is that file, field by field. It says what each section selects,
which backends a section may name and what fields each of those takes, and how
the account roster decides what a request is allowed to do. What a claim is,
how a #gls("closure") is verified, and the query language a read is written in
belong to the Ranke-Graph specification and the `ranke-go` library that
implements it; the REST surface itself is fixed by the OpenAPI contract,
`openapi/openapi.gen.yaml`, which ships in this manual's own download.

#part[Part I — Launching an instance]

#include "01-running.typ"

#part[Part II — The launch artifact]

#include "02-adapters.typ"

#include "03-endpoints.typ"

#include "04-access.typ"
