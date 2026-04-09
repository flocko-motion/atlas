---
citekey: wu2025longmemeval
title: "LongMemEval: Benchmarking Chat Assistants on Long-Term Interactive Memory"
authors: "Wu, Di and Wang, Hongwei and Yu, Wenhan and Zhang, Yuwei and Chang, Kai-Wei and Yu, Dong"
year: 2025
venue: "ICLR 2025"
type: inproceedings
---

# LongMemEval: Benchmarking Chat Assistants on Long-Term Interactive Memory

**Authors:** Di Wu, Hongwei Wang, Wenhan Yu, Yuwei Zhang, Kai-Wei Chang, Dong Yu
**Affiliations:** UCLA, Tencent AI Lab Seattle, UC San Diego
**Year:** 2025
**Venue:** ICLR 2025
**arXiv:** 2410.10813v2

> 500 manually curated questions across five core long-term memory abilities; LongMemEval_S ~115k tokens, LongMemEval_M ~1.5M tokens (500 sessions); long-context LLMs and commercial assistants show 30–60% accuracy drop.

## Online resource

- GitHub: <https://github.com/xiaowu0162/LongMemEval>
- arXiv: <https://arxiv.org/abs/2410.10813v2>

## Why this matters for the RankeDB papers

**Primary evaluation reference for Paper 3**, and a general design-vocabulary reference for Papers 2 and 3 beyond the evaluation. Wu et al. provide: (1) the five-ability decomposition we adopt as design goals; (2) a unified view of memory-augmented chat assistants across three execution stages and four control points that we use as vocabulary; (3) an explicit summary of the three common approaches to LLM long-term memory that Paper 1 §7.1 positions RankeDB *against*; (4) a comparison table across prior benchmarks that justifies using LongMemEval as our single task benchmark; (5) four experimental findings (§5.2–§5.5) that map cleanly onto existing RankeDB design decisions.

**Not the reference for Paper 1 evaluation.** Paper 1 is a data-structure paper and its argument rests on philosophy and comparative architecture, not on empirical benchmarks. LongMemEval is cited in Paper 1 only in §7.1 (for the three-approaches quote) and §7.4 (as the source of the five-ability vocabulary used as a design-rationale lens).

## Notes

### Core contribution

A benchmark for long-term memory in chat assistants. **500 manually curated questions** embedded in freely scalable user-assistant chat histories, with two standard settings:

- **LongMemEval_S** — approximately 115k tokens per problem
- **LongMemEval_M** — 500 sessions, ~1.5 million tokens per problem

Preliminary results: long-context LLMs show **30%–60% accuracy drop** on LongMemEval vs easier settings; commercial products achieve **30%–70% accuracy** even in a much-simpler-than-LongMemEval_S configuration. The benchmark is hard by design.

### The five core memory abilities (adopted verbatim as design goals in Papers 2 and 3)

From §3.2 Benchmark Curation:

1. **Information Extraction (IE).** *"Ability to recall specific information from extensive interactive histories, including the details mentioned by either the user or the assistant."* — Assistant utterances are first-class memory, not just scaffolding.

2. **Multi-Session Reasoning (MR).** *"Ability to synthesize information across multiple history sessions to answer complex questions that involve aggregation and comparison."*

3. **Knowledge Updates (KU).** *"Ability to recognize the changes in the user's personal information and update the knowledge of the user dynamically over time."*

4. **Temporal Reasoning (TR).** *"Awareness of the temporal aspects of user information, including both explicit time mentions and timestamp metadata in the interactions."*

5. **Abstention (ABS).** *"Ability to identify questions seeking unknown information, i.e., information not mentioned by the user in the interaction history, and answer 'I don't know.'"*

### Comparison to prior benchmarks (Wu et al. Table 1)

LongMemEval strictly dominates every earlier long-term memory benchmark on capability coverage and matches or exceeds them on scale:

| Benchmark | Domain | #Sessions | Context depth | IE | MR | KU | TR | ABS | Covered |
|---|---|---|---|---|---|---|---|---|---|
| MSC (Xu 2022a) | Open-domain | 5k | 1k | – | – | – | – | – | 0/5 |
| DuLeMon (Xu 2022b) | Open-domain | 30k | 1k | – | – | – | – | – | 0/5 |
| MemoryBank (Zhong 2024) | Personal | 300 | 5k | ✓ | – | – | – | – | 1/5 |
| PerLTQA (Du 2024) | Personal | 4k | ~1M | ✓ | – | – | – | – | 1/5 |
| LoCoMo (Maharana 2024) | Personal | – | 10k | ✓ | ✓* | – | ✓ | – | 3/5 |
| DialSim (Kim 2024) | TV shows | 1k–2k | 350k | ✓ | ✓ | – | ✓ | – | 3/5 |
| **LongMemEval** | Personal | 50k | 115k / 1.5M | ✓ | ✓ | ✓ | ✓ | ✓ | **5/5** |

\* At most two sessions tested.

**No earlier benchmark covers more than 3/5 abilities.** This is the justification for using LongMemEval as our single task benchmark in Paper 3 §7.2 — a system evaluated on LongMemEval is evaluated on a strict superset of what earlier benchmarks test.

### Seven question types (qualitative test fixtures for Paper 2 §6.1)

Figure 1 of the paper gives examples of seven question types derived from the five abilities:

- **single-session-user** — user fact recall from a single session
- **single-session-assistant** — assistant-produced fact recall (the restaurant the assistant recommended is a fact the user can ask about later)
- **single-session-preference** — latent preference from multiple signals in one session
- **temporal-reasoning** — questions requiring arithmetic on time
- **knowledge-update** — where the most recent version of a superseded fact is the answer
- **multi-session** — aggregation/counting across multiple separate conversations
- **abstention** — the correct answer is "I don't know"

