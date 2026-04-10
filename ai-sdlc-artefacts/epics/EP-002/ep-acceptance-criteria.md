# EP-002 Automatic memory summarization — Acceptance criteria

**Pipeline:** Stage 5 (Acceptance criteria). Inputs: [ep-scope.md](ep-scope.md), [ep-requirements.md](ep-requirements.md).

**Contents**

- [Introduction](#introduction)
- [Acceptance criteria index](#acceptance-criteria-index)
- [Acceptance criteria](#acceptance-criteria)

---

## Introduction

This document defines epic-level acceptance criteria for **EP-002 Automatic memory summarization**. Criteria are Gherkin-style and trace to [ep-requirements.md](ep-requirements.md). Manual steps reference [ep-scope.md](ep-scope.md) **Manual E2E scenario** where automation is impractical.

---

## Acceptance criteria index

| AC ID | REQ | Summary |
|-------|-----|---------|
| [AC-02.001](#ac-02-001) | [REQ-02.001](ep-requirements.md#automatic-summarization-schedule) | Previous calendar day in pa_timezone summarized at fixed built-in local time (01:00) without external cron |
| [AC-02.002](#ac-02-002) | [REQ-02.002](ep-requirements.md#automatic-summarization-schedule) | Previous calendar month summarized on built-in schedule (first local day 01:00) in pa_timezone |
| [AC-02.003](#ac-02-003) | [REQ-02.003](ep-requirements.md#automatic-summarization-schedule) | Previous calendar year summarized on built-in schedule (first local day 01:00) in pa_timezone |
| [AC-02.004](#ac-02-004) | [REQ-02.004](ep-requirements.md#automatic-summarization-schedule) | Summarization triggers without operator-defined external cron |
| [AC-02.005](#ac-02-005) | [REQ-02.005](ep-requirements.md#startup-catch-up) | Startup catch-up creates missing day summary when logs exist |
| [AC-02.006](#ac-02-006) | [REQ-02.006](ep-requirements.md#startup-catch-up) | Startup catch-up creates missing month summary when day summaries exist |
| [AC-02.007](#ac-02-007) | [REQ-02.007](ep-requirements.md#startup-catch-up) | Startup catch-up creates missing year summary when month summaries exist |
| [AC-02.008](#ac-02-008) | [REQ-02.008](ep-requirements.md#date-and-chunk-labels-in-vector-memory) | Vector-stored turn and summaries include calendar date (or month/year) in text |
| [AC-02.009](#ac-02-009) | [REQ-02.009](ep-requirements.md#date-and-chunk-labels-in-vector-memory) | Retrieved chunks passed to LLM include type label (turn / summary:day / summary:month / summary:year) |
| [AC-02.010](#ac-02-010) | [REQ-02.010](ep-requirements.md#memory-retrieval-skill-and-native-tool) | Native tool returns memory for valid ISO date; rejects path injection |
| [AC-02.011](#ac-02-011) | [REQ-02.010](ep-requirements.md#memory-retrieval-skill-and-native-tool) | **read_memory** rejects oversized range or output with structured error (no silent truncation) |
| [AC-02.012](#ac-02-012) | [REQ-02.011](ep-requirements.md#memory-retrieval-skill-and-native-tool) | Memory retrieval skill package present; policy documents tool use and pa_timezone phrasing |
| [AC-02.013](#ac-02-013) | [REQ-02.012](ep-requirements.md#memory-retrieval-skill-and-native-tool) | Semantic vector retrieval works when memory tool is not invoked |
| [AC-02.014](#ac-02-014) | [REQ-02.013](ep-requirements.md#upsert-semantics) | Re-run summarization replaces same-period summary; no duplicate vector id for that period |
| [AC-02.015](#ac-02-015) | [REQ-02.014](ep-requirements.md#non-functional) | `make check` passes with new tests for EP-002 behaviour |
| [AC-02.016](#ac-02-016) | [REQ-02.015](ep-requirements.md#non-functional) | Pending user LLM work runs before queued summarization job (observable ordering under test) |
| [AC-02.017](#ac-02-017) | [REQ-02.016](ep-requirements.md#non-functional) | After file write succeeds and vector indexing fails, later run indexes same period |

---

## Acceptance criteria

<a id="ac-02-001"></a>**AC-02.001** ([REQ-02.001](ep-requirements.md#automatic-summarization-schedule))

Given the server is running with valid memory, LLM log, and embedder configuration and a fixed **pa_timezone**,  
And the previous calendar day in **pa_timezone** has at least one LLM log entry,  
When the local time in **pa_timezone** reaches the built-in day summarization time (**01:00**),  
Then the system runs day summarization for that previous calendar day and writes `memory_dir/YYYY/MM/DD/summary.md` consistent with **pa_timezone**, without requiring an external cron job.

---

<a id="ac-02-002"></a>**AC-02.002** ([REQ-02.002](ep-requirements.md#automatic-summarization-schedule))

Given the server is running and the previous calendar month in **pa_timezone** has at least one day summary,  
When the built-in month summarization schedule fires (first day of the new month in **pa_timezone** at **01:00** local),  
Then the system runs month summarization for the previous month and writes `memory_dir/YYYY/MM/summary.md` for that month.

---

<a id="ac-02-003"></a>**AC-02.003** ([REQ-02.003](ep-requirements.md#automatic-summarization-schedule))

Given the server is running and the previous calendar year in **pa_timezone** has at least one month summary,  
When the built-in year summarization schedule fires (first day of the new year in **pa_timezone** at **01:00** local),  
Then the system runs year summarization for the previous year and writes `memory_dir/YYYY/summary.md` for that year.

---

<a id="ac-02-004"></a>**AC-02.004** ([REQ-02.004](ep-requirements.md#automatic-summarization-schedule))

Given **paths.memory_dir**, **paths.llm_log_dir**, embedding, and the vector index are available so the automatic summarization worker runs,  
When the operator has not configured an external cron entry solely to trigger summarization,  
Then day, month, and year summarization still run per built-in schedules (fixed in product code, not JSON).

---

<a id="ac-02-005"></a>**AC-02.005** ([REQ-02.005](ep-requirements.md#startup-catch-up))

Given the previous calendar day in **pa_timezone** has LLM log entries and no `summary.md` exists for that day,  
When the server process starts,  
Then the system enqueues or runs day catch-up with **higher priority than timer-fired summarization jobs** so the missing `summary.md` is produced **before** the first scheduled day tick for the new uptime (or runs synchronously during startup before that tick can fire).

---

<a id="ac-02-006"></a>**AC-02.006** ([REQ-02.006](ep-requirements.md#startup-catch-up))

Given a previous calendar month in **pa_timezone** has at least one day summary and no month `summary.md` exists for that month,  
When the server process starts,  
Then the system runs month summarization for that month once, producing the missing month summary file.

---

<a id="ac-02-007"></a>**AC-02.007** ([REQ-02.007](ep-requirements.md#startup-catch-up))

Given a previous calendar year in **pa_timezone** has at least one month summary and no year `summary.md` exists for that year,  
When the server process starts,  
Then the system runs year summarization for that year once, producing the missing year summary file.

---

<a id="ac-02-008"></a>**AC-02.008** ([REQ-02.008](ep-requirements.md#date-and-chunk-labels-in-vector-memory))

Given conversation turns and summaries are indexed into the vector store,  
When a document is stored for a turn or for a day, month, or year summary,  
Then the stored text includes an explicit calendar date or month/year line (e.g. `Date: YYYY-MM-DD` or equivalent for the period).

---

<a id="ac-02-009"></a>**AC-02.009** ([REQ-02.009](ep-requirements.md#date-and-chunk-labels-in-vector-memory))

Given vector search returns chunks for injection into the LLM context,  
When the system builds the dynamic context from those results,  
Then each chunk’s text includes a visible type label from the set **turn**, **summary:day**, **summary:month**, **summary:year**.

---

<a id="ac-02-010"></a>**AC-02.010** ([REQ-02.010](ep-requirements.md#memory-retrieval-skill-and-native-tool))

Given a day summary file exists at the path derived from a valid ISO calendar date under **memory_dir**,  
When the LLM invokes the native **read_memory** tool with that ISO date in a request where tool-calling is enabled (same trust boundary as other native tools sent to the model; **read_memory** is registered whenever **memory_dir** is available),  
Then the tool returns the summary content and does not accept arbitrary filesystem paths outside the derived memory layout.

---

<a id="ac-02-011"></a>**AC-02.011** ([REQ-02.010](ep-requirements.md#memory-retrieval-skill-and-native-tool))

Given **read_memory** supports a **from**–**to** range,  
When the requested range exceeds the configured maximum span or output size,  
Then the tool **rejects** the call with a structured error and does not read unbounded directories or return silently truncated range content.

---

<a id="ac-02-012"></a>**AC-02.012** ([REQ-02.011](ep-requirements.md#memory-retrieval-skill-and-native-tool))

Given runtime skills are configured per EP-013 (`skills_dir`, `vec_skills` index build, tool union per design),  
When the memory retrieval skill package is installed,  
Then its `SKILL.md` lists **read_memory** under `tools`, describes when to call it, and documents how relative calendar phrases relate to **pa_timezone**, and EP-013 native tool validation accepts **read_memory** on the allowlist.

---

<a id="ac-02-013"></a>**AC-02.013** ([REQ-02.012](ep-requirements.md#memory-retrieval-skill-and-native-tool))

Given vector search is enabled and the user sends a message that does not trigger the memory retrieval tool,  
When the handler performs semantic retrieval,  
Then relevant chunks are still supplied from the vector store in the usual way.

---

<a id="ac-02-014"></a>**AC-02.014** ([REQ-02.013](ep-requirements.md#upsert-semantics))

Given a day summary already exists in memory and vector for calendar day **D**,  
When day summarization runs again for **D**,  
Then the memory file is overwritten and the vector store holds a single logical summary document for **D** (same stable id scheme), with no second summary document for the same period.

---

<a id="ac-02-015"></a>**AC-02.015** ([REQ-02.014](ep-requirements.md#non-functional))

Given the EP-002 implementation is merged,  
When `make check` runs in the repository,  
Then all tests pass including new or updated unit and integration tests covering EP-002 behaviour.

---

<a id="ac-02-016"></a>**AC-02.016** ([REQ-02.015](ep-requirements.md#non-functional))

Given a user message and a pending summarization job are both queued in the **memoryjob** priority queue where **lower numeric priority runs first** (interactive LLM work = **0**, background summarization = **larger integer**, e.g. 10),  
When the system dequeues work,  
Then the interactive path is started before the summarization job (verified by test harness dequeue order or timestamps).

---

<a id="ac-02-017"></a>**AC-02.017** ([REQ-02.016](ep-requirements.md#non-functional))

Given day summarization wrote `summary.md` for day **D** and vector indexing then failed,  
When the next **vector reconciliation** pass runs on **process startup** or the defined **background reconciliation cycle**,  
Then the system embeds and upserts the vector row for **D** from the existing file without a new summarization LLM call, and semantic search retrieves that summary without manual operator repair.
