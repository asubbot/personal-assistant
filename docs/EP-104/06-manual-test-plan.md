# EP-104: Manual test plan

**Purpose:** Manual test scenarios for acceptance criteria verified by human review or manual execution with log inspection; companion to test strategy (Manual level).  
**Pipeline:** [PIPELINE.SPEC.md](PIPELINE.SPEC.md)  
**Previous:** [06-test-strategy.md](06-test-strategy.md)  
**Next:** [08-08-user-stories.md](08-08-user-stories.md)  
**Related:** [10-acceptance-criteria.md](10-acceptance-criteria.md), [06-current-coverage.md](06-current-coverage.md)

This document lists manual test scenarios for acceptance criteria that are verified by human review (architecture, documentation) or by manual execution with log inspection. Check off each step when done.

---

## [AC-013](10-acceptance-criteria.md#ac-013-us-07), [AC-014](10-acceptance-criteria.md#ac-014-us-07) ([US-07](08-user-stories.md#us-07--semantic-memory)) — Vector memory: indexing and search

**Goal:** Confirm that the assistant indexes conversation turns into the vector store and uses semantic search to inject "Relevant past context" into the LLM request for later messages.

- [ ] **Precondition:** Bot is configured with embedding and vector store (e.g. `config/config.json` has `embedding` block, vector DB path `./data/pa_vectors.sqlite`). Start the bot with **`PA_LOG_LEVEL=debug`** so that full LLM request (including context block) is logged.
- [ ] **Step 1:** Send a first message that gives a specific fact (e.g. "Запомни: мой любимый цвет — синий" or "My project deadline is March 15").
- [ ] **Step 2:** Send a follow-up that should retrieve that fact via semantic search (e.g. "Какой у меня любимый цвет?" or "When is my project deadline?"). The assistant should be able to answer using the stored context.
- [ ] **Step 3:** In the logs for the second (or a later) request, at DEBUG level, find the `llm request` entry for the **system** message (`role=system`). The `content` field must contain the substring **`Relevant past context:`** followed by one or more lines of context (or **`Relevant memory (today):`** if today's memory is used). This confirms that vector search (and/or daily memory) ran and that the built context is passed to the LLM.
- [ ] **Step 4 (optional):** Confirm that the assistant's reply uses the remembered fact (e.g. "синий" or "March 15"). If the reply is correct and the logs show the context block, vector memory is working end-to-end.

**Expected:** Logs at DEBUG show the system message containing `Relevant past context:` (and/or `Relevant memory (today):`); the assistant's answers reflect previously shared information when relevant.

### Example logs (DEBUG)

With `PA_LOG_LEVEL=debug`, the handler logs each LLM request message. The system message includes the context block built by `gatherContext` (today's memory + vector search results). Example snippet:

```
level=DEBUG msg="llm request" index=0 role=system content_len=842 content="You are a helpful assistant. Reply concisely. You have access to relevant past context and memory below; use it to personalize replies and to remember what the user has told you.\n\nUse the following context if relevant to the user's message.\n\n---\n\nRelevant past context:\n- User said: мой любимый цвет — синий\n- Assistant replied: Запомнила, твой любимый цвет — синий."
level=DEBUG msg="llm request" index=1 role=user content_len=32 content="Какой у меня любимый цвет?"
level=DEBUG msg="llm response" content="Твой любимый цвет — синий." content_len=28 ...
```

If no vector results are found for the query, the system message may contain only the fixed prompt and no `Relevant past context:` block (or only `Relevant memory (today):` when there is same-day memory). After at least one prior turn has been indexed, a related question should produce `Relevant past context:` in the next request.

---

## [AC-025](10-acceptance-criteria.md#ac-025-us-14) ([US-14](08-user-stories.md#us-14--architecture-boundaries)) — Module boundaries and separation

**Criterion:** Given the codebase, when an architect or developer reviews the module boundaries, then ingestion adapters (e.g. Telegram), core, memory store, vector index, LLM abstraction, scheduler, and tools are clearly separated so that replacing or extending one part does not require a full redesign.

- [ ] **Precondition:** Codebase is at a revision to be reviewed (branch or tag).
- [ ] **Step 1:** Identify the ingestion adapter layer (e.g. Telegram). Confirm it only depends on core interface, not on memory/LLM internals.
- [ ] **Step 2:** Identify the core (conversation flow, handler). Confirm it depends on abstractions (LLM, memory, vector, tools) not concrete implementations where possible.
- [ ] **Step 3:** Identify memory store and vector index. Confirm they are separate from core and from each other; replacing one does not force redesign of the other.
- [ ] **Step 4:** Identify LLM abstraction (provider interface). Confirm core uses the abstraction; swapping provider does not require core changes.
- [ ] **Step 5:** Identify scheduler and tools (if present). Confirm they are clearly separated from core and from adapters.
- [ ] **Step 6:** Document any violations (e.g. direct dependency that would require a full redesign to replace). If none, [AC-025](10-acceptance-criteria.md#ac-025-us-14) is satisfied.

**Expected:** Module boundaries are clear; no unjustified coupling that would require full redesign to replace or extend one part.

---

## [AC-027](10-acceptance-criteria.md#ac-027-us-15) ([US-15](08-user-stories.md#us-15--version-control-git)) — Versioned state: tracked paths documented

**Criterion:** Given the versioned state feature is implemented or in design, when the operator or developer consults the documentation, then the exact set of tracked paths is documented or explicitly marked TBD until research is done.

- [ ] **Precondition:** Locate the documentation that describes versioned state (e.g. README, deployment doc, or design doc).
- [ ] **Step 1:** Find the section or page that describes which paths/directories are tracked (e.g. config, memory files, other artifacts).
- [ ] **Step 2:** If the feature is implemented: confirm the list of tracked paths is explicit and matches the implementation (no vague “and other files” without listing).
- [ ] **Step 3:** If the feature is not yet implemented or in design: confirm the doc states either the planned tracked paths or “TBD” (or equivalent) until research is done.
- [ ] **Step 4:** If documentation is missing or unclear, record gaps and treat [AC-027](10-acceptance-criteria.md#ac-027-us-15) as not satisfied until updated.

**Expected:** The exact set of tracked paths is documented, or explicitly marked TBD; no undocumented or vague tracking scope.

---

## [AC-032](10-acceptance-criteria.md#ac-032-us-18) ([US-18](08-user-stories.md#us-18--verify-node-availability)) — Verify node availability via CLI

**Criterion:** Given the application is invoked with the designated parameter to verify node availability, when it runs, it loads config, connects to each node over SSH, runs one allowlisted command per node, reports success or failure, and exits without starting the bot.

- [ ] **Precondition:** Config has at least one node with valid host, dedicated user, key path, and allowlist file; allowlist includes a safe command (e.g. `uptime`, `echo ok`).
- [ ] **Step 1:** Run the binary with the verify parameter (e.g. `go run ./cmd/pa -config ./config/config.json -verify-nodes` or as documented).
- [ ] **Step 2:** Confirm output lists each configured node and reports OK or FAIL; on success, probe command output (e.g. uptime) may be shown.
- [ ] **Step 3:** Confirm the process exits (does not start Telegram polling or webhook).
- [ ] **Step 4 (optional):** Intentionally break one node (e.g. wrong key or unreachable host), run again; confirm at least one FAIL and non-zero exit code.

**Expected:** Verify run completes without starting the bot; each node is reported as OK or FAIL; exit code 0 only when all nodes succeed.
