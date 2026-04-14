# EP-016 — Acceptance criteria

**Introduction:** Testable acceptance criteria for **EP-016** (manual `notes.md`, `write_memory`, extended `read_memory`, split vector memory, event-aligned turn dates, turn deduplication). Each AC traces to [ep-requirements.md](ep-requirements.md).

---

## Acceptance criteria index

| AC ID | REQ | Summary |
|-------|-----|---------|
| [AC-16.001](#ac-16-001) | [REQ-16.001](ep-requirements.md#req-16-001) | notes.md lives beside summary.md for a resolved day |
| [AC-16.002](#ac-16-002) | [REQ-16.002](ep-requirements.md#req-16-002) | Summarize write does not delete notes.md |
| [AC-16.003](#ac-16-003) | [REQ-16.004](ep-requirements.md#req-16-004), [REQ-16.008](ep-requirements.md#req-16-008) | write_memory append starts with RFC3339 UTC line |
| [AC-16.004](#ac-16-004) | [REQ-16.006](ep-requirements.md#req-16-006) | Oversized append rejected with clear limit error |
| [AC-16.005](#ac-16-005) | [REQ-16.009](ep-requirements.md#req-16-009) | write_memory rejects path outside memory_dir |
| [AC-16.006](#ac-16-006) | [REQ-16.011](ep-requirements.md#req-16-011), [REQ-16.012](ep-requirements.md#req-16-012) | read_memory returns summary and notes sections when both exist |
| [AC-16.007](#ac-16-007) | [REQ-16.013](ep-requirements.md#req-16-013) | read_memory skips days with neither file |
| [AC-16.008](#ac-16-008) | [REQ-16.014](ep-requirements.md#req-16-014) | read_memory still enforces max_span_days and max_output_bytes |
| [AC-16.009](#ac-16-009) | [REQ-16.015](ep-requirements.md#req-16-015), [REQ-16.016](ep-requirements.md#req-16-016) | Integration or unit proof that summary upsert targets summary table and turn add targets turn table |
| [AC-16.010](#ac-16-010) | [REQ-16.017](ep-requirements.md#req-16-017) | Retrieved chunk ordering places notes before summary before turn for equal scores fixture |
| [AC-16.011](#ac-16-011) | [REQ-16.018](ep-requirements.md#req-16-018) | Legacy vec_items query path returns only summary-prefixed ids |
| [AC-16.012](#ac-16-012) | [REQ-16.019](ep-requirements.md#req-16-019) | Turn retrieval does not duplicate legacy vec_items turn with dedicated turn row |
| [AC-16.013](#ac-16-013) | [REQ-16.020](ep-requirements.md#req-16-020) | Turn chunk Date line matches adapter-supplied message calendar day in pa_timezone |
| [AC-16.014](#ac-16-014) | [REQ-16.021](ep-requirements.md#req-16-021) | When adapter timestamp absent, Date line follows documented fallback |
| [AC-16.015](#ac-16-015) | [REQ-16.022](ep-requirements.md#req-16-022), [REQ-16.023](ep-requirements.md#req-16-023) | Re-indexing same turn twice does not increase dedicated turn table row count by two |
| [AC-16.016](#ac-16-016) | [REQ-16.010](ep-requirements.md#req-16-010) | After write_memory, notes vector search returns the new entry |
| [AC-16.017](#ac-16-017) | [REQ-16.005](ep-requirements.md#req-16-005) | Optional kind appears in stored notes line when provided |
| [AC-16.018](#ac-16-018) | [REQ-16.007](ep-requirements.md#req-16-007) | write_memory appears in native tool registry when allowlist permits |
| [AC-16.019](#ac-16-019) | [REQ-16.024](ep-requirements.md#req-16-024) | Log path for write_memory uses redactor when configured (behaviour test or handler test) |
| [AC-16.020](#ac-16-020) | [REQ-16.025](ep-requirements.md#req-16-025), [REQ-16.026](ep-requirements.md#req-16-026) | **Deferred:** `./bin/validate EP-016` is enforced by CI (`make check` then validate); no standalone unit test (bootstrap) |
| [AC-16.021](#ac-16-021) | [REQ-16.027](ep-requirements.md#req-16-027) | Curated runtime memory-tool doc lists write_memory when the profile enables it |

---

## Acceptance criteria

<a id="ac-16-001"></a>**AC-16.001** (Trace: REQ-16.001)  
Given a memory store rooted at `memory_dir` with pa_timezone `Europe/Amsterdam`  
When `write_memory` creates the first entry for `2026-04-10`  
Then the file path SHALL equal `<memory_dir>/2026/04/10/notes.md`.

<a id="ac-16-002"></a>**AC-16.002** (Trace: REQ-16.002)  
Given `notes.md` exists for a day with content `preserve-me`  
When the Summarize job writes a new `summary.md` for that same calendar day in tests  
Then `notes.md` SHALL still contain `preserve-me`.

<a id="ac-16-003"></a>**AC-16.003** (Trace: REQ-16.004, REQ-16.008)  
Given a valid `write_memory` call with plain text  
When the tool completes successfully  
Then the appended block SHALL begin with a line matching RFC3339 UTC timestamp pattern `Z` and the body SHALL contain the submitted text.

<a id="ac-16-004"></a>**AC-16.004** (Trace: REQ-16.006)  
Given `max_bytes_per_append` configured below the length of submitted text  
When the operator invokes `write_memory`  
Then the tool SHALL return an error whose message names the per-append limit.

<a id="ac-16-005"></a>**AC-16.005** (Trace: REQ-16.009)  
Given parameters that would resolve outside `memory_dir`  
When `write_memory` runs  
Then the tool SHALL return an error and SHALL not create files outside `memory_dir`.

<a id="ac-16-006"></a>**AC-16.006** (Trace: REQ-16.011, REQ-16.012)  
Given a day with both `summary.md` and `notes.md` populated  
When `read_memory` queries that single ISO date  
Then the output SHALL contain distinct headings for automatic day summary versus manual notes and SHALL include text from both files.

<a id="ac-16-007"></a>**AC-16.007** (Trace: REQ-16.013)  
Given a date range where a middle day has neither summary nor notes  
When `read_memory` runs for the range  
Then the output for that middle day SHALL be absent (no empty placeholder day block).

<a id="ac-16-008"></a>**AC-16.008** (Trace: REQ-16.014)  
Given `max_span_days` of 2  
When the operator requests a 10-day range  
Then `read_memory` SHALL return an error referencing the span limit.

<a id="ac-16-009"></a>**AC-16.009** (Trace: REQ-16.015, REQ-16.016)  
Given sqlite vector tables `vec_summaries` and `vec_turns` exist for the deployment  
When a day summary is upserted and a turn is indexed in one test scenario  
Then the summary row SHALL appear only in the summary virtual table and the turn row only in the turn virtual table.

<a id="ac-16-010"></a>**AC-16.010** (Trace: REQ-16.017)  
Given controlled embedding stubs return fixed ordering for notes, summary, and turn searches  
When retrieval assembles chunks for the dynamic tail  
Then concatenated chunk order SHALL list all notes hits before any summary hit and all summary hits before any turn hit.

<a id="ac-16-011"></a>**AC-16.011** (Trace: REQ-16.018)  
Given `vec_items` contains one row with id `turn-legacy` and one with id `summary:day:2026-04-01`  
When the summary legacy query path runs with top-K large enough  
Then results SHALL include only the `summary:day:` row.

<a id="ac-16-012"></a>**AC-16.012** (Trace: REQ-16.019)  
Given legacy `vec_items` holds an old turn and the dedicated turn table holds the same logical turn id after migration-style setup  
When turn retrieval runs  
Then the assembled chunk list SHALL contain at most one chunk for that stable turn id.

<a id="ac-16-013"></a>**AC-16.013** (Trace: REQ-16.020)  
Given the adapter passes a Telegram message unix time mapping to `2026-04-02` in pa_timezone  
When the core indexes the turn  
Then the stored chunk SHALL contain `Date: 2026-04-02`.

<a id="ac-16-014"></a>**AC-16.014** (Trace: REQ-16.021)  
Given no adapter timestamp is supplied in a test harness  
When the core indexes the turn using the documented fallback  
Then the `Date` line SHALL still be a valid ISO calendar date in pa_timezone per the documented rule.

<a id="ac-16-015"></a>**AC-16.015** (Trace: REQ-16.022, REQ-16.023)  
Given identical user text and assistant reply and the same event-aligned day  
When `indexTurn` is invoked twice in the test  
Then the dedicated turn table row count for that stable id SHALL remain one.

<a id="ac-16-016"></a>**AC-16.016** (Trace: REQ-16.010)  
Given `write_memory` appends new text for a day  
When vector search runs against the notes table with a query embedding matching that text  
Then at least one result id SHALL map to that notes append.

<a id="ac-16-017"></a>**AC-16.017** (Trace: REQ-16.005)  
Given `write_memory` is called with `kind` set to `preference`  
When `notes.md` is read back  
Then the file SHALL contain a token or field agreed in design that records `preference`.

<a id="ac-16-018"></a>**AC-16.018** (Trace: REQ-16.007)  
Given native tools include `write_memory` in test configuration  
When the tool registry is enumerated  
Then `write_memory` SHALL appear with non-empty description and JSON schema.

<a id="ac-16-019"></a>**AC-16.019** (Trace: REQ-16.024)  
Given a redacting logger wrapper is configured for tool args in tests  
When `write_memory` logs its parameters  
Then the stored log line SHALL not contain a synthetic secret token present in the args fixture.

<a id="ac-16-020"></a>**AC-16.020** (Trace: REQ-16.025, REQ-16.026) **Deferred:** CI SHALL run `make check` and `./bin/validate EP-016` from the repository root on changes affecting EP-016 coverage; this AC is not mapped to a unit test (bootstrap / gate criterion).

<a id="ac-16-021"></a>**AC-16.021** (Trace: REQ-16.027)  
Given the operator-facing runtime or configuration documentation for native memory tools in this repository  
When that document lists tools available for memory read/write in the curated profile used by EP-013  
Then the document SHALL name `write_memory` in the same section or table as `read_memory` when `write_memory` is part of that profile.
