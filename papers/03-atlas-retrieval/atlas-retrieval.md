---
title: "Atlas Applications: Chat, Memory Agents, and the Coordination Problem"
author: Florian Noël
date: 2026-04-09
status: sketch
license: CC-BY-4.0
---

# Atlas Applications: Chat, Memory Agents, and the Coordination Problem

## Abstract

*TBD — this paper is a structured idea collection, not yet a coherent argument.*

We explore the application layer built on the Atlas provenance database: a chat interface backed by a memory agent that uses the semantic graph as source of truth, with multiple background agents contributing to context. The paper documents design questions, early experiments, and the coordination problem that emerges when multiple agents compete for conversational attention — a problem whose solution motivates future work on attention economics.

## 1. Introduction

- Papers 1 and 2 deliver a populated provenance database with a semantic graph
- This paper asks: what can you *do* with it?
- Primary application: conversational interface with provenance-grounded memory
- This is an idea sketch, not a finished architecture

## 2. The Memory Agent

### 2.1 Role

- The memory agent is the primary consumer of Atlas
- It is a worker — reads all three levels via the API
- It is the human-facing interface to the knowledge graph
- It translates between natural language (chat) and structured knowledge (graph)

### 2.2 Retrieval Across Levels

- Entry via Level 2: associative search, entity traversal, semantic similarity
- Descent into Level 1: provenance chains, derivation history, competing interpretations
- Access to Level 0: original sources when needed for verification or full context
- The memory agent decides which level to query based on conversational need

### 2.3 Grounded Responses

- **Open question:** Should every claim in a response be traceable to a graph node?
- If yes: the agent operates like a RAG system where the graph is the corpus
- If no: the agent can reason freely but must distinguish graph-grounded from inferred
- **Idea:** Provenance annotations on responses — "I know this because [link to L1 Thought]"

## 3. Background Agents

### 3.1 Proactive Researcher

- Runs continuously in the background
- Monitors conversation topics, searches Atlas for related knowledge
- Pushes findings into conversational context
- **Open question:** visible or invisible to the user?
  - Visible: "I found something related..." — transparent but potentially noisy
  - Invisible: silently enriches context — cleaner UX but less trust
  - Hybrid: silent injection with option to inspect ("why did you know that?")

### 3.2 Memory Filler

- Extracts facts from ongoing chat and writes them back to Atlas
- Chat → Record (L0) → extraction worker → Thoughts (L1) → entities (L2)
- The conversation feeds the graph, the graph feeds the conversation — feedback loop
- **Open question:** real-time or batched? Per-message or per-conversation?

### 3.3 Verification Agent

- When the memory agent cites a graph node, the verifier checks the provenance chain
- Can the claim actually be derived from the cited sources?
- Catches "fake citations" — agent hallucinating a provenance link
- **Open question:** how deep does verification go? L2→L1 sufficient? Or L2→L1→L0?

## 4. The Coordination Problem

### 4.1 Multiple Agents, One Conversation

- Memory agent, researcher, verifier — all want to contribute
- Only one conversational turn at a time
- Who speaks? When? How much context can each inject?

### 4.2 Naive Solutions

- Round-robin: fair but dumb — irrelevant agents waste turns
- Priority queue: static ranking — inflexible, doesn't adapt to conversational dynamics
- Central orchestrator: single point of control — bottleneck, doesn't scale

### 4.3 The Attention Problem

- This is not a scheduling problem — it's an attention allocation problem
- Relevant context is abundant, conversational bandwidth is scarce
- Need a mechanism where agents *compete* for attention based on relevance
- **This is the bridge to Paper 4** — the coordination problem motivates attention economics

## 5. Chat Engine Requirements

### 5.1 Context Management

- Finite context window must be allocated across: conversation history, agent injections, retrieved knowledge
- Context is a scarce resource — who gets how much?
- **Observation:** this infrastructure is shared with Paper 4's stacker system
- Building the chat engine for Paper 3 *is* building the substrate for Paper 4

### 5.2 MCP Integration

- Memory agent as MCP server: exposes Atlas capabilities to any MCP-compatible chat client
- Tools: search graph, traverse provenance, retrieve source, add record
- Decouples memory from chat UI — any client can use Atlas memory

### 5.3 Session Architecture

- Persistent vs. ephemeral sessions
- Conversation history as Records in Atlas (feedback loop)
- Multi-device access to same memory

## 6. Open Questions

*Collected during design, to be resolved through experimentation:*

- How to prevent citation hallucination without making every response slow?
- How much background agent activity is useful vs. noisy?
- Should the user see the provenance graph or only the conversational surface?
- What's the right granularity for memory extraction from chat — per message, per topic, per session?
- How to handle contradictions between chat statements and existing graph knowledge?
- When multiple agents disagree, how does the user experience this?

## 7. Conclusion

- Atlas + memory agent = a chat system with grounded, provenance-tracked memory
- Background agents create a richer but harder-to-coordinate system
- The coordination problem is real and motivates economic mechanisms (Paper 4)
- The chat engine infrastructure built here is the substrate for future stacker experiments

## References

*TBD — will reference Papers 1–2, agent memory literature, MCP specification*

---

*Sketch v0.1 — April 2026*
