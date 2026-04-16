# Paper 1 — Reading notes and TODO items

Extracted from `rankedb.md` to keep the main draft clean.
Use this for the quote / citation knock-out session.

---


## Under 1. The Problem: Knowledge Without Provenance

> **TODO — Write §1 opening (1–2 paragraphs).** State the thesis of Part I: traditional knowledge graphs optimize for *current truth* and treat provenance as metadata; this was defensible in an era of expensive storage and limited query capacity; it is untenable in a regime where knowledge is read and written by machines at scale.
> Introduce the three movements (archival tradition → CS priority → LLM rupture) without yet arguing them.
> Source material: `quotes.md`, Talisman, Ranke.
> Close with a one-line preview of §1.4's resolution.


## Under 1.1 The Archival Tradition

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


## Under 1.2 The CS Priority That Was Never Operationalized

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


## Under 1.3 The Rupture: Machines Reading and Writing at Scale

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


## Under 1.4 Convergence: A Foundation, Not a Feature

> **TODO — Write §1.4 closing.** The three movements converge.
> The archival tradition had the insight; the CS literature has the components; the machine-reading/writing era makes the gap urgent.
> A provenance-first foundation is not a refinement — it is the shape that emerges when the three are taken seriously at once.
> Part II describes the foundation; the follow-up papers present the first generation of application that tests whether the philosophy-derived foundation actually bears load.
> The paper is therefore a falsifiable bet: if the assumptions hold, later generations of workers and applications will keep building on the same base; if they do not, the foundation was misjudged.
> Either way, what this paper owes is the argument for why *this* shape is the right one to try.


## Under 2.1 Everything Is Knowledge

> **TODO — Reading for §2.1 (epistemological tradition + the architectural inversion):**
>
> *Two threads to weave in here:*
> *(a) The intellectual lineage of "everything is knowledge" — PDF1 traces a tradition from 1979 TMSes to 2026 AI memory papers that RankeDB sits squarely inside; no existing PKG has operationalized it.*
> *(b) The architectural inversion as the core novelty — PDF2 is entirely dedicated to documenting this as the genuine research gap. **Read PDF2 in full before rewriting this section.** Its opening claim is the spine of this paper:*
>
> > *"No existing system — academic or production — fully implements an architecture where an immutable, append-only provenance DAG serves as the primary data structure for a knowledge system. This represents a real and well-documented research gap, not a solved problem repackaged."*
>
> > *"In every existing system surveyed, the knowledge graph is primary and provenance is secondary metadata attached to it. RankeDB proposes the reverse."*
>
> **Sources — epistemological tradition:**
>
> - **Doyle, J. (1979). "A Truth Maintenance System."** *Artificial Intelligence.* JTMS — dependency network of beliefs and justifications; traces conclusions to premises; propagates revision through the network. **RankeDB's direct intellectual ancestor.** `read.pdf`. **Priority: H.**
> - **de Kleer, J. (1986). "An Assumption-Based TMS."** ATMS extends JTMS to maintain all alternative assumption sets simultaneously — conceptually parallel to RankeDB's add-only preservation of multiple states of belief. `read.pdf`. **Priority: H.**
> - **Alchourrón, Gärdenfors & Makinson (1985). AGM framework.** Formal postulates for rational belief change (expansion, revision, contraction). See SEP entry `plato.stanford.edu/entries/logic-belief-revision/`. `read.pdf`. **Priority: M.**
> - **"Graph-Native Cognitive Memory for AI Agents: Formal Belief Revision Semantics for Versioned Memory Architectures"** (2025-2026, arXiv 2603.17244). Applies AGM postulates to a Neo4j-based memory architecture for AI agents — proves graph memory operations can satisfy formal belief revision axioms. **Closest published work to RankeDB's epistemological framing**, targets AI agent memory rather than personal cognition. `read.pdf`. **Priority: H.**
> - **Carneades argumentation framework.** Varying proof standards per statement — direct analog to per-entity conviction levels. `read.pdf`. **Priority: L.**
> - **ASPIC+ framework.** Strict vs defeasible inference with three attack types (undermining premises, rebutting conclusions, undercutting rule applicability). `read.pdf`. **Priority: L.**
> - **AKReF (2025, arXiv 2506.00713).** Constructs argumentation knowledge graphs from text using ASPIC+. Heterogeneous graphs with argument nodes and attack/support edges. `read.pdf`. **Priority: L.**
> - *PDF1 observation:* the phrase *"thoughts as provenance"* — where synthesized knowledge generates semantic edges whose provenance the thought becomes — **appears genuinely unique**. No other system explicitly frames the act of thinking as evidence generation. Make this explicit here. **Priority: H.**
>
> **Sources — architectural inversion contrast:**
>
> - **RDF-star (being standardized as RDF 1.2).** Embedded triples: `<<:bob :knows :alice>> :source :wikipedia`. ~50% data volume reduction vs classical reification (Ontotext benchmarks, GraphDB 11.2 docs). *Still an annotation mechanism, not a derivation chain system — sharpen this contrast.* `read.pdf` + `read_2.pdf`. **Priority: M.**
> - **Named graphs (Carroll et al. 2005, ACM 1060745.1060835).** Foundational RDF provenance mechanism. W3C Provenance WG (2011) explicitly documented the granularity mismatch: named graphs operate at document level, triple-level requires verbose singleton graphs, derived triples have no natural provenance "home." `read.pdf`. **Priority: M.**
> - **Palantir Foundry.** Tracks complete dataset-level lineage from raw ingestion through all transformations with interactive DAG visualization. **Dataset-level, not fact-level** — sharpen the contrast. `read.pdf`. **Priority: L.**
> - **Google Knowledge Vault (2014) / NELL (CMU).** Per-fact confidence and extraction source tracking, but no complete transformation lineage. `read.pdf`. **Priority: L.**
> - **UaG — Uncertainty-Aware Graph (CIKM 2024).** Conformal prediction in KG-LLM reasoning; uses uncertainty to **guide** reasoning paths. *PDF1: "the closest academic work to RankeDB's provenance as query direction."* `read.pdf`. **Priority: M — elevates the "provenance guides query behavior" claim beyond mere filtering.**
> - **Dagstuhl survey (2024) on uncertainty in KG construction.** Documents how confidence scores propagate through construction pipelines. Facebook's KG removes facts below confidence thresholds. Treats provenance as a filter, not as substrate — sharpen distinction. `read.pdf`. **Priority: L.**


