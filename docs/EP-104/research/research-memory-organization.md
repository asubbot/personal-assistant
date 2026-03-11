# Research: Memory organization for AI agents — from raw interaction to reusable knowledge

**Epic:** EP-104 — PersonalAssistant  
**Related:** [REQUIREMENTS.md](../REQUIREMENTS.md) (REQ-006, REQ-007, REQ-018–REQ-020), [system-design.md](../system-design.md), [research.md](../research.md)  
**Source:** Microsoft Research blog, March 2026 — *From raw interaction to reusable knowledge: Rethinking memory for AI agents*  
**Date:** 2026-03-10  
**Purpose:** Ground memory design (including explicit “remember” and day-level storage) in current research and update recommendations.

---

## 1. Source summary

**Article:** [From raw interaction to reusable knowledge: Rethinking memory for AI agents](https://www.microsoft.com/en-us/research/blog/from-raw-interaction-to-reusable-knowledge-rethinking-memory-for-ai-agents/) (Microsoft Research, March 10, 2026).

**Main thesis:** Giving agents *more* raw memory often makes them *less* effective. Long interaction logs are hard to search, mix useful and irrelevant content, and consume context. The fix is not more storage but **transforming raw experience into structured, reusable knowledge** that the agent can retrieve and use at decision time.

**Framework (cognitive science):**

- **Remembering events** — what happened (context).
- **Knowing facts** — propositional knowledge extracted from events.
- **Knowing how** — prescriptive knowledge, reusable skills.

Effective decisions rely on facts and skills; raw events alone are noisy.

**PlugMem (paper’s system):**

- **Structure:** Raw interactions → standardized **knowledge units**: facts (propositional) and reusable skills (prescriptive), organized in a memory graph.
- **Retrieval:** Retrieve **task-aligned** knowledge units (not long passages). Use high-level concepts and intents as routing so the right information surfaces for the current task.
- **Reasoning:** Distill retrieved knowledge into **concise, task-ready guidance** before putting it in the agent’s context — only decision-relevant content enters the context window.

**Evaluation:** Same memory module tested on (1) long multi-turn QA, (2) multi-document fact-finding, (3) web-browsing decisions. PlugMem outperformed both generic retrieval and task-specific memory while **using fewer memory tokens** — i.e. higher utility per token (more decision-relevant information, less context consumption).

**Design principle:** General-purpose memory can outperform task-specific memory when the decisive factor is **surfacing the right knowledge at the right time** (structure + retrieval + reasoning), not specialization per task.

---

## 2. Implications for PersonalAssistant

### 2.1 Current design (as implemented)

| Layer | What is stored | How it is used |
|-------|----------------|----------------|
| **Calendar store** (`memory_dir` / year/month/day) | Markdown files per day. **Nothing writes to it** in the current conversation flow; only `ReadDay` is used. | Injected as “Relevant memory (today)” in the system message — raw text, truncated by `contextMaxLen`. |
| **Vector index** (`pa_vectors.sqlite`) | Each turn: one chunk `"User: …\nAssistant: …"` embedded and added. | Semantic search (top-k) over user query; results injected as “Relevant past context” — again raw chunks, truncated. |

So today we already have:

- **Raw-heavy retrieval:** Vector store returns raw dialogue chunks; calendar store would return raw day file. No explicit “facts” or “skills” as first-class units.
- **No write path for user-driven memory:** No “запомни” / “remember” flow; no structured write into the day file or into a fact layer.
- **Context cap:** `contextMaxLen` limits total injected context but does not optimize for *utility* (decision-relevant knowledge) per token.

Requirements (REQ-019, REQ-020) call for **day-level summaries** produced from LLM logs, tool results, and scheduler events — i.e. transforming raw interaction into summarized content. That aligns with the article’s direction: move from raw logs toward structured/summarized knowledge.

### 2.2 Alignment with the article

- **Structure:** Day-level summaries (and later month/year) are a step from “raw interaction” toward “reusable knowledge” (facts and narrative). Explicit “remember” writes are another: user declares what should be kept.
- **Retrieval:** Today we do semantic search over turn chunks and full-day text. The article suggests task-aligned retrieval and routing (e.g. by intent or high-level concept) so the agent gets the *right* slice of knowledge, not just “recent” or “similar text.”
- **Reasoning / distillation:** We inject raw chunks. The article suggests distilling retrieved content into short, task-ready guidance before putting it in context — reducing tokens and increasing relevance.

We do not need to implement PlugMem itself; we can adopt the **principles**: (1) store and expose knowledge in a more structured way where feasible, (2) prefer compact, decision-relevant context over dumping raw history, (3) support explicit user-driven writes (“запомни”) as a first-class path into memory.

---

## 3. Updated recommendations

### 3.1 Explicit “remember” (запомни) into day memory

**Goal:** When the user explicitly asks to remember something (e.g. “запомни …” / “remember …”), persist that information in **daily memory** (calendar store) so it is available as “Relevant memory (today)” on the same and future days until summarized or superseded.

**Options (from prior discussion, refined):**

1. **Trigger in handler (recommended for MVP)**  
   - Detect intent (e.g. prefix or phrase “запомни”, “remember”, “save this”) in the user message.  
   - Extract the content to store (rest of message or simple heuristics).  
   - Read current day via `memoryStore.ReadDay(ctx, now)`, append a clear, timestamped or bullet line (e.g. “- [time] User asked to remember: …”), call `memoryStore.WriteDay(ctx, now, existing + newLine)`.  
   - Reply with short confirmation (“Записал в дневную память”) or optionally also send to LLM for a natural reply.  
   - **Pros:** Simple, predictable, no LLM output parsing. **Cons:** No normalization of formulation (what the user said is what we store).

2. **LLM-assisted formulation**  
   - User says “запомни …”; we still write to day memory, but we can optionally call the LLM with a narrow prompt: “The user asked to remember the following. In one short sentence, state the fact to store. Reply with only that sentence.” Then store the LLM’s reply instead of raw text.  
   - **Pros:** Cleaner, more fact-like entries. **Cons:** Extra LLM call and latency; needs careful prompting to avoid leaking or inventing.

3. **Tool / structured response**  
   - Add a tool (e.g. `save_to_memory`) or a structured segment in the LLM response (e.g. `SAVE_TO_DAY: …`). Handler parses and calls `WriteDay`.  
   - **Pros:** Agent can normalize and choose what to save. **Cons:** More moving parts; requires tool-calling or output format contract.

**Recommendation:** Start with (1) for MVP: trigger in handler, append to today’s file, confirm. Optionally later add (2) or (3) to improve quality of stored “facts” and align with the article’s emphasis on propositional knowledge.

### 3.2 Day-level content: toward “knowledge” rather than raw dump

- **Explicit “remember” writes:** Treat them as **user-declared facts** for the day. Store them in the same day file but in a consistent format (e.g. a “## Remembered” or “## Facts” section or a list with a clear marker) so future summarization or retrieval can treat them as first-class.
- **Day summaries (REQ-019, REQ-020):** When implementing end-of-day summarization, design the summary to be **reusable**: decisions made, facts established, follow-ups — not a verbatim transcript. That matches the article’s idea of converting raw interaction into structured, compact knowledge.
- **Retrieval:** When we have both “raw” turns in the vector store and day (or summary) files, consider:
  - Preferring day/summary content for “what do we know about X?” and explicit “remember” items.
  - Using vector search for semantic similarity but combining with recency or source type (e.g. “fact from day file” vs “turn excerpt”) so the agent gets the most decision-relevant slice.

### 3.3 Context and utility

- **Cap total context:** Keep a strict cap (as now) on memory + vector context injected into the system message to avoid overwhelming the model.
- **Prefer shorter, higher-value blocks:** Where possible, inject **summaries or fact lists** rather than long raw logs. For “Relevant memory (today)”, if the day file grows large, consider a short “today’s facts” section or a one-paragraph summary produced on read (or by a lightweight pass) instead of dumping the full file.
- **Future:** If we introduce explicit “knowledge units” (e.g. facts with types or tags), retrieval can be intent- or task-aware (e.g. “preferences”, “decisions”, “pending tasks”) to surface the right units and distill them into task-ready guidance, as in the article.

### 3.4 Summary table

| Aspect | Current PA | Article’s direction | Recommended next steps |
|--------|------------|---------------------|-------------------------|
| **Write path** | No user-driven write to day memory | Knowledge units (facts/skills) from interaction | Add “запомни” → append to day file (MVP); optional LLM formulation later |
| **Day content** | Empty in practice (no writer) | Structured, reusable knowledge | Implement day write for “remember”; later day summary as compact knowledge |
| **Retrieval** | Vector top-k + full day text | Task-aligned, intent-aware retrieval | Keep vector + day; consider source type and “fact” vs “turn”; avoid dumping full raw history |
| **Context use** | Truncate by length | Distill to task-ready guidance, maximize utility per token | Keep cap; prefer summaries/fact lists over raw dumps when available |

---

## 4. Implementation options within our architecture

This section outlines concrete ways to implement the article’s approach (structure → retrieval → reasoning) **within the current PersonalAssistant stack**: single Go binary, `memory.Store` (calendar MD), `vector.Store` (pluggable, default SQLite+sqlite-vec), core handler, embedder, LLM provider. No new processes or external services; extensions are additive to existing components.

### 4.1 Structure: from raw interaction to knowledge units

**Article’s idea:** Store facts (propositional) and reusable skills (prescriptive) as first-class units, not only raw dialogue chunks.

| Option | Description | Where it lives | Effort / risk |
|--------|-------------|----------------|---------------|
| **S1 — Minimal** | Keep current: vector store gets one chunk per turn (`User: …\nAssistant: …`). Add only **explicit “remember”**: handler detects trigger, appends one line (or a “## Remembered” block) to today’s day file via existing `memoryStore.WriteDay`. Day file = mixed freeform + remembered items. | `internal/core/handler.go` (trigger + append); `memory.Store` unchanged. | Low. |
| **S2 — Structured day file** | Same as S1, but day file has a **fixed structure**: e.g. `## Remembered` (user-declared facts), optional `## Summary` (filled by end-of-day job per REQ-019/REQ-020). Parser or convention in handler: when reading day for context, prefer these sections over raw tail. | Handler + optional day-summary job. `memory.Store` still ReadDay/WriteDay; content format is convention. | Medium; requires day-summary implementation and consistent formatting. |
| **S3 — Explicit knowledge store** | Introduce **first-class fact units**: e.g. a small struct (date, type fact|skill, text, optional source_id), stored either (a) in the same SQLite as the vector store (new table) with an embedding for the fact text, or (b) as a dedicated section or sidecar in the calendar (e.g. `memory_dir/YYYY/MM/DD_facts.json` or YAML). “Remember” and later day-summary job write into this; vector index can index fact text for semantic search. | New types and either new table in `internal/vector/sqlite` or new file layout in `internal/memory`; handler and summary job both write facts. | Higher; new schema and retrieval path; backward compatibility with existing vector index. |

**Recommendation:** Start with **S1** (remember → append to day). Move to **S2** when implementing day summarization (REQ-019, REQ-020) so that “knowledge” (remembered + summarized) is clearly separated from raw logs. Consider **S3** only if we need queryable fact-level metadata or separate fact vs turn retrieval.

### 4.2 Retrieval: task-aligned and source-aware

**Article’s idea:** Retrieve knowledge units that match the current task; use intent or high-level concepts to route, not only similarity.

| Option | Description | Where it lives | Effort / risk |
|--------|-------------|----------------|---------------|
| **R1 — Current** | Single path: `gatherContext` runs vector `Search(ctx, queryEmbedding, topK)` and `ReadDay(ctx, today)`; concatenate and truncate. No routing, no source type. | `internal/core/handler.go` only. | None (as today). |
| **R2 — Source-aware merge** | Two retrieval paths: (1) vector search (turns + optionally fact chunks if S3); (2) “today’s knowledge” — from day file either full content or only `## Remembered` / `## Summary` if S2. Merge into one context block but **label sources** (e.g. “Relevant memory (today): …” vs “Relevant past context: …”) and optionally **cap each source** (e.g. max 1500 chars from day, 2500 from vector) so facts are not drowned by long turn chunks. | Handler: split `gatherContext` into day read + vector search; apply per-source caps; same system message injection. | Low–medium; no new interfaces, only logic in handler. |
| **R3 — Intent or type routing** | Before retrieval, classify user message (e.g. “preference / decision / follow-up / general”). Use classification to choose: e.g. prefer day file and “remembered” for “what do we know about X?” or “what did I ask to remember?”; prefer vector for “what did we discuss about Y?”. Implementation: simple keyword rules, or one cheap LLM call (“Classify intent: preference|decision|task|general”) and then branch in `gatherContext`. | Handler + optional tiny classifier (rule-based or LLM). | Medium; extra latency if using LLM; need to define intent set and mapping to sources. |

**Recommendation:** Implement **R2** once S1 or S2 is in place: separate day vs vector in context, with per-source limits so that compact “knowledge” (day facts/summary) gets guaranteed space. Add **R3** only if we see that retrieval often returns the wrong type of content (e.g. too many turn chunks, too few facts).

### 4.3 Reasoning: distill before context

**Article’s idea:** Turn retrieved knowledge into short, task-ready guidance before putting it in the agent’s context to maximize utility per token.

| Option | Description | Where it lives | Effort / risk |
|--------|-------------|----------------|---------------|
| **D1 — No distillation** | Current behaviour: inject raw day content and raw vector results, truncate by `contextMaxLen`. | Already in place. | None. |
| **D2 — Structural preference, no LLM** | When building the context string: if day file has structured sections (S2), inject only `## Remembered` and `## Summary` (or first N lines of each), then fill remaining budget with vector results. Or: sort vector results by a simple heuristic (e.g. prefer shorter chunks as more “fact-like”). No extra LLM call. | Handler: in `gatherContext`, parse or slice day content by section; apply order and caps. | Low; depends on S2 format. |
| **D3 — LLM distillation** | After retrieval, call LLM with a dedicated prompt: “User query: … . Retrieved memory excerpts: … . Produce 2–5 bullet points of task-relevant guidance only. No preamble.” Inject only the model’s bullet list into the system message. | New step in handler (or a small helper): retrieve → distill call → then main completion. Requires second LLM call per turn. | Medium–high; latency and cost; need to tune prompt and length. |

**Recommendation:** Start with **D1**. Add **D2** when we have structured day content (S2) so that “knowledge” sections are preferred over raw dump. Consider **D3** only if we have evidence that context is still noisy or low-utility after S2+R2+D2.

### 4.4 Combined adoption paths

| Path | Structure | Retrieval | Reasoning | Use case |
|------|-----------|-----------|-----------|----------|
| **Minimal** | S1 (remember → day) | R1 (current) | D1 (current) | MVP: user can say “запомни” and see it in “Relevant memory (today)”; no other changes. |
| **Standard** | S1 + S2 (structured day + day summary) | R2 (source-aware, caps) | D2 (prefer fact/summary sections) | Aligns with article: knowledge in day file, retrieval favors it, context stays compact. |
| **Full** | S3 (explicit fact store) | R2 + R3 (intent routing) | D2 or D3 (optional distillation) | Maximum alignment with PlugMem-style design; more code and surface area. |

### 4.5 Dependencies and order

- **S1** is independent; can ship first.
- **S2** depends on having a day-summary job (REQ-019, REQ-020) that writes a `## Summary` (or similar); “remember” can create `## Remembered` from day one.
- **R2** is most useful once S1 or S2 exists (so “today’s knowledge” is a distinct slice).
- **D2** depends on S2 (structured sections to prefer).
- **S3, R3, D3** can be added later without blocking the minimal or standard path.

---

## 5. References

- Microsoft Research blog (2026): [From raw interaction to reusable knowledge: Rethinking memory for AI agents](https://www.microsoft.com/en-us/research/blog/from-raw-interaction-to-reusable-knowledge-rethinking-memory-for-ai-agents/).  
- Paper: *PlugMem: A Task-Agnostic Plugin Memory Module for LLM Agents*; code and experiments on GitHub (linked from the blog).  
- EP-104: [REQUIREMENTS.md](../REQUIREMENTS.md) (REQ-006, REQ-007, REQ-018–REQ-020), [research.md](../research.md), [system-design.md](../system-design.md).
