# EP-016 — Acceptance criteria

This document defines testable acceptance criteria for EP-016 (manual `notes.md`, **write_memory**, **read_memory** extension, vector split, split-query retrieval, event-aligned turn dates, dedup). Each AC traces to [ep-requirements.md](ep-requirements.md).

**Append line format (normative for REQ-16.003 and tooling):** each **write_memory** append is one line:  
`- <UTC_ISO8601_Z> [<kind>] <text>`  
where `<kind>` is `fact`, `guideline`, `preference`, or `custom:<label>`; `<text>` is single-line or follows a documented multiline block terminator in system design if extended.

---

## Introduction

EP-016 acceptance is demonstrated by automated tests, `./bin/validate EP-016`, and `make check`, plus structured scenarios below. Manual E2E may supplement Telegram paths.

---

## Acceptance criteria index

| AC ID | REQ (trace) | Summary |
|-------|-------------|---------|
| [AC-16.001](#ac-16-001) | REQ-16.001 | notes.md path correctness |
| [AC-16.002](#ac-16-002) | REQ-16.002 | Summarization leaves notes.md alone |
| [AC-16.003](#ac-16-003) | REQ-16.003 | Append format includes UTC time and kind |
| [AC-16.004](#ac-16-004) | REQ-16.004, REQ-16.007 | write_memory creates append under memory_dir |
| [AC-16.005](#ac-16-005) | REQ-16.005 | Default date is today pa_timezone |
| [AC-16.006](#ac-16-006) | REQ-16.006 | vec_notes row after write_memory |
| [AC-16.007](#ac-16-007) | REQ-16.008 | read_memory returns Summary and Notes headings |
| [AC-16.008](#ac-16-008) | REQ-16.009 | read_memory respects byte cap |
| [AC-16.009](#ac-16-009) | REQ-16.010–REQ-16.012 | Three vector tables used for new writes |
| [AC-16.010](#ac-16-010) | REQ-16.013 | Legacy vec_items summary search excludes turn ids |
| [AC-16.011](#ac-16-011) | REQ-16.014, REQ-16.015 | Merge order and per-class similarity order |
| [AC-16.012](#ac-16-012) | REQ-16.016, REQ-16.017 | Turn chunk Date matches event-aligned rule |
| [AC-16.013](#ac-16-013) | REQ-16.018, REQ-16.019 | Turn upsert idempotent |
| [AC-16.014](#ac-16-014) | REQ-16.020 | Startup log mentions migration awareness |
| [AC-16.015](#ac-16-015) | REQ-16.022 | Injected chunks carry correct type labels |
| [AC-16.016](#ac-16-016) | REQ-16.023 | write_memory rejects path escape |
| [AC-16.017](#ac-16-017) | REQ-16.024 | Oversized append rejected |
| [AC-16.018](#ac-16-018) | REQ-16.026–REQ-16.028 | make check and validate EP-016 pass |

---

## Acceptance criteria

<a id="ac-16-001"></a>**AC-16.001** (Trace: [REQ-16.001](ep-requirements.md#req-16-001))  
Given a configured **memory_dir** and a calendar day *D* in **pa_timezone**  
When **write_memory** targets *D*  
Then the tool writes only under `…/YYYY/MM/DD/notes.md` matching *D*’s calendar components.

<a id="ac-16-002"></a>**AC-16.002** (Trace: [REQ-16.002](ep-requirements.md#req-16-002))  
Given an existing **day notes file** with byte content *N*  
When automatic day summarization runs for that calendar day  
Then the **day notes file** still has byte content *N* unchanged (summarization writes only **day summary file** for that day).

<a id="ac-16-003"></a>**AC-16.003** (Trace: [REQ-16.003](ep-requirements.md#req-16-003))  
Given a valid **write_memory** call with kind `fact`  
When the append completes  
Then the new line in **day notes file** matches the normative append line format at the top of this document.

<a id="ac-16-004"></a>**AC-16.004** (Trace: [REQ-16.004](ep-requirements.md#req-16-004), [REQ-16.007](ep-requirements.md#req-16-007))  
Given **memory_dir** on a writable test root  
When **write_memory** runs with valid `text` and ISO `date`  
Then `notes.md` exists for that day and ends with the appended line  
And no path outside **memory_dir** is created.

<a id="ac-16-005"></a>**AC-16.005** (Trace: [REQ-16.005](ep-requirements.md#req-16-005))  
Given a test clock fixed to 23:59 local on **pa_timezone** date *D*  
When **write_memory** runs with `text` only  
Then the append targets *D* (not the next local day).

<a id="ac-16-006"></a>**AC-16.006** (Trace: [REQ-16.006](ep-requirements.md#req-16-006))  
Given embedder and **vec_notes** configured  
When **write_memory** succeeds  
Then **vec_notes** contains a row whose **document id** matches the design for that append  
And search against **vec_notes** returns the new text in the top results for a paraphrased query.

<a id="ac-16-007"></a>**AC-16.007** (Trace: [REQ-16.008](ep-requirements.md#req-16-008))  
Given a day with both **day summary file** and **day notes file**  
When **read_memory** queries that single ISO date  
Then the tool output contains a `## Summary` section (or the exact heading chosen in implementation, recorded in tests) with **day summary file** body  
And a `## Notes` section with **day notes file** body.

<a id="ac-16-008"></a>**AC-16.008** (Trace: [REQ-16.009](ep-requirements.md#req-16-009))  
Given **read_memory** `max_output_bytes` set to a small positive value  
When the combined summary and notes for a range exceed that value  
Then **read_memory** returns a structured error without partial silent truncation beyond the documented behaviour.

<a id="ac-16-009"></a>**AC-16.009** (Trace: [REQ-16.010](ep-requirements.md#req-16-010)–[REQ-16.012](ep-requirements.md#req-16-012))  
Given a fresh vector database after migration wiring  
When summarization indexes a day and the core indexes a turn and **write_memory** indexes a note  
Then rows appear in **vec_summaries**, **vec_turns**, and **vec_notes** respectively  
And no new turn row appears in **vec_summaries**.

<a id="ac-16-010"></a>**AC-16.010** (Trace: [REQ-16.013](ep-requirements.md#req-16-013))  
Given a **legacy mixed table** fixture containing both `summary:day:*` ids and numeric turn ids  
When the summary-only search path runs for rollup retrieval  
Then no result has an **document id** outside the `summary:*` prefix set.

<a id="ac-16-011"></a>**AC-16.011** (Trace: [REQ-16.014](ep-requirements.md#req-16-014), [REQ-16.015](ep-requirements.md#req-16-015))  
Given controlled embedding fixtures that return fixed ordering per table  
When retrieval merge runs  
Then concatenated chunk order is all **notes** hits in similarity order, then **summary** hits, then **turn** hits.

<a id="ac-16-012"></a>**AC-16.012** (Trace: [REQ-16.016](ep-requirements.md#req-16-016), [REQ-16.017](ep-requirements.md#req-16-017))  
Given an inbound message with Unix timestamp mapping to calendar *D* in **pa_timezone**  
When the core indexes a **conversation turn**  
Then the stored vector text’s first `Date:` line equals *D* in `YYYY-MM-DD` form.

<a id="ac-16-013"></a>**AC-16.013** (Trace: [REQ-16.018](ep-requirements.md#req-16-018), [REQ-16.019](ep-requirements.md#req-16-019))  
Given identical user text and final assistant reply and the same **event-aligned date**  
When **indexTurn** runs twice successfully  
Then **vec_turns** row count for that **document id** remains one.

<a id="ac-16-014"></a>**AC-16.014** (Trace: [REQ-16.020](ep-requirements.md#req-16-020))  
Given migration flag detects legacy rows excluded from retrieval  
When the process starts  
Then logs contain the **INFO** line prescribed in [ep-system-design.md](ep-system-design.md).

<a id="ac-16-015"></a>**AC-16.015** (Trace: [REQ-16.022](ep-requirements.md#req-16-022))  
Given merged chunks from notes, summary, and turn tables  
When the dynamic tail formatter runs  
Then each chunk presented to the LLM includes the correct type label token for its source class.

<a id="ac-16-016"></a>**AC-16.016** (Trace: [REQ-16.023](ep-requirements.md#req-16-023))  
Given **write_memory** arguments crafted to escape **memory_dir**  
When the tool runs  
Then the call fails with validation error and no file is written.

<a id="ac-16-017"></a>**AC-16.017** (Trace: [REQ-16.024](ep-requirements.md#req-16-024))  
Given **write_memory** `text` longer than configured max append runes or bytes  
When the tool runs  
Then the call fails before filesystem write.

<a id="ac-16-018"></a>**AC-16.018** (Trace: [REQ-16.026](ep-requirements.md#req-16-026)–[REQ-16.028](ep-requirements.md#req-16-028))  
Given the implementation branch  
When `make check` runs from the repository root  
Then the command exits zero  
When `make build && ./bin/validate EP-016` runs  
Then the command exits zero.