## Under 2.2 Immutability and Accumulation

> **TODO — Reading for §2.2 (immutability as foundational principle):**
>
> - **Helland, P. (2015). "Immutability Changes Everything."** *CIDR 2015.* Already in references. Key quotes to lift: *"accountants don't use erasers"* and *"the truth is the log; the database is a cache of a subset of the log."* **PDF2 explicitly notes: "No subsequent work has explicitly applied Helland's thesis to knowledge graphs, despite its enormous influence on event sourcing and distributed systems."** RankeDB would be the first. `read_2.pdf`. **Priority: H.**
> - **Nelson, T. (1960-present). Project Xanadu.** Specified immutable, add-only content space where documents are lists of pointers to regions in an "ever-growing" store; transclusion maintains *"visible provenance to the source"*; every connection bidirectional. **PDF2: "arguably the direct ancestor of what RankeDB proposes."** Cautionary lesson: Xanadu's refusal to compromise prevented adoption while the simpler Web prevailed. `read_2.pdf`. **Priority: H — currently missing from §6, must add.**
> - **Hickey, R. (2012). Datomic.** Already cited. PDF2 nuance to add: Datomic captures the *temporal* dimension of knowledge (what changed, when) but **not the *epistemic* dimension** (how knowledge was derived, from what evidence, by what process). `read_2.pdf`. **Priority: M.**
> - **Records in Contexts (RiC-O v1.1, May 2025, ICA).** International Council on Archives standard. Describes archival world as *"a graph of interconnected things"*; models `rico:ProvenanceRelation` as first-class OWL relation type. Archival profession's 180-year-old *respect des fonds* principle rendered as a knowledge graph standard — a conceptual ancestor of RankeDB from a completely different intellectual tradition. `read_2.pdf`. **Priority: M — adds archival-theory legitimacy, currently missing.**
> - **Google Always-On Memory Agent (March 2026, open-source).** The explicit opposite of RankeDB. ConsolidateAgent runs every 30 minutes, merging duplicates and dropping information to *"mimic how the human brain processes information during sleep."* No vector DB, no embeddings — the LLM reads, thinks, and writes structured memory into SQLite, making the **LLM the truth arbiter**. RankeDB's motto inversion: *"the graph is the truth, the LLM translates."* `read.pdf`. **Priority: H — cite explicitly as anti-RankeDB in §7.1 contrast.**
> - **XTDB (formerly CruxDB).** Append-only log with native bitemporal support. Rare precedent for add-only temporal database. `read.pdf`. **Priority: L.**
> - **DefraDB / Arweave.** Content-addressable distributed storage with immutability guarantees — check how they handle provenance. `read_2.pdf`. **Priority: L.**


