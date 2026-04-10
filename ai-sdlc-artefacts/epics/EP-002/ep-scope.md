# Epic scope — EP-002 Automatic memory summarization

## Introduction

This document is the epic scope for EP-002 (Automatic memory summarization). It builds on the system delivered in EP-001 (PersonalAssistant MVP). It is aligned with project [scope.md](../../scope.md) and [strategy.md](../../strategy.md). Requirements, acceptance criteria, and implementation details are produced in later pipeline stages.

Calendar summarization (day → month → year) remains the **write path** to long-term memory. Answering user questions about **when** something happened uses a **runtime skill plus native structured memory-read tool(s)** (natural language in chat, **ISO dates in tool arguments**), together with existing **semantic vector retrieval**—not a separate rule engine inside the core message handler.

## Epic ID, title, short description

| Field | Content |
|-------|---------|
| **ID** | EP-002 |
| **Status** | NEW |
| **Title** | Automatic memory summarization |
| **Description** | Automatic daily, monthly, and yearly summarization into long-term memory, date-aware vector storage, chunk-type labels in retrieved context, and answering questions about past conversations using a **memory retrieval runtime skill** and **native tool(s)** that read `memory_dir` by ISO date or range—plus vector search where relevant. |

## Glossary

Terms from the project [scope.md](../../scope.md) glossary apply (Long-term memory, Vector store, Scheduler, Core). Epic-specific terms:

| Term | Definition |
|------|-------------|
| **Day summary** | A markdown summary of one calendar day's LLM conversation, produced from that day's LLM log entries; stored under memory_dir in calendar structure (YYYY/MM/DD/summary.md) and in the vector store. |
| **Month summary** | A markdown summary of one calendar month, produced from that month's day summaries; stored under memory_dir (YYYY/MM/summary.md) and in the vector store. |
| **Year summary** | A markdown summary of one calendar year, produced from that year's month summaries; stored under memory_dir (YYYY/summary.md) and in the vector store. |
| **Upsert (day/month/year)** | When writing a summary for a date (day, month, or year) that already has a summary, the existing summary (in memory and in the vector store) is replaced rather than duplicated. |
| **Memory retrieval tool** | A **native** tool implemented in product code and registered on the native tool registry (not an operator-defined catalog tool). It reads long-term memory from `memory_dir` using **structured ISO date arguments only** (e.g. a single calendar day, or `from`/`to` for a bounded range). Enforces maximum range length and input validation; does not accept arbitrary filesystem paths. |
| **Memory retrieval skill** | A runtime skill (EP-013) whose `SKILL.md` defines when to use the memory retrieval tool(s), how to handle ambiguous time phrases in **pa_timezone** (fixed semantics or user clarification), and operational limits. The model maps free-form user wording to tool arguments under this policy. |

## Scope (features/capabilities)

- **Automatic daily summarization:** The system runs day summarization for the previous calendar day automatically (no user or external cron required). Schedule is fixed (e.g. 01:00 in pa_timezone) and is a mandatory part of the memory mechanism, not optional.
- **Automatic month and year summarization:** The system runs month summarization for the previous calendar month and year summarization for the previous calendar year on a schedule (e.g. first day of the new month / new year, or after day summarization has stabilized). Same mandatory, built-in mechanism as for day summarization.
- **Startup catch-up:** On server start, if the previous day has LLM log entries but no day summary exists, the system runs summarization for that day once so that a missed run (e.g. server down at schedule time) is recovered. Catch-up for missed month or year summaries (when day/month summaries exist) is also performed where applicable.
- **Date in vector memory:** Every document stored in the vector store (conversation turns and day/month/year summaries) includes the calendar date (or month/year) in the stored text so that retrieved context is date-aware (e.g. "Date: YYYY-MM-DD\n..." or equivalent). This enables the model to interpret and use dates when answering.
- **Memory retrieval — skill + native tool:** User questions about past calendar periods (specific days, weeks, or other phrasing such as "what happened last week") are handled by (1) a **memory retrieval runtime skill** (selection and policy per EP-013), and (2) one or more **native memory retrieval tools** that return content from `memory_dir` for validated ISO dates or ranges. Vector retrieval remains in use for semantic context; the combination of skill policy, tool I/O, and vector search defines the full retrieval story. **Core does not implement** a parallel rule-based phrase→date resolver or silent injection of day summaries outside this skill+tool contract.
- **Upsert semantics for the same day/month/year:** Re-running summarization for the same calendar day, month, or year overwrites the existing summary in memory and in the vector store (no duplicate entries for that period).
- **Chunk type in context:** When injecting relevant past context from the vector store into the LLM (semantic search results), each chunk is labeled with its type so the model can distinguish direct conversation turns from summaries. Types: turn (single user/assistant exchange), summary:day, summary:month, summary:year. The label is included in the text passed to the model (e.g. `[turn]` or `[summary:day]` prefix) so the assistant can use exact wording from turns and treat summaries as high-level overviews.

## Success criteria