We adopt these as **qualitative extraction test fixtures** for Paper 2 §6.1. Each scenario is self-contained and small enough to render the resulting semantic graph in full.

### The three common approaches Wu et al. identify (Paper 1 §7.1 rejection)

From §2 Related Work / Long-Term Memory Methods:

> "To equip chat assistants with long-term memory capabilities, three major techniques are commonly explored. The first approach involves directly adapting LLMs to process extensive history information as long-context inputs (Beltagy et al., 2020; Kitaev et al., 2020; Fu et al., 2024; An et al., 2024). While this method avoids the need for complex architectures, it is inefficient and susceptible to the 'lost-in-the-middle' phenomenon, where the ability to utilize contextual information weakens as the input length grows (Shi et al., 2023; Liu et al., 2024). A second line of research integrates differentiable memory modules into language models, proposing specialized architectural designs and training strategies to enhance memory capabilities (Weston et al., 2014; Wu et al., 2022; Zhong et al., 2022; Wang et al., 2023). Lastly, several studies approach long-term memory from the perspective of context compression..."

All three commit the memory solution **to the language model itself** — by expanding its context, modifying its architecture, or compressing what it receives. Paper 1 §7.1 quotes this passage verbatim and positions RankeDB as explicitly not taking any of the three paths: the memory lives *outside* the model, as a structured, provenance-addressable database.

### The unified framework (design vocabulary for Paper 3)

Wu et al. decompose memory-augmented chat assistants into:

- **Three execution stages:** indexing, retrieval, reading
- **Four control points:** value, key, query, reading strategy

This vocabulary is adopted in Paper 3 throughout, not just in the evaluation section. The stages and control points give us a consistent way to talk about memory-agent design choices.

### Four experimental findings (§5.2–§5.5) — each maps to an existing RankeDB design decision

**§5.2 — Round granularity beats sessions.** Round-level (single-turn) granularity is optimal; further compression into individual user facts loses information overall but improves multi-session reasoning specifically. *On RankeDB, this tradeoff disappears* — the append-only DAG carries every granularity simultaneously (raw Record at L0, normalization, per-round, per-session summary, extracted fact, L2 projection). The consumer picks the granularity per query. This is the clearest signal that RankeDB's multi-LOD design is aligned with what Wu et al. observed empirically.

**§5.3 — Key expansion with extracted user facts.** Expanding memory keys with extracted user facts adds **+9.4% recall@k** and **+5.4% QA accuracy** over a flat memory-values-as-keys baseline. *This is what RankeDB's L2 already is* — the semantic index is built from extracted facts projected from L1, with per-node/per-edge provenance back to the content.

**§5.4 — Time-aware indexing improves temporal reasoning.** An indexing + query expansion strategy that explicitly associates timestamps with facts gives **+6.8%–11.3% recall** on temporal questions with strong-LLM query expansion. *RankeDB has this at the schema level* — every L2 edge carries `valid_from`/`valid_until`, plus transaction time from the L1 DAG.

**§5.5 — Chain-of-Note and structured data format.** Even with perfect recall, utilization is non-trivial. Applying Chain-of-Note (Yu et al. 2023) and structured data format (Yin et al. 2023) adds **up to +10 absolute points of QA accuracy** across three LLMs. *This motivates Paper 3 §2.3* — the memory agent should return structured entity-and-relation context (which RankeDB provides by default through L2) and should use Chain-of-Note-style reading to trace provenance chains before answering.

### Relation to the RankeDB "multi-session reasoning" bet

RankeDB hypothesizes that **MR is the ability on which the stack has the clearest structural advantage** (Paper 3 §7.3), because sessions are a container in the raw data but not an organizing principle in the data model. Wu et al.'s §5.2 result supports this: they found empirically that the regime where compression into facts *helps* (multi-session reasoning) is exactly the regime where a single-granularity system has to pick — and RankeDB does not have to pick.

### What we used from this paper, and where

- **Paper 1 §7.1** — quote from Related Work on the three approaches RankeDB rejects
- **Paper 1 §7.4** — five abilities as lens for explaining architectural choices ("the architecture keeps each ability *possible*"), citation only, no verbatim definitions
- **Paper 2 §1.1** — five abilities as worker-fleet design goals, verbatim definitions, each translated into a worker requirement
- **Paper 2 §6.1** — seven question types as qualitative extraction test fixtures, with a mapping table to the extraction capability each exercises
- **Paper 3 §1.1** — five abilities as memory-agent design goals, verbatim definitions, each translated into an agent requirement
- **Paper 3 §7** (Evaluation) — LongMemEval adopted as the single task benchmark; Table 1 comparison lifted; §7.3 ability-to-stack mapping; §7.4 design findings as implications

### Open questions this paper leaves us with

- **Reader strategy beyond Chain-of-Note.** §5.5 shows reader strategy matters a lot; this is Paper 3 §2.3 territory. We have not yet committed to a specific reader style for the RankeDB memory agent.
- **How to run the benchmark.** LongMemEval evaluates a full chat-assistant stack. Building the reference reader worker that sits between the benchmark harness and the RankeDB API is a nontrivial amount of Paper 3 implementation work not yet scoped.
- **Abstention calibration.** Wu et al. treat abstention as a binary capability. RankeDB's provenance-based abstention could support a more graduated refusal ("I have weak evidence that X, with provenance chain Y"), which is outside LongMemEval's grading but may be a richer property worth reporting.
- **LongMemEval_M at 1.5M tokens.** Running against the larger setting is operationally heavy; a question for Paper 3 is whether we report both or only _S.
