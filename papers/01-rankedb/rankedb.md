---
title: "RankeDB: A Provenance-First Foundation for Knowledge Systems"
author: Florian Noël
date: 2026-04-15
status: draft
license: CC-BY-4.0
---

# RankeDB: A Provenance-First Foundation for Knowledge Systems

## Abstract

We present RankeDB, a database architecture for knowledge systems in which provenance is not metadata but the foundational data structure.
RankeDB inverts the conventional relationship between knowledge graphs and provenance: rather than attaching provenance information to an existing knowledge graph, the Provenance DAG *is* the primary representation, and the Semantic Graph is a materialized view projected from it.
The architecture comprises three storage levels — Sources (an immutable content-addressable object store for raw artifacts), Cognition (an append-only directed acyclic graph for derivation history), and Semantics (a graph index optimized for associative retrieval) — unified behind a single API.
All data, including metadata, classifications, and provenance itself, is treated as knowledge: queryable, derivable, and subject to the same immutability guarantees.
We argue that this architecture is uniquely suited to the emerging regime of large-context-window language models, where accumulation of full inferential history provides a strategic advantage over systems that destructively consolidate.

The system is delivered as two components: **RankeDB Server**, a self-contained server (Docker Compose stack) exposing the complete data model through a REST API, and **RankeDB Explorer**, a bundled visual interface for navigating the Provenance DAG and the Semantic Graph.
Together they constitute a complete, deployable provenance database with an integrated visualization tool.

> **Note on structure.** The paper is organized in two parts.
> **Part I — The Intuition** (§1) argues why a provenance-first foundation must exist, from the archival tradition through the CS priority to the machine-reading/writing rupture.
> **Part II — The Foundation** (§2–§5) describes what was built on that intuition: three levels, one graph, one API, and the properties that follow from them.
> Part II closes by pointing forward to follow up papers that will continue that investigation by presenting the **first generation of application** on the foundation — the test of whether the philosophy-derived architecture bears load.
> The paper is a falsifiable bet: if the assumptions hold, later generations of workers and applications will continue to build on the same base; if they do not, the foundation was misjudged.

---

## Part I — The Intuition

Part I builds the case for why a provenance-first foundation must exist.
The argument runs in three movements: a 180-year archival tradition that already understood knowledge as inseparable from its chain of attribution (§1.1); a computer-science priority that was identified but never operationalized as a knowledge-graph substrate (§1.2); and an acute rupture in the machine-learning era that makes the old oversight untenable (§1.3).
The three converge on a single conclusion (§1.4): the foundation described in Part II is the minimum response to what has been lost.

## 1. The Problem: Knowledge Without Provenance

> **TODO — Write §1 opening (1–2 paragraphs).** State the thesis of Part I: traditional knowledge graphs optimize for *current truth* and treat provenance as metadata; this was defensible in an era of expensive storage and limited query capacity; it is untenable in a regime where knowledge is read and written by machines at scale.
> Introduce the three movements (archival tradition → CS priority → LLM rupture) without yet arguing them.
> Source material: `quotes.md`, Talisman, Ranke.
> Close with a one-line preview of §1.4's resolution.

### 1.1 The Archival Tradition