- **Automatic day run:** When the server is running, day summarization for the previous day runs at the configured time (e.g. 01:00 in pa_timezone) without manual or external cron intervention.
- **Automatic month/year run:** Month summarization for the previous month and year summarization for the previous year run on schedule (e.g. first day of month/year in pa_timezone) without manual or external cron intervention.
- **Catch-up:** After a cold start, if yesterday has LLM logs and no day summary, one summarization run for yesterday is performed (e.g. on startup or shortly after). Missed month or year summaries are similarly caught up when applicable.
- **Date in context:** Vector search results and stored summary/turn texts include the date (or month/year) in the text visible to the model (e.g. "Date: YYYY-MM-DD" or equivalent).
- **Calendar-bound questions via skill + native tool:** A user can ask in natural language about past periods (e.g. a specific day, "last week", or equivalent in another language). When day summaries exist for the requested period, the assistant obtains them through **native memory retrieval tool call(s)** with **ISO date arguments**, following the **memory retrieval skill**; answers are consistent with `pa_timezone` semantics defined in the skill. Vector retrieval may supplement semantic context.
- **Upsert:** Running summarization again for the same day, month, or year updates the existing summary (memory and vector), with no duplicate summary documents for that period.
- **Chunk type in context:** Injected context blocks from vector search include a type label (turn | summary:day | summary:month | summary:year) so the model can interpret each chunk appropriately.
- **Tests:** New or changed behaviour is covered by unit and/or integration tests; existing tests continue to pass.

## Manual E2E scenario (operator smoke test)

Use this flow after implementation to validate EP-002 end-to-end in a real environment (Telegram or another configured adapter). Adjust concrete paths, tool ids, and skill package names to match deployment. Record **pass/fail** and notes for each checklist item.

### Preconditions

- Valid config: `memory_dir`, `pa_timezone`, LLM log retention, vector store and embedder enabled as required for production memory.
- **Runtime skills:** Memory retrieval skill package is installed and discoverable per EP-013; `SKILL.md` documents `pa_timezone` semantics for relative phrases (e.g. “last week”).
- **Tools:** Native memory retrieval tool(s) are built in and registered on the **native** registry with the schema expected by the skill (ISO `date` and/or `from` / `to`). Catalog-defined tools are **not** used for this read path.
- Optional: log level or tracing that shows **tool invocations** and arguments (redacted if needed) so the operator can confirm ISO dates without guessing.

### Part A — Write path: conversation → day summary on disk and in vector

1. On calendar day **D**, send one or more distinctive user messages via the live channel (e.g. a unique sentence or codeword **W** that is unlikely to appear elsewhere).
2. Ensure LLM traffic for **D** is logged as required for summarization input.
3. Produce a **day summary** for **D** using the mechanism under test—either wait for the scheduled run after midnight in `pa_timezone`, or run the supported **CLI** `-summarize=YYYY-MM-DD` for **D** if the operator needs to shorten the wait (implementation may expose both).
4. **Verify files:** `memory_dir/YYYY/MM/DD/summary.md` exists for **D** and contains a sensible summary (references to the topic or **W** are a strong signal).
5. **Verify vector (spot check):** Inspect logs, debug output, or a controlled retrieval test so that indexed content for that day includes a **date line** (or equivalent) in the stored text, as required by scope.

**Checklist A:** [ ] Summary file for **D** present and plausible. [ ] Vector payload for day/summary path is date-aware per policy.

### Part B — Startup catch-up (day)

1. With the bot **stopped**, ensure **D** still has LLM log data but **delete** `memory_dir/.../summary.md` for **D** (only for a test environment).
2. Start the process.
3. Confirm that a summarization run for **D** (or “yesterday” if **D** is yesterday in `pa_timezone`) completes and **recreates** the missing `summary.md` without manual CLI.

**Checklist B:** [ ] Catch-up recreated the day summary after restart.

### Part C — Read path: natural language → native memory retrieval tool → answer

1. In the same channel, ask in **natural language** about calendar day **D** (e.g. by weekday name and week context, or the operator’s locale equivalent). Optionally ask about a **range** that includes **D** (e.g. “last week”) if the skill defines that phrase.
2. Confirm in traces/logs that the assistant invoked the **native memory retrieval tool** with **ISO date arguments** matching **D** (or a range that includes **D**), not arbitrary paths.
3. Confirm the user-visible reply uses content consistent with the day summary (e.g. mentions **W** or the same topic).

**Checklist C:** [ ] Tool called with valid ISO args. [ ] Answer grounded in the summary for **D** (or honest “no summary” if file missing).

### Part D — Month / year schedule (abbreviated)

1. If the test calendar allows, use days that already have day summaries in the same month **M** and year **Y**, then wait for or trigger the **month** summarization schedule (or supported manual month run) and confirm `memory_dir/YYYY/MM/summary.md` for **M**.
2. Similarly spot-check **year** rollup for **Y** → `memory_dir/YYYY/summary.md` when month summaries exist.

**Checklist D:** [ ] Month file created when inputs exist. [ ] Year file created when inputs exist. (Mark **N/A** if timeboxed smoke skips long waits.)

### Part E — Upsert (no duplicate summary id)

