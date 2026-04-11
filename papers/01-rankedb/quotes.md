"LLMs strip provenance from knowledge. Systematically, architecturally and by design. And in so doing, AI systems are creating a form of knowledge network decay that degrades the knowledge infrastructures that human civilization rely upon." [talisman2026](sources/talisman2026provenance)

"The concept, first formalized by Giorgio Cencetti in 1939, captures the essence of the critical importance of provenance: knowledge does not exist as isolated units. It exists in structured relationships, and those relationships are themselves carriers of meaning. Destroy the relationships and intelligibility is lost." [talisman2026](sources/talisman2026provenance)

"The historian Peter Burke, in *What is the History of Knowledge?* (2016), situates this principle within a broader intellectual context. Burke argues that to qualify as 'knowledge,' items of information must be discovered, analyzed, and systematized—what he calls, *Verwissenschaftlichen*, the shift towards a more scientific approach. Burke posits that knowledge is not raw data. It is information that has been processed through systems of verification, classification, and contextual placement. The mechanisms of that processing—and their history—are themselves a form of knowledge. Burke further argues that even the idea of scientific objectivity—'an attempt to separate knowledge from the knower'—has a history. Provenance is how we preserve the record of that processing. Without it, we collapse knowledge back into unverified assertion." [talisman2026](sources/talisman2026provenance)

"The quest for knowledge rather than mere information is the crux of the study of archives… All the key words applied to archival records—provenance, respect des fonds, context, evolution, inter-relationships, order—imply a sense of understanding, of 'knowledge,' rather than the merely efficient retrieval of names, dates, subjects, or whatever, all devoid of context, that is 'information.'" — Council on Library and Information Resources, via [talisman2026](sources/talisman2026provenance)

"Suzanne Briet, librarian, historian and poet, in her 1951 manifesto *What Is Documentation?*, argued that a document is not simply a text — it is any piece of evidence organized to represent or prove something. An antelope in the wild is not a document. An antelope cataloged in a zoo, with a record of its capture, its species classification, its provenance of origin — that is a document. The act of documentation is what transforms raw existence into knowledge that can be evaluated, transmitted, and trusted. Without the record, there is no evidence. Without evidence, there is no knowledge — only assertion." — via [talisman2026](sources/talisman2026provenance)

"Patrick Wilson, writing on cognitive authority in *Second-Hand Knowledge* (1983), made a complementary point: we accept most of what we know not through direct experience but on the authority of others. The question is never simply what is claimed but who claims it, on what basis and the source of the claim. Wilson argued that the credibility of knowledge relies upon our ability to trace the lineage of information or a claim. And tracing works to their sources requires infrastructure and knowledge." — via [talisman2026](sources/talisman2026provenance)

"Subject headings and classification systems organize knowledge into retrievable, navigable structures so that a researcher can move from a question to a source, from a source to its author and to that author's citations. Every layer of this system is provenance infrastructure — designed to make knowledge findable and evaluable. In library science, provenance is documentation. It is the chain of custody that connects a claim to its source, a source to its author, and an author to the context in which they produced knowledge. This chain is the mechanism by which knowledge becomes trustworthy. Large language models break this chain by design." [talisman2026](sources/talisman2026provenance)

*Note: RankeDB's positioning — rebuild the provenance chain LLMs break, using LLMs as workers, for LLMs as consumers. The tool that severed provenance becomes the tool that restores it, inside an architecture that refuses to let it strip the chain.*

"Science, industry, and society are being revolutionized by radical new capabilities for information sharing, distributed computation, and collaboration offered by the World Wide Web. This revolution promises dramatic benefits but also poses serious risks due to the fluid nature of digital information. One important cross-cutting issue is managing and recording provenance, or metadata about the origin, context, or history of data." — Cheney, Chong, Foster, Seltzer & Vansummeren (OOPSLA 2009), via [talisman2026](sources/talisman2026provenance)

*Note: Provenance was identified as a cross-cutting research priority in computer science two decades before LLMs broke it at scale. The irony: the discipline had the theoretical groundwork ready when the problem arrived, but the architectural integration — provenance as substrate rather than annotation — was never built. RankeDB picks up that thread.*

*Scope note (important — must be stated explicitly in Part 1):*

*RankeDB does not aim to solve global provenance. It targets personal up to small-enterprise scale: individual archives, project teams, small organizations. At this scale, the sources are largely self-trusted — your own email, your own chats, your bank statements, your photos, your team's documents. The trust problem is about preserving the chain from things you already trust, not establishing trust across an adversarial public.*

*Where RankeDB does NOT go: Wikipedia-scale consensus, web-scale retrieval, public scientific record. Those are different problems with different failure modes (adversarial editors, commercial stakes, peer review, citation networks). Trying to solve them all at once is what killed the Semantic Web.*