> **TODO — Scaffold §1.1 from `quotes.md`.** The archival profession already understood — for 180 years — that knowledge stripped of its derivation chain decays into hearsay.
> RankeDB is not inventing this insight; it is operationalizing it in a regime the archivists did not live to see.
> Intended arc of the subsection:
>
> - **Ranke** (1795–1886): every claim traces to a critically examined primary source; the discipline of attribution as the foundation of historiography.
> - **Cencetti / *respect des fonds*** (1841): the archival principle that records must be kept in the order and context of their origin — provenance as the organizing principle of memory itself.
> - **Briet** (1951, *Qu'est-ce que la documentation?*): documentation as evidence; the object is not the thing but the trace it leaves.
> - **Wilson** (1968, *Two Kinds of Power*): the bibliographic control problem — the difference between having information and being able to trust it.
> - **Burke** (*A Social History of Knowledge*): knowledge as a historically contingent product of institutions that *ratify* claims through attribution chains.
> - **CLIR / digital preservation**: the modern reframing of provenance as the precondition for long-term trust in digital evidence.
>
> Close with: the archivists spent two centuries working out what it means to preserve the chain of attribution.
> None of them had to contend with a generation of machines that could write faster than the chain could be maintained — but they left the discipline in place for those who would.

RankeDB (pronounced *run-keh-dee-bee*) is named after Leopold von Ranke (1795–1886), the historian who transformed his discipline by insisting that every historical claim must trace back to a critically examined primary source.
Ranke's famous phrase — history *"wie es eigentlich gewesen"*, "as it actually was" — has since been rightly criticized for assuming unmediated access to past reality.
RankeDB takes that criticism as foundational: the primary data point is never *"how it was"* but the artifact of a communicative act that reports, claims, or interprets it — an email, a chat message, a voicemail, a document.
What RankeDB stores is always someone's utterance about the world, never the world itself.
What survives from Ranke's method, intact, is the discipline of attribution: nothing is asserted without its derivation, and nothing is derived without its sources.

### 1.2 The CS Priority That Was Never Operationalized

> **TODO — Scaffold §1.2.** Computer science identified provenance as a first-class concern — and built robust machinery for it — but never integrated it as the substrate of a knowledge graph.
> The building blocks are mature; the architectural composition is the gap.
> Intended arc:
>
> - **Cheney (2009).** Provenance as a first-class concern for scientific workflows and database systems. The tooling was built; the KG integration was not.
> - **Pérez, Rubio & Sáenz-Adán (2018, *Knowledge and Information Systems*).** Systematic review of 105 provenance systems; six-dimensional taxonomy (general aspects, data capture, data access, subject, storage, non-functional). Evidence that the components exist — the integration with knowledge representation is what is missing. Cited via [talisman2026](sources/talisman2026provenance). **Priority: H.**
> - **Sikos & Seneviratne (2020). Data Science and Engineering.** *RDF "inherently lacks the mechanism to attach provenance data."* Named graphs, reification, RDF-star, singleton properties, nanopubs — each a workaround, none a substrate. `read_2.pdf`. **Priority: H.**
> - **Takan (2023, PeerJ).** *"Although the issue of immutability in data structures has been frequently studied, there is no research on immutability in knowledge graphs."* `read_2.pdf`. **Priority: H.**
> - **Dibowski (2024, FOIS, Bosch Research).** *"A problem that has not yet adequately been solved for KGs is the traceability and provenance of changes… KGs typically contain the current snapshot of data valid at a certain moment in time only."* `read_2.pdf`. **Priority: M.**
> - **Figay (2025). "When Knowledge Graphs Fail, It's Not the Ontology — It's the Epistemology"** (Medium). Enterprise KGs fail because teams conflate data / information / facts / inferences / unknowns — precisely the conflation RankeDB's three levels separate. `read.pdf`. **Priority: M.**
> - **PDF2's five-angle framing:** the gap has been identified independently from (1) KG engineering, (2) LLM/AI provenance, (3) scientific reproducibility, (4) enterprise AI governance, (5) content addressability for AI. Five fields arrived at the same unmade proposal.
>
> Close with the "groundwork was ready but never assembled" line: the parts have been on the shelf for two decades.
> What has been missing is a design that puts them in the right order.

Existing systems that address this tension do so partially.
Temporal knowledge graphs (Graphiti/Zep, [Rasmussen 2025](sources.gen.md#rasmussen2025graphiti)) preserve *when* facts were valid but perform destructive entity summary updates, losing derivation history.
Versioned knowledge bases ([TerminusDB, Mendel-Gleason et al.](sources.gen.md#terminusdb)) track *what* changed across snapshots but not *why* or *how* knowledge was derived.
Immutable databases ([Datomic, Hickey 2012](sources.gen.md#hickey2012datomic); [Fluree](sources.gen.md#fluree)) preserve all historical states but lack a semantic knowledge layer and do not model derivation chains.
No existing system treats the full chain of provenance — from raw source artifact through extraction, normalization, and synthesis — as first-class, queryable knowledge.

### 1.3 The Rupture: Machines Reading and Writing at Scale

> **TODO — Scaffold §1.3 from `quotes.md` and Talisman.** The old oversight — provenance treated as annotation — was tolerable when knowledge was written by humans at human speed.
> It collapses when knowledge is read and written by machines at scale.
> Intended arc:
>
> - **Talisman (Feb 2026). "Where Provenance Ends, Knowledge Decays."** Substack. Traces provenance from 1841 *respect des fonds* through Semantic Web to LLMs. Key quote: *"LLMs strip provenance from knowledge — systematically, architecturally, and by design."* RAG addresses retrieval-level provenance while *"leaving the deeper layer entirely unattributed."* Closest existing articulation of RankeDB's motivation; proposes no technical design. `read_2.pdf`. **Priority: H — lift framing.**
> - **Vibe citing.** The phenomenon of plausible-looking citations generated without a verifiable chain — the visible symptom of a substrate that does not demand attribution.
> - **Knowledge network decay / doom loop.** Models trained on the outputs of earlier models, with provenance severed at every generation; the cumulative effect on the integrity of the knowledge commons.
> - **Berners-Lee as foil.** The Semantic Web's promise was machine-readable knowledge; its unmade promise was machine-traceable provenance. What arrived in the LLM era was the opposite: massive scale, zero attribution.
>
> Close with: the severity of the rupture is what changes the calculus.
> Before: provenance-as-substrate would have been a nice-to-have.
> After: it is the minimum response.

Knowledge management systems face a fundamental tension: they must serve both *current* truth and *historical* understanding.
Traditional knowledge graphs optimize for the former — they store what is believed to be true now, updated in place as understanding changes.
This design made sense in an era of expensive storage and limited query capacity.
It makes less sense in a regime where the ability to present a model with the full derivation history of a belief — including contradictions, revisions, and competing interpretations — may support qualitatively better reasoning than presenting a single consolidated snapshot.

For a rich treatment of what *provenance* has meant across 180 years — from the archival principle of *respect des fonds* through the Semantic Web to the LLM era — we refer the reader to Talisman's essay ([Talisman 2026](sources.gen.md#talisman2026provenance)).
In this paper we use the term in a narrower, operational sense: the complete derivation chain of a piece of knowledge — the raw source artifact, every intermediate processing step, every tool and configuration involved, and every transformation applied.
This is compatible with W3C PROV-DM's Entity/Activity/Agent vocabulary ([Moreau & Missier 2013](sources.gen.md#moreau2013provdm)) but makes a stronger commitment: in RankeDB, provenance is not metadata about knowledge — it *is* knowledge, stored in the same graph, queryable through the same API, subject to the same invariants.
Each node in the graph is both a statement and the record of how that statement came to be.
There is no separate "provenance layer" — the derivation chain is the knowledge, and the knowledge is the derivation chain.

### 1.4 Convergence: A Foundation, Not a Feature

> **TODO — Write §1.4 closing.** The three movements converge.
> The archival tradition had the insight; the CS literature has the components; the machine-reading/writing era makes the gap urgent.
> A provenance-first foundation is not a refinement — it is the shape that emerges when the three are taken seriously at once.
> Part II describes the foundation; the follow-up papers present the first generation of application that tests whether the philosophy-derived foundation actually bears load.
> The paper is therefore a falsifiable bet: if the assumptions hold, later generations of workers and applications will keep building on the same base; if they do not, the foundation was misjudged.
> Either way, what this paper owes is the argument for why *this* shape is the right one to try.

RankeDB addresses this gap through a structural inversion: the provenance DAG is the system, and everything else — including the semantic knowledge graph — is a view derived from it.
It is deliberately **under-prescribed** in how it should be used: the data model preserves every level of detail in parallel — from the raw source artifact up to the semantic triplet — networked by provenance, and leaves the strategy of retrieval and reasoning to the consumer.
The follow-up papers are the first generation of application on that foundation; Part II describes what they are built against.

---

## Part II — The Foundation

Part II describes what was built from the intuition of Part I: a single graph with three levels, one API, and a small set of invariants that encode the provenance-first commitment.
The architecture is presented first (§2), then the core properties that follow from it (§3), then the reference implementation that enforces it (§4), and finally the worker model through which the graph is populated (§5).
Part II closes with a brief forward pointer to the follow-up papers as the first generation of application on this foundation.

## 2. Architecture

RankeDB is a single connected graph organized into three levels (see Figure 1).
Each level, viewed in isolation, has its own characteristic shape:

- **Level 0 — Sources.** A forest: every source has at most one parent (a format conversion, a normalization, an item extracted from a bulk container), and each originally ingested artifact is a root.
- **Level 1 — Cognition.** Nodes in Level 1 may combine multiple inputs from Levels 0 and 1.
  Together with Level 0, Level 1 forms the **Provenance DAG**: the strictly acyclic backbone of the system.
- **Level 2 — Semantics.** A **Semantic Graph**: a property graph of entities and relations with cyclic semantic connections, optimized for associative retrieval. Each relation in Level 2 is described by a relationship node with a *head* edge and a *tail* edge pointing to an entity node. Every node in Level 2 is pointed at from Level 1 nodes, documenting the provenance of each atomic piece of knowledge.
  Level 2 is populated by projection workers reading Level 1, not a deterministic function of it (§2.1.3).

The level names (Sources, Cognition, Semantics) classify their content; the functional terms (the Provenance DAG at L0+L1, the Semantic Graph at L2) describe their role in the system.
Writes flow in one direction across levels: a node at any level may reference nodes at the same or an earlier level, never later.
This keeps the Provenance DAG acyclic even though the Semantic Graph may contain semantic cycles.

![Figure 1: The three storage levels of RankeDB.](drawio/layers.svg "Figure 1: The three storage levels of RankeDB. Node types such as Email, Conversation, Fact, and Summary are application-defined examples; RankeDB provides type categories (e.g. source, conversation) but leaves concrete types to the application.")

*Figure 1: The three storage levels of RankeDB.
Concrete node labels such as Email, Conversation, Fact, and Summary are illustrative examples.
RankeDB defines content type categories (e.g. `source/conversation`, `classification/entity`) and leaves encodings and application-specific types to the application layer.*

The graph is strictly append-only.
Once written, a node or edge is never modified or deleted — beliefs later found to be false remain in the graph as knowledge of what was once held true, annotated by new nodes that record their falsification.
Removal is possible only as an administrative operation, treated as a fork of the original graph (as in functional programming over immutable data structures).
§3.2 develops the consequences.

> **TODO — Reading for §2 (three-layer architecture precedents):**
>
> - **Enterprise Knowledge consultancy (2024-2025). "Graph Analytics in the Semantic Layer: An Architectural Framework for Knowledge Intelligence."** Documents a "three-graph architecture": metadata graphs (lineage, ownership) / knowledge graphs (ontology-backed entities) / analytics graphs (pattern detection). *Key distinction: operates three graph types in parallel, whereas RankeDB arranges three layers sequentially.* `read.pdf`. **Priority: M.**
> - **IntuitionLabs (2025) biotech/pharma KG pattern.** Data lake → semantic integration (graph DB) → service layer. Closer to RankeDB's sequential flow. `read.pdf`. **Priority: L.**
> - **Ant Group OpenSPG/KGFabric (VLDB 2024).** Industrial-scale integration of property graph performance with semantic constraints; **98% storage reduction vs Neo4j** via hybrid compression. `read.pdf`. **Priority: L — cite for scale validation.**
> - **SPADE (SRI International).** Provenance auditing system storing derivation chains in Neo4j OR Postgres, abstracting over both through its QuickGrail query language (ACM Queue 3476885). *The closest direct analog to RankeDB's split-store architecture (FalkorDB for semantics, Postgres for provenance).* `read.pdf`. **Priority: H — must cite in §6, currently missing.**
> - **dbt Semantic Layer, Cube.dev, AtScale.** Analytics semantic layer tradition — abstraction over warehouse data into business metrics. January 2026 **Open Semantic Interchange (OSI)** spec supported by 40+ companies (Snowflake, Salesforce, Databricks). Shares DNA with transformation lineage but at dataset level, not per-fact. `read.pdf`. **Priority: L.**

### 2.1 Nodes

All nodes in the graph share a common format.
Level-specific extensions are introduced in §2.1.1, §2.1.2, and §2.1.3; there is no overlap between level-specific fields.

Content and identity are separated: a node carries its payload in `content` together with `content_sha256` and `content_len` for integrity and size, while `id` is the node's identity in the graph.
For L0 root artifacts, `id` is deterministic from `content_sha256` — this is what makes ingestion idempotent: re-uploading the same bytes maps to the same root node.
For all other nodes (derived L0 nodes, L1 derivations, L2 projections), `id` is synthesized independently, because two nodes with identical content but different provenance are distinct knowledge.

| Field            | Purpose                                                                                          |
| ---------------- | ------------------------------------------------------------------------------------------------ |
| `id`             | Node identity (deterministic from `content_sha256` for L0 root artifacts; synthesized otherwise) |
| `content`        | Payload (text or bytes, interpreted per `encoding`)                                              |
| `content_sha256` | Cryptographic hash of `content`                                                                  |
| `content_len`    | Byte length of `content`                                                                         |
| `content_type`   | Category and type (dispatch key for workers and consumers)                                       |
| `encoding`       | MIME-style `class/format` (e.g. `text/eml`, `image/png`); dispatch key for workers               |
| `created_at`     | When the node entered the graph                                                                  |

The `content_type` field follows a two-part pattern: `category/type`.
RankeDB defines the categories and a set of foundational types; applications may extend the types within each category.

The `encoding` field follows a MIME-style pattern: `class/format`.
The class is hardcoded and small — `text`, `image`, `audio`, `video`, `application` — and doubles as a machine-readable policy hint (only `text/*` is treated as text; everything else is binary).
The format is the specific syntax (e.g. `text/eml`, `text/whatsapp`, `image/png`, `application/pdf`) and is the primary dispatch key for reactive workers.
Formats are application-extensible; each format is a micro-project: a parser, quickly written, easily tested.
Nodes can wait patiently until a parser for their format becomes available.

#### 2.1.1 Level 0: Sources

Level 0 is the region of the graph that holds ingested external artifacts.
An artifact may be a communicative act (an email, a chat transcript, a letter, a voicemail), a document (a book, a contract, an article), a perceptual capture (a photograph, a recording), a machine observation (a sensor reading, a transaction log), or structured data (a spreadsheet, a database export).
Every artifact enters the graph as a node, self-describing through attached metadata and its own content.
A source node may be an original artifact or a source derived deterministically from another source - e.g. unpacked from a bundle (e.g. an individual conversation extracted from a bulk chat export), converted from another format (e.g. TIFF to PNG), or cleaned and normalized (e.g. stripped of html tags).
All of these remain sources: captured artifacts of the world, not knowledge *about* them.

Level 0 is the archive. It does not claim truth about the world — it just stores the artifacts ingested. That's our ground truth, the fixpoint against which every derivation can be traced.

In addition to the common fields, L0 nodes carry:

| Field                 | Purpose                                         |
| --------------------- | ----------------------------------------------- |
| `artifact_created_at` | Original creation date of the external artifact |
| `origin`              | Ingest pathway                                  |
| `original_name`       | Original filename                               |

**Content types:**

Level 0 holds only `source/*` content types. RankeDB defines four source types and one container type.
The design principle is *few types, many encodings*: the diversity of the world lives in encodings, not in the type system.
Format-specific diversity within a content type (e.g. `text/eml`, `text/whatsapp`, `text/telegram` for conversations) is reconciled by normalization workers that produce new Level 0 nodes with the same content type but a canonical encoding (e.g. `text/plain`).
Normalization changes only the format, not the kind of thing — the node is still a source artifact.
By the time Level 1 workers pick up a conversation for cognitive processing, source-format diversity is irrelevant — workers will likely process only normalized content (though they have full access to the graph and can process whatever they need).

| Content type | What it captures | Examples |
|---|---|---|
| `source/conversation` | Communicative act with sender and receiver, even if implicit. An invoice is a conversation (sender → receiver). An article is a conversation (author → readers). What *kind* of conversation it is — invoice, contract, smalltalk — is determined by a classification worker in Level 1, not at import. | email, chat, letter, voicemail transcript, article |
| `source/media` | Audio, visual, or audiovisual capture. Content is opaque until a worker processes it — could be a voicemail, art, a surveillance recording, or a meeting. | photo, video, audio recording, screen capture |
| `source/record` | Objective, machine-generated observation of world-state. Not human expression — structured readings from sensors, APIs, instruments. | GPS positions, weather readings, stock prices, bank transactions |
| `source/data` | Structured information that does not fit the above categories. Defined by exclusion: not a communicative act, not a perceptual capture, not a machine observation. Application layer decides boundary cases. | spreadsheets, configuration files, database exports |
| `source/bulk` | Container of other sources. Unpacked by workers into individual source nodes. The bulk node serves deduplication across repeated exports — if a contained source already exists (same hash), it is skipped. | ChatGPT export, WhatsApp backup, Gmail archive, photo library export |

**Invariants:**

- Root artifact identity is deterministic from content. Re-ingesting the same bytes yields the same node.
- Source nodes are self-describing. Metadata is sufficient for full reconstruction (modulo worker non-determinism).
- Writes are idempotent. A duplicate `PUT` has no effect.

#### 2.1.2 Level 1: Cognition

Level 1 adds derived knowledge to the Provenance DAG.
Its nodes are derived from source nodes or other derived nodes through processing by external tools (workers).
Every derived node requires at least one input and one tool attribution.
The DAG is strictly acyclic because derivations cannot be circular: a node cannot be derived from its own output.

We call that process *cognition* as a metaphor for signal processing in the human brain, spanning low-level operations (edge detection in the visual cortex, phoneme recognition) up to abstract reasoning.
The term does not imply consciousness, awareness, or intent — we use it as shorthand for *information processing that extracts knowledge from lower levels of the graph*.

Together with Level 0, Level 1 forms the complete Provenance DAG.
Where Level 0 provides the roots — the sources — Level 1 stores the derivation history: not just what is believed, but *how it came to be believed*.
This history is itself knowledge: queryable, traversable, and available as context for downstream consumers.

L1 nodes use only the common fields (§2.1); there are no L1-specific node fields.
A derivation produced by a 2024 language model and one produced by a 2028 model from the same source are both preserved as competing interpretations, each with a full provenance chain that includes the tool node in effect at the time.

**Content types:**

Every node in Level 1 is the output of a worker interpreting, classifying, extracting, summarizing, or reasoning about the graph.
The content type categories distinguish *what kind* of derivation it produces.

Level 1 content types follow the same `category/type` pattern as Level 0.
The following categories are part of the RankeDB architecture; the types within each category are application-defined.
Examples given are illustrative only — RankeDB does not commit to any particular structure within a type.

| Category           | Purpose                                                                                                                                                                                                                                       | Examples                                                                                                                                              |
| ------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| `conversation/*`   | L1 representations of conversations, beyond raw source format.                                                                                                                                                                                | `conversation/transaction`, `conversation/interview` (never by source format)                                                                         |
| `image/*`          | L1 representations of images, beyond raw source format.                                                                                                                                                                                       | `image/photo`, `image/diagram` (never by source format)                                                                                               |
| `video/*`         | L1 representations of videos, beyond raw source format.                                                                                                                                                                                       | `video/meeting`, `video/lecture` (never by source format)                                                                                             |
| `classification/*` | A worker's statement about a node: what it is, who appears in it, what it concerns. Classification nodes bridge the Provenance DAG and the Semantic Graph — they live in Level 1 with full provenance and their edges project into Level 2.  | `classification/entity` (who/what was identified), `classification/content` (what kind of thing is this), `classification/topic` (what is this about) |
| `observation/*`    | A worker's statement about relationships between nodes — grouping, contradiction, correlation, sequence, gaps. The natural output of analytical workers that traverse the graph rather than processing individual nodes.                      | `observation/contradiction`, `observation/alias`, `observation/grouping`                                                                              |
| `summary/*`        | Condensed representation of one or more nodes.                                                                                                                                                                                                | by length, audience, or purpose                                                                                                                       |
| `fact/*`           | Extracted factual claim with provenance to the node that supports it.                                                                                                                                                                         | by domain, or by confidence threshold                                                                                                                 |

Source types and Level 1 types are not 1:1.
A `source/record` containing bank transactions can resolve into `conversation/transaction` nodes — sender, receiver, amount as message.
A `source/media` might resolve into `conversation/transaction` or into `image/diagram`.
The source type in Level 0 captures how an artifact entered the graph; the Level 1 type captures what it means.

Dependencies among Level 1 types are emergent from the workers and their usage of the infrastructure, not prescribed by the architecture: a `conversation` worker may wait for `classification/entity` results before it can resolve participants, producing a natural ordering without the architecture having to enforce one.

**Invariants:**

- The graph is acyclic within this level. No node can transitively depend on itself.
- Every node has provenance. No node exists without at least one input edge and one tool attribution.

#### 2.1.3 Level 2: Semantics

Level 2 holds the Semantics of the knowledge graph, structured as a Semantic Graph: a property graph optimized for associative traversal and retrieval.
It is a first-class level of the graph, populated by projection workers that read Level 1 and produce the entity and relation nodes that make associative retrieval possible.
If Level 2 were merely a deterministic view of Level 1, it would be redundant; it exists as its own level because associative traversal over the full Provenance DAG would be prohibitively expensive at scale, and because the cognitive work of deciding *what* to project is itself a worker activity.

Every node in Level 2 has a provenance edge back into Level 1.
Relation nodes additionally carry a `head` edge and a `tail` edge to the entity nodes they connect.
Semantic connections within Level 2 may be cyclic; the acyclicity of the Provenance DAG applies only to provenance edges (§2.2).

Relation labels are natural-language strings, not formal ontology predicates.
The ontology is not predefined — it emerges from the data as workers extract and normalize relations over time.

In addition to the common fields, L2 nodes carry:

| Field         | Purpose                                |
| ------------- | -------------------------------------- |
| `valid_from`  | Start of temporal validity window      |
| `valid_until` | End of temporal validity window        |
| `confidence`  | Confidence score                       |

**Content types:**

Level 2 defines two foundational categories — entities and relations — each with a fixed architectural shape.
Entity subtypes name a minimal upper-ontology that many applications will share; applications may extend with their own subtypes.

| Category     | Purpose                                                            | Foundational subtypes (applications may extend)                                                                                                                          |
| ------------ | ------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `entity/*`   | Projected nodes representing identifiable things in the world.     | `entity/person`, `entity/organization`, `entity/place`, `entity/thing`, `entity/work`, `entity/idea`, `entity/event`, `entity/role`                                      |
| `relation/*` | Reified semantic relations between entities, with head/tail edges. | `relation/alias` (two entities refer to the same thing), `relation/part_of` (structural composition), `relation/has_role` (entity holds a role, time-bounded)            |

The foundational subtypes are a *shared vocabulary*, not a privileged class.
RankeDB does not enforce or validate them at the storage layer — an application that ignores them and defines its own types is architecturally equivalent.
What the foundational set provides is a coordination point: consumers walking the graph (entity-resolution libraries, UI renderers, analytical tools) can assume that `entity/person` means a human individual, `relation/has_role` is time-bounded, and so on.
Apps adopting this vocabulary become interoperable with tools built against it; apps inventing their own are responsible for interoperability themselves.
Like every other L2 relation, aliases carry confidence and compete with other claims — consumers decide how to weight them, not the architecture.

**Invariants:**

- Every L2 node has a provenance edge to Level 1.
- Every relation node has exactly one `head` edge and one `tail` edge, each to an entity node.
- Relation labels are natural-language; no formal ontology is required.
- Provenance edges are not stored in Level 2 — they belong to the Provenance DAG.

### 2.2 Edges

Two classes of edges coexist in the graph.

**Provenance edges** connect every derived node to its inputs.
They form the strict DAG that spans Level 0 and Level 1, and they carry the provenance references that connect Level 2 back into Level 1.
Provenance edges are acyclic by construction: a node cannot transitively depend on itself.

Each provenance edge carries a `run_id` property identifying the worker run that produced it.
The `run_id` is not a node field; it lives on the edge.
This is what enables administrative operations such as purging a defective run — the edges produced by that run are removed, orphaned nodes follow.

**Semantic edges** live in Level 2 only and connect reified relation nodes to their head and tail entities.
They are permitted to form cycles, because the Semantic Graph is a graph of associations, not a graph of derivation.

Three further design decisions follow from treating the graph as a single data structure with two edge classes:

- **Parent relationships are edges.** An L0 node derived from another L0 node (a format conversion, a normalization, an item unpacked from a bulk container) is connected by a provenance edge, not a `parent` field. Denormalization may appear at the storage layer as an optimization; it is not part of the node format.
- **Tool and tool config are themselves nodes.** A worker's identity and configuration at a point in time are stored as L1 nodes (content type `tool/*`), which a derivation links to as an input. This gives tool configurations their own provenance and lineage, and puts "which tool produced this?" in the same graph as everything else.
- **Every node has provenance.** No node exists without at least one incoming provenance edge (except L0 root artifacts, which are roots by definition) and, where applicable, at least one input edge to a tool node.

## 3. Core Properties

### 3.1 Everything Is Knowledge

RankeDB makes no distinction between data, metadata, and provenance.
A classification ("this node belongs to the finance domain") is a node with provenance.
A visibility decision ("this node is accessible to group X") is a node with provenance.
An assessment of quality ("the 2028 extractor produces better results than the 2024 extractor") is a node with provenance.

This principle — *everything is knowledge* — eliminates the need for separate metadata systems, tagging taxonomies, or access control lists as external infrastructure.
All of these are expressible as nodes in the DAG, derived from the same sources, subject to the same immutability guarantees, and queryable through the same API.

> **TODO — Reading for §3.1 (epistemological tradition):**
>
> *This is the intellectual lineage §3.1 is currently missing.
> PDF1 traces a tradition from 1979 TMSes to 2026 AI memory papers that RankeDB sits squarely inside — no existing PKG has operationalized it.*
>
> - **Doyle, J. (1979). "A Truth Maintenance System."** *Artificial Intelligence.* JTMS — dependency network of beliefs and justifications; traces conclusions to premises; propagates revision through the network. **RankeDB's direct intellectual ancestor.** `read.pdf`. **Priority: H.**
> - **de Kleer, J. (1986). "An Assumption-Based TMS."** ATMS extends JTMS to maintain all alternative assumption sets simultaneously — conceptually parallel to RankeDB's add-only preservation of multiple states of belief. `read.pdf`. **Priority: H.**
> - **Alchourrón, Gärdenfors & Makinson (1985). AGM framework.** Formal postulates for rational belief change (expansion, revision, contraction). See SEP entry `plato.stanford.edu/entries/logic-belief-revision/`. `read.pdf`. **Priority: M.**
> - **"Graph-Native Cognitive Memory for AI Agents: Formal Belief Revision Semantics for Versioned Memory Architectures"** (2025-2026, arXiv 2603.17244). Applies AGM postulates to a Neo4j-based memory architecture for AI agents — proves graph memory operations can satisfy formal belief revision axioms. **Closest published work to RankeDB's epistemological framing**, targets AI agent memory rather than personal cognition. `read.pdf`. **Priority: H.**
> - **Carneades argumentation framework.** Varying proof standards per statement — direct analog to per-entity conviction levels. `read.pdf`. **Priority: L.**
> - **ASPIC+ framework.** Strict vs defeasible inference with three attack types (undermining premises, rebutting conclusions, undercutting rule applicability). `read.pdf`. **Priority: L.**
> - **AKReF (2025, arXiv 2506.00713).** Constructs argumentation knowledge graphs from text using ASPIC+. Heterogeneous graphs with argument nodes and attack/support edges. `read.pdf`. **Priority: L.**
> - *PDF1 observation:* the phrase *"thoughts as provenance"* — where synthesized knowledge generates semantic edges whose provenance the thought becomes — **appears genuinely unique**. No other system explicitly frames the act of thinking as evidence generation. Make this explicit in §3.1. **Priority: H.**

### 3.2 Immutability and Accumulation

RankeDB is strictly append-only at the knowledge level.
No node or edge is ever modified or deleted through normal operation.
When new information contradicts existing knowledge, the contradiction is represented as a new node — not as an update to the old one.
Both coexist in the graph, each with full provenance.

This design is a deliberate bet on the trajectory of language model context windows.
Systems that destructively consolidate today — merging entity summaries, deduplicating facts, compacting histories — optimize for current retrieval efficiency at the cost of inferential depth.
RankeDB optimizes for a future in which a model receiving the full derivation history of a belief (including contradictions and revisions) produces better reasoning than one receiving a consolidated summary.

Immutability applies to knowledge, not to infrastructure.
Technical defects (e.g., a buggy worker producing a million duplicate entries) are addressable through administrative purge operations that act on the storage level outside RankeDB's data model, identified by worker run IDs.
These operations are logged but are not part of the knowledge graph — they are infrastructure maintenance, not epistemological events.

> **TODO — Reading for §3.2 (immutability as foundational principle):**
>
> - **Helland, P. (2015). "Immutability Changes Everything."** *CIDR 2015.* Already in references. Key quotes to lift: *"accountants don't use erasers"* and *"the truth is the log; the database is a cache of a subset of the log."* **PDF2 explicitly notes: "No subsequent work has explicitly applied Helland's thesis to knowledge graphs, despite its enormous influence on event sourcing and distributed systems."** RankeDB would be the first. `read_2.pdf`. **Priority: H.**
> - **Nelson, T. (1960-present). Project Xanadu.** Specified immutable, add-only content space where documents are lists of pointers to regions in an "ever-growing" store; transclusion maintains *"visible provenance to the source"*; every connection bidirectional. **PDF2: "arguably the direct ancestor of what RankeDB proposes."** Cautionary lesson: Xanadu's refusal to compromise prevented adoption while the simpler Web prevailed. `read_2.pdf`. **Priority: H — currently missing from §6, must add.**
> - **Hickey, R. (2012). Datomic.** Already cited. PDF2 nuance to add: Datomic captures the *temporal* dimension of knowledge (what changed, when) but **not the *epistemic* dimension** (how knowledge was derived, from what evidence, by what process). `read_2.pdf`. **Priority: M.**
> - **Records in Contexts (RiC-O v1.1, May 2025, ICA).** International Council on Archives standard. Describes archival world as *"a graph of interconnected things"*; models `rico:ProvenanceRelation` as first-class OWL relation type. Archival profession's 180-year-old *respect des fonds* principle rendered as a knowledge graph standard — a conceptual ancestor of RankeDB from a completely different intellectual tradition. `read_2.pdf`. **Priority: M — adds archival-theory legitimacy, currently missing.**
> - **Google Always-On Memory Agent (March 2026, open-source).** The explicit opposite of RankeDB. ConsolidateAgent runs every 30 minutes, merging duplicates and dropping information to *"mimic how the human brain processes information during sleep."* No vector DB, no embeddings — the LLM reads, thinks, and writes structured memory into SQLite, making the **LLM the truth arbiter**. RankeDB's motto inversion: *"the graph is the truth, the LLM translates."* `read.pdf`. **Priority: H — cite explicitly as anti-RankeDB in §7.1 contrast.**
> - **XTDB (formerly CruxDB).** Append-only log with native bitemporal support. Rare precedent for add-only temporal database. `read.pdf`. **Priority: L.**
> - **DefraDB / Arweave.** Content-addressable distributed storage with immutability guarantees — check how they handle provenance. `read_2.pdf`. **Priority: L.**

### 3.3 The Provenance DAG as Content

In conventional systems, the knowledge graph is primary and provenance is attached to it as secondary metadata — an annotation layer on top of the "real" content.
RankeDB rejects this split.
The Provenance DAG is not an annotation layer and not a substrate beneath the content: it *is* the content.
The Semantic Graph is a projection from it, optimized for associative retrieval, but every node and edge there points back to a node in the DAG, and the DAG node is what the knowledge actually is.

This inversion has a concrete architectural consequence: operations that would require complex graph surgery in a conventional system become simple view operations in RankeDB.
Reprocessing sources with better tools produces new nodes alongside old ones — no migration required.
Filtering out results from an obsolete worker is a query parameter, not a data operation.
Evaluating competing interpretations of the same source is a traversal of the DAG, not a diff between snapshots.

> **TODO — Reading for §3.3 (the architectural inversion — the core novelty claim):**
>
> ***Read PDF2 in full before rewriting this section.*** PDF2 is entirely dedicated to documenting this inversion as the genuine research gap.
> Its opening claim is the spine of this paper:
>
> > *"No existing system — academic or production — fully implements an architecture where an immutable, append-only provenance DAG serves as the primary data structure for a knowledge system.
> This represents a real and well-documented research gap, not a solved problem repackaged."*
>
> And the core framing:
>
> > *"In every existing system surveyed, the knowledge graph is primary and provenance is secondary metadata attached to it.
> RankeDB proposes the reverse."*
>
> **Sources relevant to §3.3:**
>
> - **RDF-star (being standardized as RDF 1.2).** Embedded triples: `<<:bob :knows :alice>> :source :wikipedia`. ~50% data volume reduction vs classical reification (Ontotext benchmarks, GraphDB 11.2 docs). *Still an annotation mechanism, not a derivation chain system — sharpen this contrast.* `read.pdf` + `read_2.pdf`. **Priority: M.**
> - **Named graphs (Carroll et al. 2005, ACM 1060745.1060835).** Foundational RDF provenance mechanism. W3C Provenance WG (2011) explicitly documented the granularity mismatch: named graphs operate at document level, triple-level requires verbose singleton graphs, derived triples have no natural provenance "home." `read.pdf`. **Priority: M.**
> - **Palantir Foundry.** Tracks complete dataset-level lineage from raw ingestion through all transformations with interactive DAG visualization. **Dataset-level, not fact-level** — sharpen the contrast. `read.pdf`. **Priority: L.**
> - **Google Knowledge Vault (2014) / NELL (CMU).** Per-fact confidence and extraction source tracking, but no complete transformation lineage. `read.pdf`. **Priority: L.**
> - **UaG — Uncertainty-Aware Graph (CIKM 2024).** Conformal prediction in KG-LLM reasoning; uses uncertainty to **guide** reasoning paths. *PDF1: "the closest academic work to RankeDB's provenance as query direction."* `read.pdf`. **Priority: M — elevates §3.3's "provenance guides query behavior" claim beyond mere filtering.**
> - **Dagstuhl survey (2024) on uncertainty in KG construction.** Documents how confidence scores propagate through construction pipelines. Facebook's KG removes facts below confidence thresholds. Treats provenance as a filter, not as substrate — sharpen distinction. `read.pdf`. **Priority: L.**

### 3.4 Visibility as a Derived Property

Access control in RankeDB is not a system feature but an application-level concern expressible within the data model.
Every input artifact has an owner — not a person but a user group.
Groups are hierarchically organized.
A node in the DAG is visible to a user if and only if all of its inputs are visible to that user.
At the root (Level 0), this means ownership by one of the user's groups.

Visibility propagates through the provenance graph: a node derived from one public and one confidential source is automatically confidential. Changing visibility at a source node propagates immediately through all derived nodes — not through explicit re-tagging but through graph traversal. This is compliance by architecture, not by policy.

The classification of nodes into visibility groups is itself a node with provenance — produced by a worker, subject to revision, queryable like any other knowledge.

### 3.5 Reprocessing and Forking

The immutability of Level 0 supports two operational properties of practical value, neither of which amounts to a general rebuild guarantee.

**Reprocessing.** Sources in Level 0 can be re-run through new workers as tools improve.
A better OCR engine in 2028 produces better text extractions from a 2024 photograph; a better summarizer in 2030 produces a better summary from a 2026 conversation.
Old and new outputs coexist (§3.2); the consumer chooses which to prefer.
This is the practical value of keeping the archive byte-exact: the archive does not rebuild the cognitive level, but it keeps the raw material available for every future attempt at it.

**Forking.** Content-addressed blob storage makes forks of the database cheap.
Because blobs are addressed by `content_sha256`, two or more graph instances can share a single blob pool without copying bytes — only the graph (Postgres and FalkorDB) needs to be duplicated, and the graph is small relative to the blobs.
In practice the fork can be even lighter: FalkorDB is a deterministic projection of Postgres and can be rebuilt rather than copied.
This enables experimentation, A/B testing of worker pipelines, and isolated development against production data without touching it.

Backups correspondingly have two parts.
The graph must be backed up in full (Postgres, and optionally FalkorDB); the blob pool must be backed up once and is shared across any number of forks.
A dropped graph is a continuity-of-cognition event (the specific derivation history is gone, though a new one can be grown from the same sources); a dropped blob pool is a true data loss event.

### 3.6 Under-Prescription: A Base for Evolution

RankeDB stores **multiple levels of detail in parallel, networked by provenance.** The raw source artifact sits at Level 0.
Every intermediate derivation — normalization, extraction, summary, alias resolution, classification — sits as a node in Level 1.
Projected entities and relations sit in Level 2 as semantic triplets.
Nothing is omitted at any level.
A consumer can traverse from a semantic triplet down to the exact byte range in the raw source that supports it, or from a raw source up through every derivation it participated in, in a single query across the three levels.

This is a deliberate design choice with a specific purpose: **to leave the strategy of use to the consumer, and to expect that strategy to evolve.** RankeDB does not decide in advance which level of detail is the right one to query, which granularity is best for which question, or which projection best serves which application.
It captures everything and exposes everything.
Consumers — memory agents, analytics pipelines, experimental workflows — work out their own strategies against the captured data, and when a better strategy emerges, it runs over the same data without migration.

We call this **under-prescription**: the database is deliberately short on commitments that would constrain future use.
The alternative — committing to a specific retrieval strategy, a specific indexing scheme, a specific granularity — makes today's consumers faster but freezes the design around today's capabilities.
If entity resolution gets better in 2028, a session-centric store from 2026 has already discarded the evidence needed to exploit the improvement.
If a new reasoning pattern emerges in 2030, an aggressively-summarized store from 2026 cannot reconstruct the context that pattern requires.
RankeDB preserves the full take so that future strategies can still run against it.

The design principle: **capture well now with as few decisions as possible that lead to future constraints.** §3.2 says *do not throw knowledge away.* §3.5 says *derived state is always rebuildable.* §3.6 says *do not commit to a single way of using what you kept.* Together they describe a database optimized not for a benchmark, not for a specific application, and **not for today's capabilities** — but for the set of applications that will exist when the capabilities have changed.

The same property that enables strategy evolution over time also enables **strategy pluralism at any given time.** Because every level of detail is preserved and independently addressable, specialized agents with different strategies of recall can operate on the same data in parallel — one walking the Semantic Graph at L2, another pulling raw spans from L0, a third reasoning over the provenance chains in L1 — without any of them precluding the others.
They can **compete** (running independently and having their results ranked against one another), **cooperate** (one agent's output becoming context for another's query), or **coexist** as alternatives that a higher-level mechanism selects between.
Each agent can pick the level of detail its strategy needs without negotiating with the others.
This is the substrate on which the multi-agent coordination mechanisms in the companion paper on multi-agent systems are built: the base that makes parallel, competing, and cooperating agents a native capability of the data model rather than an application-layer convention.

This stance has a concrete consequence for how the rest of the RankeDB papers should be read.
The companion paper on workers describes *one way* to populate the levels, using the tools we have today.
The companion paper on chat and memory agents describes *one way* to consume them, using the strategies we understand today.
Neither claims that its pipeline or its strategy is the final answer — both are first-generation consumers of a base that is designed to outlive them.
The database's job is to make sure that when the second, third, and tenth generations arrive, their data is already waiting for them.

## 4. Reference Implementation

The sections below describe a specific **proof-of-concept implementation** of the RankeDB architecture.
The architectural claims in §2 and §3 — three levels networked by provenance, append-only derivation, Semantic Graph as materialized projection, under-prescription as a design stance — do not depend on this particular choice of storage engines.
The reference implementation splits Level 0, Level 1, and Level 2 across three engines (S3-compatible object store, Postgres, FalkorDB) because each engine is well-suited to one level's access pattern, and because off-the-shelf components let us validate the concept quickly with existing operational know-how.
The split is not essential to the architecture.
A single database capable of content-addressable blob storage, append-only DAG traversal, and property-graph projection could in principle host the entire stack, and we expect such consolidated implementations to emerge if the underlying concept proves valuable.

**RankeDB is a data structure and a set of invariants, not a deployment topology.** The paper's core claims rest on the architectural and philosophical arguments in §2, §3, and §6 — not on properties of this particular stack.
The empirical validation of the architecture's usefulness is delivered in the companion papers: one demonstrates that the levels can be populated by a worker pipeline, another that the levels can be consumed by a chat and memory-agent stack.
Readers interested in measured performance should look there.
This paper's job is to show that the shape is right.

The reference implementation is delivered as two components: a **server** that encapsulates the three storage engines behind a single API, and an **Explorer** that serves as a visual front-end for developing and inspecting the data model.
Both are development tools for the current phase of the project; neither is essential to the architecture.

### 4.1 RankeDB

RankeDB is a self-contained server deployed as a Docker Compose stack.
It encapsulates the three storage engines (S3-compatible object store, Postgres, FalkorDB) behind a single REST API.
The storage engines are hidden implementation details — consumers interact exclusively with the API, which enforces all invariants: immutability, acyclicity of the Provenance DAG, mandatory provenance on every node, and content-addressability of sources.

The API is the sole interface to the system.
There is no query language in the traditional sense — the API *is* the query language, imperative rather than declarative.
All operations — ingesting sources, creating nodes, traversing provenance chains, querying the Semantic Graph — are API calls.
Workers, the Explorer, and any future application consume the same interface.

RankeDB is the data platform, not an application.
What runs on top — chat interfaces, memory agents, research tools — is delivered by clients that consume the API.
The stack is designed to run on a single host in the reference deployment, but the API contract is the same regardless of topology.

#### 4.1.1 Level 0: S3-compatible object store

The reference deployment uses Hetzner Object Storage, but any S3-compatible provider will do — S3 is a de-facto standard whose API surface is stable since 2006, and migration to another provider is a single `rclone sync` away (Backblaze B2, Cloudflare R2, AWS, others).
Buckets are configured with versioning enabled (as a guard against accidental deletion during development) and with Object Lock (WORM) for production operation, to make immutability enforceable at the storage layer rather than only at the API layer.

Source nodes are stored with the SHA-256 content hash as the object key.
The API-level metadata fields defined in §2.1 (Nodes) are mapped to S3 user-metadata headers (prefixed `x-amz-meta-`) — this mapping is internal to the storage layer and invisible to API consumers.

Access to the bucket is split along least-privilege lines: ingest workers hold a key pair with `s3:PutObject` and `s3:ListBucket` only — enough to write new records and check for duplicates, but not to read existing content — while the RankeDB backend holds a full read/write key pair.
This makes ingest workers blind to the archive they feed, which is useful both as a security property (a compromised worker cannot exfiltrate the archive) and as an architectural discipline (workers cannot accidentally become RankeDB-aware).

#### 4.1.2 Level 1: Postgres

Level 1 runs in Postgres.
Postgres holds the full content of every derived node, the Provenance DAG as a table of edges, full-text search indices (via `tsvector`), vector embeddings (via pgvector, in a separate table with a foreign key to the content), user accounts, auth, and configuration.
Backups are taken with `pg_dump` and shipped to Level 0 storage — Postgres itself is therefore rebuildable from the combination of its source nodes and its schema.

#### 4.1.3 Level 2: FalkorDB

Level 2 runs in FalkorDB.
It holds a lightweight projection of Level 1 nodes (without full content) together with the semantic edges: associative, cross-domain, potentially cyclic, and suitable for traversal.
Because Level 2 is a materialized view of Level 1, it is rebuildable at any time from Postgres alone, and backups are optional — a dropped FalkorDB instance is a rebuild event, not a data loss event.

#### 4.1.4 The API contract

The API is the only way into RankeDB, and it makes three guarantees on behalf of every caller:

- **Immutability.** Once written, a node is never modified. Corrections are new nodes with provenance that refers to the thing they correct.
- **Provenance.** Every derivation carries the source(s) it was produced from and the tool that produced it. The API refuses to create a node without these.
- **Access control.** Visibility is derived from the provenance graph (§3.4). The API enforces it on every read.

What lies beneath the API — which object store, which relational database, which graph engine — is implementation detail and can change without affecting consumers.

#### 4.1.5 Implementation sequence

Each level is independently useful before the next one exists, which makes the rollout incrementally valuable rather than all-or-nothing:

1. **Level 0 and a first ingest worker.** Full capture: stop losing data. A single source (for example, an email archive) is enough to start.
2. **Postgres and the ingestion pipeline.** Content extraction, node creation, full-text search, provenance. RankeDB is a searchable archive with derivation history at this point, even without a Semantic Graph.
3. **Vector embeddings.** pgvector, semantic similarity search over the content.
4. **FalkorDB and semantic projection.** The Semantics level. RankeDB becomes a navigable Semantic Graph with full provenance back to source.

The design and implementation of the workers that drive this pipeline are the subject of the companion paper on RankeDB Workers.

### 4.2 RankeDB Explorer

RankeDB Explorer is a bundled visual interface for navigating and inspecting the RankeDB data model.
It is the first and reference application built against the RankeDB API, shipped alongside RankeDB but architecturally separate — it reads from the API like any other consumer.

The Explorer serves three purposes:

- **Provenance inspection.** Given any node in the Semantic Graph (Level 2), the Explorer traces its full derivation chain through the Cognition level (Level 1) down to the Sources (Level 0). This makes the "everything is knowledge" principle tangible: a user can follow any fact back to the raw source that produced it, through every intermediate processing step.
- **Graph exploration.** The Semantic Graph can be navigated visually — entities, relations, temporal validity, confidence scores. This provides the primary human interface to RankeDB's knowledge state, since Level 2 is optimized for exactly this kind of associative traversal.
- **Architecture validation.** The Explorer demonstrates that the API is sufficient to support rich interactive applications. If the Explorer can render full provenance chains, temporal graphs, and cross-level traversals, then so can any downstream application — agent systems, dashboards, or export tools.

The Explorer is not part of RankeDB's core architecture.
It is an application.
But it is bundled because a database without a way to see its contents is not usable as a research tool — and RankeDB, in its current phase, is primarily a research tool.

## 5. Workers

RankeDB is a database.
It does not contain application logic, AI models, or processing pipelines.
External processes — **workers** — interact with RankeDB exclusively through the API, reading existing nodes and writing new nodes.
Workers are applications implemented against the API; RankeDB defines categories of worker patterns but leaves concrete implementations to the application layer.

Two broad categories emerge in practice.
**Reactive workers** poll for unprocessed nodes whose content type matches their profile, producing new nodes of a different content type — format converters, bulk-archive unpackers, normalizers, fact extractors.
**Analytical workers** traverse the DAG more freely, searching for contradictions, gaps, or patterns across existing nodes.
Both categories interact with RankeDB through the same API; the distinction is in their traversal strategy, not their interface.

Workers may be LLM-based (entity extraction, summarization, synthesis), deterministic (format conversion, deduplication detection), or hybrid.
RankeDB is agnostic to the nature of its workers — it tracks only the provenance of their outputs: what inputs they consumed, which tool and configuration they used, and when they ran.
Workers are identified by run IDs, enabling administrative operations (such as purging defective runs) without affecting the knowledge model.

The same node can be processed by multiple workers or by successive versions of the same worker.
Old and new results coexist with full provenance.
Consumers select between them through view configuration (e.g., preferring the most recent extractor), not through data operations.

The design, implementation, and evaluation of a concrete worker pipeline — from raw data ingestion through normalization, summarization, and entity extraction to a populated Semantic Graph — is the subject of a companion paper.

### 5.1 From Foundation to First Generation

> **TODO — Write §5.1 (closing of Part II, 1–2 short paragraphs).** Frame Papers 2–4 as the **first generation of application** on the foundation described in Part II — not independent downstream products, but the test of whether the philosophy-derived architecture (Part I) actually bears load.
> Intended points:
>
> - Paper 2 (workers): a first-generation population strategy — ingestion, normalization, classification, extraction, summarization — using the tools available today (2026). A different generation of workers, built on different technology, will run over the same DAG without migration.
> - Paper 3 (chat and memory agents): a first-generation consumption strategy against the populated graph, delivering the five long-term-memory abilities identified by Wu et al. (2025, LongMemEval). A different generation of consumers will run against the same graph with different retrieval strategies.
> - Paper 4 (multi-agent coordination): a first-generation pluralism strategy — competing, cooperating, coexisting agents on the same substrate — enabled by the under-prescription stance of §3.6.
> - The foundation is a falsifiable bet. The optimistic outcome: the assumptions hold, the first generation proves the base useful, later generations continue to build. The pessimistic outcome: the assumptions do not hold, and the system is less useful than the philosophy promised. Paper 1 owes the argument for trying; Papers 2–4 owe the evidence that the trying was worth it.

## 6. Related Work

> **TODO — §6 is currently under-cited relative to PDF1 (8 research areas) and PDF2 (5 closest systems + 5-angle gap).
> New subsections to add: Quit Store, Blue Brain Nexus, SPADE, Fluree detail, Xanadu, Helland, RiC-O, PROV-AGENT, Bitemporal KGs (AeonG/BiTRDF), Personal Knowledge Graph community (Balog/Stavanger), JTMS/ATMS/AGM, Senzing, Tools-for-Thought lineage.
> See per-subsection TODOs below.**

### 6.1 Temporal Knowledge Graphs: Graphiti/Zep

Graphiti (Zep, 2024–2025) is the closest existing system to RankeDB in the LLM context management space.
It builds temporal, provenance-aware knowledge graphs using FalkorDB or Neo4j, with bidirectional episode indices and temporal validity windows.
Facts are invalidated rather than deleted.

However, Graphiti performs destructive entity summary updates, lacks a separate content-addressable source archive (Level 0), embeds provenance in the knowledge graph rather than maintaining a separate provenance DAG, and does not treat provenance as queryable knowledge.
RankeDB can be understood as an extension of Graphiti's philosophy — adding immutability, a source archive, and the architectural inversion that makes provenance the substrate rather than an annotation.

> **TODO — Reading for §6.1 (Graphiti/Zep expansion):**
>
> - **"Graphiti: Knowledge Graph Memory for AI Agents"** (Rasmussen et al., arXiv 2501.13956, January 2025). **Read in full before finalizing §6.1.** `read_2.pdf`. **Priority: H.**
> - `read.pdf` and `read_2.pdf` detail to add:
>   - **94.8% on the DMR benchmark**, P95 retrieval latency **300ms**.
>   - Three-layer architecture paralleling RankeDB: episodic subgraph (raw events) / semantic subgraph (extracted facts) / community subgraph.
>   - Uses the **same graph databases RankeDB specifies** (Neo4j OR FalkorDB).
>   - Bi-temporal `t_valid`/`t_invalid` fields; old facts *"invalidated, not deleted."*
>   - **55-60% architectural overlap** with RankeDB (PDF2 estimate).
>   - *Key differentiator:* Graphiti performs **destructive entity summary updates** (arXiv 2501.13956 explicit). This is the precise distinction RankeDB maintains.
>   - Graphiti's **"non-lossy" design philosophy** is PDF2's identified "closest articulation" of RankeDB's accumulation bet — but Graphiti still consolidates at entity level.
> - **Priority: H — Graphiti is the single most important comparison in the paper.**

### 6.2 Versioned Knowledge Bases: TerminusDB

TerminusDB provides Git-like versioning (branch, merge, time-travel) over an RDF knowledge graph using append-only delta encoding.
It captures *what* changed across versions but not *why* — there is no derivation chain, no source archive, and no concept of workers as provenance-tracked processors.
Its foundational structure is a versioned graph, not a provenance DAG.

> **TODO — Reading for §6.2 (TerminusDB expansion):**
>
> - **Mendel-Gleason et al. TerminusDB technical paper.** Already in references. Read before finalizing §6.2.
> - `read_2.pdf` detail to add:
>   - Origin: Trinity College Dublin, Horizon 2020 ALIGNED project (owlapps 62518551).
>   - Uses append-only **succinct data structures with delta encoding**.
>   - Every transaction creates a new immutable layer.
>   - **~75-80% architectural overlap with RankeDB — highest of all surveyed systems.**
>   - Missing: no content-addressable blob store for raw artifacts; no concept of transformation workers or AI processors as first-class participants; foundational structure is a *versioned RDF graph*, not a provenance DAG; tracks *what* changed but not *why or how knowledge was derived*.
> - **Priority: H.**

### 6.3 Immutable Databases: Datomic and Fluree

Datomic (Hickey, 2012) operationalizes Pat Helland's "Immutability Changes Everything" thesis as an append-only database of immutable datoms.
Fluree combines an append-only ledger with a semantic graph database.
Both capture temporal history but not epistemic history — they record *when* facts changed but not *how knowledge was derived from sources through processing chains*.

> **TODO — Reading for §6.3 (Datomic/Fluree expansion + Helland):**
>
> - **Helland, P. (2015). "Immutability Changes Everything."** CIDR 2015. `cidrdb.org/cidr2015/Papers/CIDR15_Paper16.pdf` + ACM Queue 2884038. **Read in full — this is the theoretical foundation of RankeDB's §3.2.** Core quotes: *"accountants don't use erasers"*, *"the truth is the log; the database is a cache of a subset of the log."* **PDF2 explicitly: "No subsequent work has explicitly applied Helland's thesis to knowledge graphs, despite its enormous influence."** RankeDB is the first. `read_2.pdf`. **Priority: H.**
> - **Nubank case study.** Engineers applied Datomic to microservice dependency graphs and used the phrase *"immutable knowledge databases"* — closest vernacular antecedent for RankeDB's framing. `read_2.pdf`. **Priority: L.**
> - `read_2.pdf` Fluree detail: founded 2017, supports RDF + JSON-LD + SPARQL + SHACL validation. Every update cryptographically chained, enabling time-travel and verifiable data history. **PDF2: "perhaps the closest *production database* to the RankeDB vision."** But Fluree's immutability operates at the transactional ledger level, not at the level of a provenance DAG tracking derivation chains. No separate content-addressable blob store. AI/ML processors not modeled as first-class graph participants. **Priority: M.**
> - Datomic nuance from PDF2: captures the *temporal* dimension (what changed, when) but **not the *epistemic* dimension** (how knowledge was derived, from what evidence, by what process). This is the distinction RankeDB introduces. **Priority: M.**

### 6.4 W3C PROV-DM

The W3C PROV Data Model provides a formal vocabulary for provenance (Entity, Activity, Agent, wasGeneratedBy, wasDerivedFrom, used).
RankeDB's internal model is semantically compatible with PROV-DM — nodes map to Entities, worker runs to Activities, tools to Agents — but RankeDB does not depend on or implement the W3C stack (RDF, SPARQL, OWL).
PROV-DM compatibility exists at the conceptual level, enabling potential export or interoperability without architectural coupling.

> **TODO — Reading for §6.4 (PROV-DM expansion):**
>
> - **Sikos & Seneviratne (2020). "Provenance-Aware Knowledge Representation: A Survey of Data Models and Contextualized Knowledge Graphs."** *Data Science and Engineering.* **Already in references. Read in full — this is the canonical survey of every RDF-provenance workaround.** Key finding: *RDF "inherently lacks the mechanism to attach provenance data"* — reviews named graphs, reification, RDF-star, singleton properties, nanopublications, finds none fully satisfactory. `read_2.pdf`. **Priority: H.**
> - `read.pdf` PROV-O adoption data: **OpenCitations tracks over 2 billion citations using PROV-O**; 2025 Nature Scientific Data paper aligned PROV-O with the ISO-standard Basic Formal Ontology (BFO). `read.pdf`. **Priority: M.**
> - **Carroll et al. (2005).** *"Named Graphs, Provenance and Trust"* (ACM 10.1145/1060745.1060835). Foundational paper establishing named graphs for provenance. W3C Provenance Working Group (2011) documented the granularity mismatch. `read.pdf`. **Priority: M.**
> - **RDF-star / PROV-STAR as bolt-on provenance.** PDF2 gap table: "Nanopubs are flat collections; PROV-STAR is a bolt-on." Contrast with RankeDB's substrate approach. `read_2.pdf`. **Priority: M.**
> - **PROV-AGENT (Souza et al., IEEE e-Science 2025, arXiv 2508.02866).** First provenance framework for AI agent workflows; extends W3C PROV with agent-specific metadata. *Operates within traditional workflow orchestration rather than proposing a provenance-first architecture.* `read_2.pdf`. **Priority: M — currently missing from §6, must add.**

### 6.5 Nanopublications

Nanopublications (Kuhn & Dumontier, 2014) are immutable, content-addressable scholarly assertions with embedded provenance.
They share RankeDB's commitment to immutability and provenance-per-assertion but are a flat collection of independent assertions — they do not form a derivation DAG connecting assertions through chains of processing, and they do not support a semantic graph layer.

> **TODO — Reading for §6.5 (Nanopublications expansion):**
>
> - **Kuhn & Dumontier (2014). "Trusty URIs."** ESWC 2014. Already in references. Read for the content-addressability mechanism (cryptographic hash URIs).
> - `read_2.pdf` detail: over **10 million nanopublications** exist, primarily in life sciences. Each nanopub contains three named RDF graphs: assertion, provenance, publication info. **Priority: M.**
> - **2025 extension: "Nanopublications with Knowledge Provenance"** (International Journal of Digital Libraries, Springer s00799-025-00431-x). Extends with trust networks where multiple agents assign truth values on a 0-1 scale — *parallel to RankeDB's conviction levels, though at scientific publication level rather than personal knowledge.* `read.pdf`. **Priority: M — add to references.**

### 6.6 TODO: Additional prior art (currently missing from §6, must add)

The following systems and traditions are flagged by PDF1 or PDF2 as meaningfully close to RankeDB but are not currently covered.
Each is a new subsection to write.

> **TODO — §6.6.1 Quit Store (AKSW Leipzig):**
>
> - SPARQL 1.1 endpoint backed entirely by Git. RDF named graphs stored as canonicalized N-Quads in Git's SHA-1 content-addressed object store. Automatic W3C PROV-O generation from commit metadata. `quit blame` for per-statement provenance.
> - **~60-65% architectural overlap with RankeDB — second-closest in PDF2.**
> - Literally uses Git's Merkle DAG as storage layer for a KG with provenance.
> - Missing: only structured RDF (not raw artifacts like PDFs); academic prototype with performance limitations; provenance derived from version control metadata, not an explicit derivation DAG.
> - Reference: ScienceDirect S1570826818300416; CEUR-WS Vol-1824 mepdaw_paper_2.pdf. `read_2.pdf`. **Priority: H.**

> **TODO — §6.6.2 Blue Brain Nexus (EPFL):**
>
> - Open-source neuroscience data management platform. Semantic Web Journal 2023. W3C PROV as provenance backbone, SHACL validation, event-sourced streaming architecture. Explicitly treats *"provenance as a first-class citizen."*
> - PDF2: *"deserves special mention as the closest working knowledge platform with provenance aspirations. Even here, the knowledge graph is primary and provenance enriches it — the architectural inversion remains unmade."*
> - Reference: ResearchGate 330751750. `read_2.pdf`. **Priority: M.**

> **TODO — §6.6.3 SPADE (SRI International):**
>
> - Provenance auditing system storing derivation chains in Neo4j OR Postgres, abstracting over both through its QuickGrail query language.
> - **Only existing system with split-store architecture analogous to RankeDB's FalkorDB + Postgres.**
> - Reference: ACM Queue 3476885. `read.pdf`. **Priority: H — currently missing.**

> **TODO — §6.6.4 Project Xanadu (Nelson, 1960-present):**
>
> - Specified immutable, add-only content space; documents as lists of pointers to regions in an ever-growing store; transclusion maintains visible provenance to source; bidirectional connections.
> - **PDF2: "arguably the direct ancestor of what RankeDB proposes — append-only content-addressable storage with provenance as the organizing principle."**
> - Cautionary lesson: Xanadu's refusal to compromise on its complete vision prevented adoption while the simpler WWW prevailed.
> - References: Grokipedia entry on Project Xanadu; WebProNews coverage. `read_2.pdf`. **Priority: H — currently missing.**

> **TODO — §6.6.5 Records in Contexts (RiC-O v1.1, May 2025, ICA):**
>
> - International Council on Archives standard. Describes archival world as *"a graph of interconnected things."*
> - Models `rico:ProvenanceRelation` as first-class OWL relation type for linked data.
> - Archival profession's **180-year-old *respect des fonds* principle** rendered as a knowledge graph standard.
> - Intellectual ancestor from an entirely different tradition than CS. `read_2.pdf`. **Priority: M.**

> **TODO — §6.6.6 Bitemporal Knowledge Graphs: AeonG, BiTRDF, XTDB, OSTRICH:**
>
> - **Chekol et al. (2018)** explicitly identified the bitemporal KG gap: Wikidata uses only valid time, NELL uses only transaction time (ACM 3184558.3191637).
> - **AeonG (Anselma et al., ADBIS 2025).** Extends property graphs with explicit bitemporal timestamps on every element — **9.74% performance overhead** (Università di Torino). Reference: Springer 978-3-032-05281-0_15. **Priority: M.**
> - **BiTRDF (MDPI Mathematics 2025).** Adds both temporal dimensions to RDF. Reference: MDPI 2227-7390/13/13/2109. **Priority: M.**
> - **XTDB (formerly CruxDB).** Append-only log with native bitemporal support — precedent in temporal databases, rare in KG systems. **Priority: L.**
> - **OSTRICH (Taelman et al., Journal of Web Semantics 2018).** Versioned RDF triple store with append-only delta ingestion, three query types across versions (ScienceDirect S1570826818300404). **Priority: L.**
> - **Key framing for RankeDB §6.6.6:** RankeDB achieves **emergent bitemporality through architectural composition** (valid time on L2 edges, transaction time via L1 DAG) — architecturally simpler than explicit bitemporal annotations. `read.pdf`. **Priority: M — this is a distinctive selling point worth a dedicated subsection.**

> **TODO — §6.6.7 Event Sourcing as KG substrate:**
>
> - **Telicent CORE platform.** Event-driven KG using Apache Kafka as the event log backbone, with events flowing through topics into RDF format and multiple derived stores (graph, search, vector). Reference: telicent.io/news/event-driven-knowledge-graphs.
> - **TMForum "Atomic Events" model.** Append-only EAV events with timestamps for building temporal KGs.
> - **Key distinction:** event sourcing stores a *linear sequence* of events per aggregate; a provenance DAG captures a *richer graph* of derivation relationships. **No published work frames event sourcing specifically as a knowledge management pattern.** `read_2.pdf`. **Priority: L.**

> **TODO — §6.6.8 Personal Knowledge Graphs (academic community):**
>
> - **Balog, K. (University of Stavanger).** "Personal Knowledge Graphs: A Research Agenda." ICTIR 2019. Foundational paper.
> - **"An Ecosystem for Personal Knowledge Graphs"** (ScienceDirect 2024, S2666651024000044). Survey defining PKGs around data ownership by single individual and personalized service delivery.
> - **PKG API (WWW Companion 2024, ACM 3589335.3651247).** Proposes RDF-based PKG vocabulary with provenance and access rights.
> - **PDF1 observation:** PKG academic community primarily targets **recommendation and personalization** — not the **cognitive augmentation** RankeDB pursues. RankeDB's position is distinct from the Balog line of work. `read.pdf`. **Priority: M — framing.**

> **TODO — §6.6.9 Truth Maintenance & Belief Revision (intellectual ancestors):**
>
> - **Doyle, J. (1979). JTMS.** *Artificial Intelligence.* Dependency network for beliefs and justifications. **Direct ancestor of RankeDB's provenance model.** `read.pdf`. **Priority: H.**
> - **de Kleer, J. (1986). ATMS.** Maintains all alternative assumption sets simultaneously. **Closest to RankeDB's add-only preservation of competing beliefs.** `read.pdf`. **Priority: H.**
> - **AGM (1985).** Formal postulates for belief change. JSTOR 41487515. `read.pdf`. **Priority: M.**
> - **"Graph-Native Cognitive Memory for AI Agents"** (arXiv 2603.17244, 2025-2026). Applies AGM to Neo4j AI memory. **Closest published work to RankeDB epistemology.** `read.pdf`. **Priority: H.**
> - This section would elevate RankeDB's intellectual lineage beyond database systems to include 45 years of AI knowledge representation work.

### 6.7 The Identified Gap

No existing system combines: (a) a content-addressable immutable source archive, (b) an append-only Provenance DAG as the primary data structure, (c) a semantic knowledge graph as a materialized view with per-edge provenance, and (d) natural-language relations with emergent ontology.
Each component has mature prior art; the architectural composition is novel.

> **TODO — Reading for §6.7 (rewrite with PDF2's 5-angle gap framing):**
>
> PDF2 documents the gap being identified from **five independent research angles**, none of which propose the integrated solution.
> Use this as the structural spine of §6.7:
>
> 1. **Knowledge graph engineering.** Sikos (2020), Takan (2023, PeerJ, *"no research on immutability in knowledge graphs"*), Dibowski (FOIS 2024). All document that provenance in KGs is fundamentally unsolved.
> 2. **LLM / AI provenance.** 2025 Frontiers in Computer Science survey on KG-LLM fusion identifies *"unclear knowledge provenance"* as a key challenge. PROV-AGENT (Souza et al. IEEE e-Science 2025) is the first provenance framework for AI agent workflows.
> 3. **Scientific reproducibility.** **72-83% of researchers acknowledge a reproducibility crisis** (SEC.gov/files/ctf-written-input-knowledge-provenance-protocol-kpp). REPRODUCE-ME ontology (2022), Knowledge Provenance Protocol (KPP 2025) — both DAG-based but domain-specific.
> 4. **Enterprise AI governance.** Amazon Bedrock AgentCore (2025) adopted append-only memory patterns marking outdated memories INVALID rather than deleting. Bolt-on solution.
> 5. **Content addressability for AI.** ISPE article (Jan 2026) advocates content-addressable storage for AI knowledge management, predicts *"AI copilots with built-in provenance: every answer cites the exact CIDs used."* Closest industry-perspective articulation.
>
> - **ICLR 2026 Workshop on Memory for LLM-Based Agentic Systems (MemAgents).** Explicitly calls for research on *"provenance-aware retrieval"* and *"structured memory access control."* **Community recognition of the open problem.** OpenReview U51WxL382H; arXiv 2603.10062. `read_2.pdf`. **Priority: H — cite as evidence that the gap is recognized in 2026 by the top ML venue.**
>
> **Priority: H — this section should become the longest in §6.**
>
> **Property-by-property novelty table (from PDF2 §5):** copy this table directly into §6.7:
>
> | RankeDB Property | Closest Existing System | What's Missing |
> |---|---|---|
> | Content-addressable immutable blob store (SHA256) | IPFS/IPLD, Git, DefraDB | Not integrated with KG layers |
> | Provenance DAG as primary data structure | **Nothing** — all systems treat provenance as secondary | **The core architectural inversion** |
> | Semantic graph with per-edge provenance to DAG | Nanopubs, RDF-star + PROV-STAR | Nanopubs flat; PROV-STAR bolt-on |
> | Strictly append-only, no destructive operations | Datomic, Fluree, Arweave | Not combined with KG + provenance DAG |
> | AI/LLM as just one type of graph processor | PROV-AGENT (2025) | Tracks agent provenance within workflows, not provenance-first |
> | Three-layer architecture (blobs → DAG → semantic graph) | **No system combines all three** | **The unified architecture is novel** |

## 7. Discussion

### 7.1 The Context Window Bet

RankeDB's append-only accumulation model is predicated on a specific technological trajectory: that large language model context windows will become fast and cheap enough to consume full derivation histories.
If this trajectory holds, systems that destructively consolidate today will be unable to reconstruct the inferential context that RankeDB preserves.
If it does not hold, RankeDB pays a storage and retrieval cost for history that cannot be effectively utilized.

Current trends support the bet.
Context windows have grown from 4K tokens (2022) to 200K+ tokens (2025), with costs per token declining by orders of magnitude.
The architectural question is not whether models *can* process full provenance chains, but when they can do so at acceptable latency and cost for interactive use.

The bet is also a deliberate rejection of the three paths the field currently takes to equip chat assistants with long-term memory.
Wu et al. (2025) summarize them as follows:

> "To equip chat assistants with long-term memory capabilities, three major techniques are commonly explored.
> The first approach involves directly adapting LLMs to process extensive history information as long-context inputs (Beltagy et al., 2020; Kitaev et al., 2020; Fu et al., 2024; An et al., 2024).
> While this method avoids the need for complex architectures, it is inefficient and susceptible to the 'lost-in-the-middle' phenomenon, where the ability to utilize contextual information weakens as the input length grows (Shi et al., 2023; Liu et al., 2024).
> A second line of research integrates differentiable memory modules into language models, proposing specialized architectural designs and training strategies to enhance memory capabilities (Weston et al., 2014; Wu et al., 2022; Zhong et al., 2022; Wang et al., 2023).
> Lastly, several studies approach long-term memory from the perspective of context compression..."
> — Wu et al., *LongMemEval* (ICLR 2025), arXiv 2410.10813v2

Each of these approaches commits the memory solution *to the language model itself* — by expanding its context, modifying its architecture, or compressing what it receives.
**RankeDB commits to none of them.** The memory lives as a structured, provenance-addressable database *outside* the model.
The reader decides at query time what slice of the available data to consume; the model sees only the slice chosen for it; the rest remains intact and addressable for a different slice next time, by the same model or a different one.
The bet on context window growth is not a bet on cramming everything into the prompt — it is a bet that, when model capabilities improve, consumers will be able to ask for larger, richer slices from the same base without reprocessing their inputs and without changing the database.
The memory problem and the modeling problem are decoupled by construction.

> **TODO — Reading for §7.1 (accumulation vs destructive consolidation — map the opposition):**
>
> PDF1's LLM-driven KG construction section (§7) is **the single most useful source for this subsection**.
> It explicitly places RankeDB at one extreme of a spectrum and Google's Always-On Memory Agent at the other.
>
> - **Google Always-On Memory Agent (March 2026, open-source).** **The anti-RankeDB.** ConsolidateAgent runs every 30 minutes, explicitly merging duplicates and dropping information to *"mimic how the human brain processes information during sleep."* No vector DB, no embeddings — LLM as truth arbiter. References: digit.in/features/general/googles-new-ai-agent-remembers-everything; elephaant.com/blog/google-always-on-memory-agent-vector-db-alternative-2026. `read.pdf`. **Priority: H — cite as explicit counter-design.**
> - **Graphiti "non-lossy" design philosophy** (getzep.com 2025 report). PDF2: *"the closest articulation"* of RankeDB's accumulation bet — but Graphiti still performs destructive entity summary updates. The closest ally; not quite an ally. `read_2.pdf`. **Priority: H.**
> - **Microsoft GraphRAG.** Community summaries are regenerated rather than appended, replacing old versions. Extraction not fully reproducible. Reference: microsoft.com/en-us/research/blog/graphrag-unlocking-llm-discovery-on-narrative-private-data. `read.pdf`. **Priority: M.**
> - **LightRAG (EMNLP 2025).** Entity deduplication merges identical entities with **no history preservation.** lightrag.github.io. `read.pdf`. **Priority: M.**
> - **EDC Framework (Zhang & Soh 2024).** Canonicalization phase explicitly consolidates schema components. `read.pdf`. **Priority: L.**
> - **iText2KG / ATOM (AuvaLab 2025).** Dual-time modeling preserves temporal metadata, but performs entity merging. github.com/AuvaLab/itext2kg. `read.pdf`. **Priority: M.**
> - **Collaborative Memory (arXiv 2505.18279).** Each memory fragment carries immutable provenance attributes — partial alignment with RankeDB. `read.pdf`. **Priority: M.**
> - **Amazon Bedrock AgentCore (2025).** Append-only memory: marks outdated memories INVALID instead of deleting. Bolt-on solution. `aws.amazon.com/blogs/machine-learning/building-smarter-ai-agents-agentcore-long-term-memory-deep-dive`. `read_2.pdf`. **Priority: L.**
>
> *PDF1 key framing to use verbatim:* *"No existing system matches RankeDB's full specification: add-only storage, content-addressable immutable raw sources, no destructive consolidation, conviction-based entity resolution instead of hard merges, and complete inferential history preservation.
> RankeDB's commitment to immutability is more extreme than any published system."*

### 7.2 Reprocessing Without Migration

A distinctive property of RankeDB's architecture is that improvements in processing tools require no data migration.
When a better entity extractor becomes available in 2028, it runs over the same Level 0 sources and produces new nodes in Level 1.
Old and new results coexist.
Consumers select between them through view configuration — filtering by worker version, recency, or confidence score.
The old results remain available as fallback, as training signal, or as historical context.

This property follows directly from immutability and the separation of Level 0 (sources) from Level 1 (derivations).
In systems that update in place, reprocessing is a migration — destructive, irreversible, and operationally risky.
In RankeDB, reprocessing is an append operation.

> **TODO — Reading for §7.2 (reprocessing vs migration — GraphRAG family comparison):**
>
> The GraphRAG landscape is the cleanest contrast point for RankeDB's reprocessing property.
>
> - **Edge et al. (April 2024). "From Local to Global: A Graph RAG Approach to Query-Focused Summarization."** Microsoft Research foundational GraphRAG paper. LLM entity/relation extraction + Leiden community detection + pre-built community summaries. `read.pdf`. **Priority: M.**
> - **DRIFT Search (Microsoft, October 2024).** Combines global and local retrieval with iterative refinement. Reference: microsoft.com/en-us/research/blog/introducing-drift-search. `read.pdf`. **Priority: L.**
> - **LazyGraphRAG (Microsoft, November 2024).** Reduces indexing costs to **0.1% of full GraphRAG** via NLP-based extraction instead of LLM summarization. lianpr.com/en/news/detail/3224. `read.pdf`. **Priority: L — cite as efficiency-trades-history example.**
> - The crucial pattern: **all GraphRAG variants require full reprocessing when the extractor improves.** Contrast explicitly with RankeDB's append semantics.

### 7.3 Toward a CRDT-Compatible Architecture

An add-only monotonic DAG is provably a Conflict-Free Replicated Data Type (CRDT) — it can be replicated across distributed nodes and always merged into a consistent state without coordination.
This property, while not exploited in the current single-node design, suggests that RankeDB's architecture could natively support decentralized, coordination-free knowledge management.
This connection between provenance DAGs and CRDTs appears unexplored in the literature.

> **TODO — Reading for §7.3 (CRDT connection — currently 1 paragraph, PDF2 says this may be the MOST significant unexplored implication of RankeDB):**
>
> PDF2's §3 "Adjacent architectural concepts" closes with:
>
> > *"A deep and underexplored connection exists between CRDTs and provenance DAGs.
> Shapiro et al.'s foundational CRDT work (2011) formally proved that an add-only monotonic DAG is a CRDT.
> Byzantine Fault Tolerant CRDTs use Merkle-DAGs (hash graphs representing causal partial order among updates) that are structurally identical to content-addressed provenance graphs.
> This connection is **entirely unexploited** in the knowledge management literature — a CRDT-based provenance DAG would enable truly decentralized, coordination-free knowledge management with automatic merge, which may be the most significant unexplored implication of the RankeDB architecture."*
>
> **Consider elevating §7.3 from a single paragraph to its own major section, or spin it off as a companion paper.**
>
> - **Shapiro, Preguiça, Baquero & Zawirski (2011). "Conflict-Free Replicated Data Types."** *Proceedings of the 13th International Symposium on Stabilization, Safety, and Security of Distributed Systems (SSS 2011).* Springer 978-3-642-24550-3_29. **Foundational paper. Formally proves: add-only monotonic DAG is a CRDT.** `read_2.pdf`. **Priority: H — must cite.**
> - **Byzantine Fault Tolerant CRDTs with Merkle-DAGs.** PDF2: "structurally identical to content-addressed provenance graphs." Reference: jzhao.xyz/thoughts/CRDT. `read_2.pdf`. **Priority: H.**
> - **IPFS / IPLD.** Content-addressable DAG substrate used in distributed systems — check how they handle epistemic layer (they don't, but understand the primitives). `read_2.pdf §5 table`. **Priority: M.**
> - **DefraDB.** Content-addressable immutable blob store with knowledge graph aspirations. `read_2.pdf §5 table`. **Priority: L.**
> - **Arweave.** Strictly append-only, no destructive operations, permanent storage. dolthub.com/blog/2022-03-21-immutable-database. `read_2.pdf §5 table`. **Priority: L.**

### 7.4 Design Rationale: What the Architecture Makes Possible

Wu et al. (2025, *LongMemEval*) identify five core long-term memory abilities — **information extraction**, **multi-session reasoning**, **knowledge updates**, **temporal reasoning**, and **abstention** — as the coverage axes for long-term memory systems.
The companion papers on workers and on chat/memory agents describe how a pipeline and a consumer stack actually deliver these abilities.
This section answers the complementary question: *why does the database look the way it does?*

RankeDB does not guarantee any of the five.
They are delivered by the consumers built on top.
What the database does is **refuse to foreclose** on them: each architectural choice in §2 and §3 was made to keep each of the abilities **possible** — reachable by *some* consumer strategy — without committing to any particular one.
A system that commits to one granularity, one retrieval path, one indexing scheme, or one consolidation policy makes some subset of the abilities cheap and the rest expensive or impossible.
RankeDB leaves all of them reachable, and leaves the strategy to the consumer.

- **Multiple levels of detail in parallel (§3.6).** *Keeps information extraction and multi-session reasoning possible from the same data.* Raw dialogue, intermediate derivations, and semantic triplets all exist at once, so a consumer can descend to the exact utterance for specific recall *or* aggregate at the entity level for cross-session synthesis — without one undermining the other.

- **Provenance as substrate (§3.3).** *Keeps abstention and knowledge updates possible.* A consumer can check whether a claim has a provenance chain and refuse to answer if it does not — an architectural precondition for any abstention policy above. And because a superseding node links to what it supersedes, current-vs-historical selection is a queryable property, not something lost in a consolidation pass.

- **Timestamps on every node and edge (§2.1.2, §2.1.3).** *Keeps temporal reasoning possible as a first-class query primitive.* Two temporal dimensions are carried at all times: when the underlying event occurred (from source metadata or explicit in-content mentions) and when the node entered the database (transaction time from the L1 DAG). Questions about *when* have direct answers; a consumer does not have to reconstruct temporal order from implicit cues.

- **Temporal validity on L2 edges (§2.1.3).** *Keeps knowledge updates possible without losing history.* Every L2 edge carries `valid_from` and `valid_until`, not just a creation timestamp. A fact true from February to April coexists with a fact true from April onward; both remain retrievable, and which one answers a given query is a view-configuration choice left to the consumer.

- **Append-only over content-addressable source (§3.2, §2.1.1).** *Keeps knowledge updates and abstention possible at the infrastructure level.* Nothing is ever mutated or overwritten, and the raw source is byte-identical to what was ingested. History cannot be corrupted by "updating" a fact, and a claim that cannot be verified today can be re-verified tomorrow against the unchanged source.

This rationale is a *reading* of the architecture, not a derivation of it.
RankeDB was not designed by starting from the five abilities and working backward — it was designed from the provenance-as-substrate inversion (§3.3) and the under-prescription principle (§3.6), and the five abilities fall out as consequences.
But the mapping is clean enough to use as an explanation: when a reader asks *why does the database look the way it does?*, one useful answer is *because each choice keeps one or more of the abilities reachable, without committing to how a consumer reaches them.*

## 8. Conclusion

RankeDB proposes a structural inversion in knowledge system design: provenance as substrate rather than metadata, knowledge graph as view rather than source of truth.
The architecture — immutable source archive, append-only Provenance DAG, Semantic Graph as materialized view — is a composition of individually well-understood components whose integration has not been previously proposed.

The core thesis is that knowledge graph history is not metadata — it is knowledge.
By treating all provenance, all classifications, all assessments as first-class nodes in the same graph, RankeDB eliminates the distinction between data and metadata, between knowledge and knowledge-about-knowledge.
Everything is knowledge.
Everything is graph.

The system is designed for a technological regime that does not yet fully exist — one in which language models can efficiently consume full derivation histories.
This is a deliberate architectural bet, analogous to developing a 3D game engine before consumer hardware can render it in real time.
Systems that accumulate inferential history today will be positioned to provide qualitatively richer context to capable models tomorrow.
Systems that destructively consolidate will not be able to reconstruct what they have discarded.

## References

- Helland, P. (2015). Immutability Changes Everything. CIDR 2015.
- Hickey, R. (2012). Datomic: A Database of Flexible, Time-Based Facts.
- Kuhn, T. & Dumontier, M. (2014). Trusty URIs: Verifiable, Immutable, and Permanent Digital Artifacts for Linked Data. ESWC 2014.
- Mendel-Gleason, G. et al. TerminusDB: An Open Source Model Driven Graph Database for Knowledge Graph Representation.
- Moreau, L. & Missier, P. (2013). PROV-DM: The PROV Data Model. W3C Recommendation.
- Nelson, T. (1965). A File Structure for the Complex, the Changing, and the Indeterminate. ACM/CSC-ER.
- Rasmussen, P. (2025). Zep: A Temporal Knowledge Graph Architecture for Agent Memory.
- Sikos, L.F. & Seneviratne, O. (2020). Provenance-Aware Knowledge Representation: A Survey. Data Science and Engineering.
- Takan, S. (2023). Knowledge Graph Augmentation: Consistency, Immutability, Reliability, and Context. PeerJ Computer Science.

---

*Draft v0.1 — April 2026*