1. Run day summarization for **D** twice (schedule + manual or two CLI runs).
2. Confirm a **single** logical summary for **D** in memory (file overwritten, not a second path) and no duplicate **summary:day:**\* document in the vector store for **D** (per implementation id scheme).

**Checklist E:** [ ] Second run updates in place; no duplicate day summary in vector index for **D**.

### Part F — Chunk type labels (vector injection)

1. Send a new user message that should trigger **vector retrieval** (a follow-up that relates to earlier content but does not require the memory tool).
2. In logs or a debug capture of the dynamic system tail, confirm retrieved chunks include type labels **`[turn]`**, **`[summary:day]`**, **`[summary:month]`**, or **`[summary:year]`** as applicable.

**Checklist F:** [ ] At least one retrieved chunk shows the expected type prefix.

### Overall pass

EP-002 manual E2E is **PASS** only if checklists **A**, **B**, **C**, **E**, and **F** pass; **D** passes or is marked **N/A** with reason.

## Out of scope / deferred

- **Hybrid retrieval (core rule-based hints + skill + tool):** Adding a small **built-in** phrase→date layer in the core handler *alongside* the skill+tool path (e.g. auto-inject on a subset of phrases while other phrasing uses tools) is **out of scope** for EP-002. Defer to a separate follow-up (**add-plugin** track) if product wants both mechanisms.
- **LLM-based date parsing:** Using the LLM to interpret arbitrary date phrases **inside core** (without the skill+tool contract) remains out of scope. Phrasing is interpreted by the model **only** as part of normal tool-calling behaviour under the memory retrieval skill, not as a second hidden parser in the handler.
- **Prompts from directory (`prompts_dir`):** Loading conversation and summarization (and related) prompt texts from a configured directory with fail-fast validation is deferred. It is a separate configuration and operability concern, not part of the automatic summarization epic; may be addressed in a later epic or increment.

## Design decisions — automatic summarization (runtime)

Product/engineering choices for scheduling, concurrency, and on-disk memory (aligned with implementation planning).

1. **Calendar day in `pa_timezone`:** All automatic summarization (which “previous day/month/year” to process) and the mapping to `memory_dir/YYYY/MM/DD` (and month/year paths) use the **configured `pa_timezone`** (IANA), not an implicit UTC calendar. LLM log selection for a calendar day must use the same definition.  
   **How to reflect timezone in memory files:** The **canonical** encoding of the calendar date is the **directory path** `YYYY/MM/DD` (and `YYYY/MM`, `YYYY`): by invariant, those components are the calendar date in **`pa_timezone`**. The **authoritative zone name** is **`pa_timezone` in config** (single source of truth). Inside `summary.md`, include a **human/LLM-visible date line** consistent with EP-002 “date in vector/text” (e.g. `Date: YYYY-MM-DD` or equivalent) — same date as the path. **Repeating the IANA zone name inside every file is optional:** use only if archives must be self-describing without config; otherwise document the invariant in operator docs. If `pa_timezone` changes later, historical paths are still “dates at write time”; migration is out of scope unless explicitly planned.

2. **Single queue with priority:** Day/month/year summarization and catch-up jobs are submitted to **one** internal job queue; **user-facing LLM work** has **higher priority** than background summarization so interactive latency stays predictable.

3. **Duplicate runs:** A second run for the same calendar day/month/year is **acceptable** when **upsert** yields the same logical outcome (idempotent writes to memory and vector). No extra deduplication layer is required solely to avoid double LLM cost.

4. **Background context and priority:** Summarization runs under a **dedicated context** with an explicit **timeout** (separate from user request contexts). Together with the queue priority, this limits tail latency and resource capture from live traffic.

5. **Persistence ordering and retry:** **Write the markdown file to `memory_dir` first**; then update the vector index. If a step fails, **retry on the next scheduled run** (and catch-up where applicable) rather than tight immediate retry loops, unless a later design adds a narrow exception.

6. **Package boundary:** Automatic scheduling, queueing, and invocation of `summarize` live in a **dedicated Go package** (not embedded ad hoc in the Telegram/handler core), with a clear API wired from `cmd/pa` / process startup.

## Traceability

- **Scope:** This epic extends the Long-term memory and Vector store capabilities from [scope.md](../../scope.md): calendar-based memory with hierarchical summarization (day → month → year) is already in scope; this epic adds automatic execution of day, month, and year summarization, date-aware vector text, chunk-type labeling, and **calendar-bound Q&A via EP-013 runtime skill(s) and native memory retrieval tool(s)**.
- **Strategy:** Aligns with [strategy.md](../../strategy.md) delivery and test strategy: new behaviour is testable (unit/integration); manual checks where necessary (e.g. skill policy, tool bounds, tool-calling edge cases).
- **Dependency:** Builds on EP-001 (PersonalAssistant MVP) and EP-013 (runtime skills): day/month/year summarization CLI and memory/vector stores exist; this epic adds scheduling and catch-up for all three levels, date-in-text for vector payloads, chunk-type labels, **native** memory-read tool(s), and skill-guided reads from `memory_dir` for time-oriented questions.