*Why this matters architecturally: at personal/project scale, the auto-verifiable domain is usually close to the whole domain, and the bounded-verification model works. At global scale the unverifiable case dominates, and a different architecture is needed. RankeDB is tuned for the former and honest about it.*

---

## Design notes for paper 1, Part 1: two principles, one stance

### Postmodern epistemology: claims, not truths

RankeDB captures *what people said*, not *what is*. Every node in the graph is a communicative act by someone, at some time, in some context. "Napoleon was born in 1769" isn't a fact in RankeDB — it's Wikipedia's claim, or your history textbook's claim, or your grandfather's claim. The graph stores the claim, the claimer, the context, and the provenance. It does *not* store "the truth" about Napoleon.

This separates RankeDB from every knowledge graph that came before it. Wikidata, Google Knowledge Vault, DBpedia, Wikipedia — all try to capture *what is true about the world*. They treat statements as converging on ground truth. Disagreement is a bug to resolve. Consensus is the goal.

RankeDB rejects that goal entirely. This is postmodern in a specific sense: knowledge is perspectival, meaning is constructed through communicative acts, and the job of a knowledge system is not to arbitrate between competing claims but to preserve them faithfully with full attribution. The consumer decides which perspective to trust for which purpose.

Consequences:

- **Contradiction is normal, not a bug.** Two nodes can claim opposite things; both stay in the graph; both carry their provenance.
- **Conviction replaces certainty.** A claim is not "true" or "false" — it has a conviction score based on corroborating sources, their authority, and who is asking.
- **The same claim can mean different things** depending on who said it, when, and to whom. Context is preserved, not abstracted away.
- **There is no "ground truth" layer.** Level 0 is the archive of communicative acts, not an archive of how the world is.
- **Ontology emerges per-perspective**, not globally. My understanding of who "Bob" is may differ from yours, and that's fine — the graph holds both.

Philosophical lineage: closer to oral history than to Wikipedia. Closer to phenomenology than to Description Logic. Closer to Ranke's attribution-first history than to Comte's positivism.

### Bounded scope: personal to small-enterprise

What is hard at global scale is tractable at subjective scale. The global problem is not hard because people are stupid — it is hard because there is no shared ground to stand on. At global scale you need consensus across billions, adversarial resistance, formal ontology coordination, jurisdictional compatibility. All of that collapses because the goal itself — a single truth layer — is philosophically incoherent when applied to the whole world.

At subjective scale, the problem evaporates:

- **Consensus is not needed.** I don't need to agree with anyone else about what my mother said in her email. I just need to preserve what she said.
- **Adversarial resistance shrinks.** My archive is mine. The threat model is "don't lose it, don't corrupt it," not "prevent millions of attackers from poisoning consensus."
- **Ontology is bounded.** The entities that matter in my life are finite. Resolving "who is Bob" across 200 conversations is tractable. Resolving "who is Bob" across all humans named Bob on Earth is not.
- **Trust is pre-established.** I already trust my own sources. The question isn't "is this source trustworthy?" but "did I capture what it said faithfully?"
- **Context is preserved by proximity.** All the documents that matter to me are about me, my work, my circle. Context stays intact because the scope stays intact.

Flipping the objection: "RankeDB doesn't scale to Wikipedia" isn't a weakness — it is a feature. Wikipedia-scale is the wrong target. The right target is the scale where the problem is solvable, *and* where the solution is actually useful to an individual. A knowledge graph about my life and work is more valuable to me than Wikipedia, because it is mine, it is complete, and it preserves context that global systems strip.

### The two principles enable each other

Postmodern epistemology and bounded scope are not two separate design decisions — they are one coherent stance.

- You can only afford to be postmodern (store claims, not truths) if you have bounded the scale to one where preserving all claims with attribution is feasible. At global scale you would drown in contradictions.
- You cannot justify bounded scope without postmodern epistemology — if you believed ground truth was the goal, you would have to aim for global, because partial truth is incoherent.

Each enables the other. Together they define what RankeDB is:

**A personal-to-project provenance database for a world where absolute truth is not available, but attributed claims are.**

This is paper 1's thesis in one sentence. The rest of the paper — the three levels, the taxonomy, the invariants, the rebuild guarantee, the under-prescription principle — all fall out as the structural consequences of these two commitments.


*Note: RankeDB aligns with Wilson's cognitive authority framing and Briet's documentation thesis: the database does not care about the antelope itself, only about who said what about the antelope, when, and on what basis. This is a deliberate departure from Berners-Lee's Semantic Web vision, which tried to ground meaning in global concept definitions. RankeDB treats the communicative act as primary.*

