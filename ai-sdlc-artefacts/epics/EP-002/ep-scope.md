# Epic scope — EP-002 Automatic memory summarization

## Introduction

This document is the epic scope for EP-002 (Automatic memory summarization). It builds on the system delivered in EP-001 (PersonalAssistant MVP). It is aligned with project [scope.md](../../scope.md) and [strategy.md](../../strategy.md). Requirements, acceptance criteria, and implementation details are produced in later pipeline stages.

## Epic ID, title, short description

| Field | Content |
|-------|---------|
| **ID** | EP-002 |
| **Status** | NEW |
| **Title** | Automatic memory summarization |
| **Description** | Automatic daily, monthly, and yearly summarization of conversations into long-term memory, date-aware vector storage, and the ability to answer user questions about past conversations by date (e.g. "What did we talk about last Thursday?"). |

## Glossary

Terms from the project [scope.md](../../scope.md) glossary apply (Long-term memory, Vector store, Scheduler, Core). Epic-specific terms:

| Term | Definition |
|------|-------------|
| **Day summary** | A markdown summary of one calendar day's LLM conversation, produced from that day's LLM log entries; stored under memory_dir in calendar structure (YYYY/MM/DD/summary.md) and in the vector store. |
| **Month summary** | A markdown summary of one calendar month, produced from that month's day summaries; stored under memory_dir (YYYY/MM/summary.md) and in the vector store. |
| **Year summary** | A markdown summary of one calendar year, produced from that year's month summaries; stored under memory_dir (YYYY/summary.md) and in the vector store. |
| **Date resolution** | The process of interpreting a user phrase (e.g. "last Thursday", "в прошлый четверг") into a concrete calendar date (YYYY-MM-DD) using a reference date and timezone (pa_timezone). |
| **Upsert (day/month/year)** | When writing a summary for a date (day, month, or year) that already has a summary, the existing summary (in memory and in the vector store) is replaced rather than duplicated. |
| **Prompts directory** | A config path (prompts_dir) to a directory containing prompt template files. All prompt texts used by the core (conversation system prompts) and by summarization (day/month/year system and user-prefix texts) are loaded from these files. No prompts are hardcoded; all parameters are explicit. |

## Scope (features/capabilities)

- **Automatic daily summarization:** The system runs day summarization for the previous calendar day automatically (no user or external cron required). Schedule is fixed (e.g. 01:00 in pa_timezone) and is a mandatory part of the memory mechanism, not optional.
- **Automatic month and year summarization:** The system runs month summarization for the previous calendar month and year summarization for the previous calendar year on a schedule (e.g. first day of the new month / new year, or after day summarization has stabilized). Same mandatory, built-in mechanism as for day summarization.
- **Startup catch-up:** On server start, if the previous day has LLM log entries but no day summary exists, the system runs summarization for that day once so that a missed run (e.g. server down at schedule time) is recovered. Catch-up for missed month or year summaries (when day/month summaries exist) is also performed where applicable.
- **Date in vector memory:** Every document stored in the vector store (conversation turns and day/month/year summaries) includes the calendar date (or month/year) in the stored text so that retrieved context is date-aware (e.g. "Date: YYYY-MM-DD\n..." or equivalent). This enables the model to interpret and use dates when answering.
- **Date-based context injection:** When the user message indicates a specific date (e.g. "what we talked about last Thursday"), the system resolves that phrase to a calendar date and, when a day summary exists for that date, injects that summary explicitly into the LLM context (e.g. "Conversation on YYYY-MM-DD (Thursday): ...") so the assistant can answer date-bound questions reliably.
- **Date resolution (rule-based):** A small set of relative-date phrases (e.g. yesterday, last Monday … Sunday, last week, and optionally "N March" style) is resolved to YYYY-MM-DD using the current date and pa_timezone, without calling the LLM for date parsing in the first version.
- **Upsert semantics for the same day/month/year:** Re-running summarization for the same calendar day, month, or year overwrites the existing summary in memory and in the vector store (no duplicate entries for that period).
- **Chunk type in context:** When injecting relevant past context from the vector store into the LLM (semantic search results), each chunk is labeled with its type so the model can distinguish direct conversation turns from summaries. Types: turn (single user/assistant exchange), summary:day, summary:month, summary:year. The label is included in the text passed to the model (e.g. `[turn]` or `[summary:day]` prefix) so the assistant can use exact wording from turns and treat summaries as high-level overviews.
- **Prompts from directory:** Configuration includes a required path (prompts_dir) to a directory containing named prompt template files. All prompt texts used by the assistant (conversation system prompt with/without context block, context block header) and by summarization (day, month, year: system prompts and user-message prefix for day) are loaded from these files at startup. The set of required file names is fixed and documented. If prompts_dir is missing in config or any required file is missing, the application fails at startup (fail fast). No fallback to hardcoded prompts; all parameters are explicit so that behaviour is testable and consistent.

## Success criteria

- **Automatic day run:** When the server is running, day summarization for the previous day runs at the configured time (e.g. 01:00 in pa_timezone) without manual or external cron intervention.
- **Automatic month/year run:** Month summarization for the previous month and year summarization for the previous year run on schedule (e.g. first day of month/year in pa_timezone) without manual or external cron intervention.
- **Catch-up:** After a cold start, if yesterday has LLM logs and no day summary, one summarization run for yesterday is performed (e.g. on startup or shortly after). Missed month or year summaries are similarly caught up when applicable.
- **Date in context:** Vector search results and injected day/month/year summaries include the date (or month/year) in the text visible to the model (e.g. "Date: YYYY-MM-DD" or equivalent).
- **Date-based questions:** A user can ask in natural language about a past day (e.g. "What did we discuss last Thursday?" or "Напомни, о чём мы разговаривали в четверг на прошлой неделе") and receive an answer based on that day's summary when it exists.
- **Upsert:** Running summarization again for the same day, month, or year updates the existing summary (memory and vector), with no duplicate summary documents for that period.
- **Chunk type in context:** Injected context blocks from vector search include a type label (turn | summary:day | summary:month | summary:year) so the model can interpret each chunk appropriately.
- **Prompts from directory:** The application starts only when prompts_dir is set and all required prompt files exist in that directory; otherwise it exits with a clear error. Conversation and summarization use only the text loaded from these files.
- **Tests:** New or changed behaviour is covered by unit and/or integration tests; existing tests continue to pass.

## Out of scope / deferred

- **LLM-based date parsing:** Using the LLM to interpret arbitrary date phrases is out of scope for this epic; only rule-based resolution is in scope. Broader natural-language date parsing can be considered later.

## Traceability

- **Scope:** This epic extends the Long-term memory and Vector store capabilities from [scope.md](../../scope.md): calendar-based memory with hierarchical summarization (day → month → year) is already in scope; this epic adds automatic execution of day, month, and year summarization, date-aware retrieval, and configurable prompts from a directory (prompts_dir, fail fast if missing).
- **Strategy:** Aligns with [strategy.md](../../strategy.md) delivery and test strategy: new behaviour is testable (unit/integration); manual checks only where necessary (e.g. date-resolution edge cases).
- **Dependency:** Builds on EP-001 (PersonalAssistant MVP): day/month/year summarization CLI and memory/vector stores exist; this epic adds scheduling and catch-up for all three levels, date-in-text, date resolution, and date-based context injection.