## Under 2.3 Under-Prescription: A Base for Evolution

> **TODO — Reading for §2.3 (reprocessing vs migration — GraphRAG family comparison):**
>
> The GraphRAG landscape is the cleanest contrast point for RankeDB's reprocessing property.
>
> - **Edge et al. (April 2024). "From Local to Global: A Graph RAG Approach to Query-Focused Summarization."** Microsoft Research foundational GraphRAG paper. LLM entity/relation extraction + Leiden community detection + pre-built community summaries. `read.pdf`. **Priority: M.**
> - **DRIFT Search (Microsoft, October 2024).** Combines global and local retrieval with iterative refinement. Reference: microsoft.com/en-us/research/blog/introducing-drift-search. `read.pdf`. **Priority: L.**
> - **LazyGraphRAG (Microsoft, November 2024).** Reduces indexing costs to **0.1% of full GraphRAG** via NLP-based extraction instead of LLM summarization. lianpr.com/en/news/detail/3224. `read.pdf`. **Priority: L — cite as efficiency-trades-history example.**
> - The crucial pattern: **all GraphRAG variants require full reprocessing when the extractor improves.** Contrast explicitly with RankeDB's append semantics.


## Under 3. Architecture

> **TODO — Reading for §2 (three-layer architecture precedents):**
>
> - **Enterprise Knowledge consultancy (2024-2025). "Graph Analytics in the Semantic Layer: An Architectural Framework for Knowledge Intelligence."** Documents a "three-graph architecture": metadata graphs (lineage, ownership) / knowledge graphs (ontology-backed entities) / analytics graphs (pattern detection). *Key distinction: operates three graph types in parallel, whereas RankeDB arranges three layers sequentially.* `read.pdf`. **Priority: M.**
> - **IntuitionLabs (2025) biotech/pharma KG pattern.** Data lake → semantic integration (graph DB) → service layer. Closer to RankeDB's sequential flow. `read.pdf`. **Priority: L.**
> - **Ant Group OpenSPG/KGFabric (VLDB 2024).** Industrial-scale integration of property graph performance with semantic constraints; **98% storage reduction vs Neo4j** via hybrid compression. `read.pdf`. **Priority: L — cite for scale validation.**
> - **SPADE (SRI International).** Provenance auditing system storing derivation chains in Neo4j OR Postgres, abstracting over both through its QuickGrail query language (ACM Queue 3476885). *The closest direct analog to RankeDB's split-store approach at the implementation level (cf. §4), though RankeDB treats the split as pure implementation detail.* `read.pdf`. **Priority: H — must cite in §6, currently missing.**
> - **dbt Semantic Layer, Cube.dev, AtScale.** Analytics semantic layer tradition — abstraction over warehouse data into business metrics. January 2026 **Open Semantic Interchange (OSI)** spec supported by 40+ companies (Snowflake, Salesforce, Databricks). Shares DNA with transformation lineage but at dataset level, not per-fact. `read.pdf`. **Priority: L.**


## Under 6. Related Work

> **TODO — §6 is currently under-cited relative to PDF1 (8 research areas) and PDF2 (5 closest systems + 5-angle gap).
> New subsections to add: Quit Store, Blue Brain Nexus, SPADE, Fluree detail, Xanadu, Helland, RiC-O, PROV-AGENT, Bitemporal KGs (AeonG/BiTRDF), Personal Knowledge Graph community (Balog/Stavanger), JTMS/ATMS/AGM, Senzing, Tools-for-Thought lineage.
> See per-subsection TODOs below.**


## Under 6.1 Temporal Knowledge Graphs: Graphiti/Zep

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


## Under 6.2 Versioned Knowledge Bases: TerminusDB

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


## Under 6.3 Immutable Databases: Datomic and Fluree

> **TODO — Reading for §6.3 (Datomic/Fluree expansion + Helland):**
>
> - **Helland, P. (2015). "Immutability Changes Everything."** CIDR 2015. `cidrdb.org/cidr2015/Papers/CIDR15_Paper16.pdf` + ACM Queue 2884038. **Read in full — this is the theoretical foundation of RankeDB's §2.2.** Core quotes: *"accountants don't use erasers"*, *"the truth is the log; the database is a cache of a subset of the log."* **PDF2 explicitly: "No subsequent work has explicitly applied Helland's thesis to knowledge graphs, despite its enormous influence."** RankeDB is the first. `read_2.pdf`. **Priority: H.**
> - **Nubank case study.** Engineers applied Datomic to microservice dependency graphs and used the phrase *"immutable knowledge databases"* — closest vernacular antecedent for RankeDB's framing. `read_2.pdf`. **Priority: L.**
> - `read_2.pdf` Fluree detail: founded 2017, supports RDF + JSON-LD + SPARQL + SHACL validation. Every update cryptographically chained, enabling time-travel and verifiable data history. **PDF2: "perhaps the closest *production database* to the RankeDB vision."** But Fluree's immutability operates at the transactional ledger level, not at the level of a provenance DAG tracking derivation chains. No separate content-addressable blob store. AI/ML processors not modeled as first-class graph participants. **Priority: M.**
> - Datomic nuance from PDF2: captures the *temporal* dimension (what changed, when) but **not the *epistemic* dimension** (how knowledge was derived, from what evidence, by what process). This is the distinction RankeDB introduces. **Priority: M.**


## Under 6.4 W3C PROV-DM

> **TODO — Reading for §6.4 (PROV-DM expansion):**
>
> - **Sikos & Seneviratne (2020). "Provenance-Aware Knowledge Representation: A Survey of Data Models and Contextualized Knowledge Graphs."** *Data Science and Engineering.* **Already in references. Read in full — this is the canonical survey of every RDF-provenance workaround.** Key finding: *RDF "inherently lacks the mechanism to attach provenance data"* — reviews named graphs, reification, RDF-star, singleton properties, nanopublications, finds none fully satisfactory. `read_2.pdf`. **Priority: H.**
> - `read.pdf` PROV-O adoption data: **OpenCitations tracks over 2 billion citations using PROV-O**; 2025 Nature Scientific Data paper aligned PROV-O with the ISO-standard Basic Formal Ontology (BFO). `read.pdf`. **Priority: M.**
> - **Carroll et al. (2005).** *"Named Graphs, Provenance and Trust"* (ACM 10.1145/1060745.1060835). Foundational paper establishing named graphs for provenance. W3C Provenance Working Group (2011) documented the granularity mismatch. `read.pdf`. **Priority: M.**
> - **RDF-star / PROV-STAR as bolt-on provenance.** PDF2 gap table: "Nanopubs are flat collections; PROV-STAR is a bolt-on." Contrast with RankeDB's substrate approach. `read_2.pdf`. **Priority: M.**
> - **PROV-AGENT (Souza et al., IEEE e-Science 2025, arXiv 2508.02866).** First provenance framework for AI agent workflows; extends W3C PROV with agent-specific metadata. *Operates within traditional workflow orchestration rather than proposing a provenance-first architecture.* `read_2.pdf`. **Priority: M — currently missing from §6, must add.**


## Under 6.5 Nanopublications

> **TODO — Reading for §6.5 (Nanopublications expansion):**
>
> - **Kuhn & Dumontier (2014). "Trusty URIs."** ESWC 2014. Already in references. Read for the content-addressability mechanism (cryptographic hash URIs).
> - `read_2.pdf` detail: over **10 million nanopublications** exist, primarily in life sciences. Each nanopub contains three named RDF graphs: assertion, provenance, publication info. **Priority: M.**
> - **2025 extension: "Nanopublications with Knowledge Provenance"** (International Journal of Digital Libraries, Springer s00799-025-00431-x). Extends with trust networks where multiple agents assign truth values on a 0-1 scale — *parallel to RankeDB's conviction levels, though at scientific publication level rather than personal knowledge.* `read.pdf`. **Priority: M — add to references.**


## Under 6.6 TODO: Additional prior art (currently missing from §6, must add)

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


## Under 6.7 The Identified Gap

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


## Under 7.1 The Context Window Bet

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


## Under 7.2 Toward a CRDT-Compatible Architecture

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

