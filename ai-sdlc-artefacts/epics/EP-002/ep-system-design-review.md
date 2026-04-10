# Architecture Review — EP-002 Automatic memory summarization

**Review date:** 2026-04-10  
**Reviewer:** Delegated subagent (stage 7)  
**Document reviewed:** [ep-system-design.md](ep-system-design.md)

---

## 1. Overall Assessment

The system design is coherent with [ep-scope.md](ep-scope.md): dedicated **`internal/memoryjob`**, pa_timezone-aligned scheduling, file-then-vector ordering plus **vector reconciliation**, **priority queue** semantics (0 / 5 / 10), native **`read_memory`** tool with **reject** on over-limit ranges, EP-013 **`tools: ["read_memory"]`** wiring, and **Risks and trade-offs** in design. Medium items from the initial review (2026-04-10) were addressed in a **follow-up edit** to ep-system-design, ep-requirements, ep-acceptance-criteria, and ep-implementation-plan.

**Verdict:** Ready

---

## 2. Strengths

### 2.1 Scope and boundaries

- Explicit alignment with scope: no core rule-based phrase→date resolver; recall via native tool + runtime skill ([ep-system-design.md](ep-system-design.md) Overview, Module boundaries).
- Package boundary matches scope decision: scheduling/queueing not embedded in Telegram/handler core (`internal/memoryjob`).

### 2.2 Architecture

- C4 container diagram and module boundaries table give a clear C2-level view ([ep-system-design.md](ep-system-design.md) Architecture).
- Persistence ordering (write `memory_dir` first, then vector) matches scope and supports [REQ-02.016](ep-requirements.md#non-functional).

### 2.3 Security and fail-fast

- Native tool contract rejects path injection; invalid ISO and oversized range → tool error ([ep-system-design.md](ep-system-design.md) Error handling).
- Config: invalid/missing `pa_timezone` → fail fast at startup ([ep-system-design.md](ep-system-design.md) Error handling). Automatic summarization wall-clock times and related intervals are **code constants** in **`internal/memoryjob`** (no `memory_summarization` JSON section).

### 2.4 Testability

- Testing strategy maps unit/integration to acceptance criteria and scope manual E2E ([ep-system-design.md](ep-system-design.md) Testing strategy).

---

## 3. Issues and Recommendations

### 3.1 Critical

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| — | None | All REQ-02.001–REQ-02.016 have design hooks and AC pairs. | — |

### 3.2 Medium (addressed in follow-up 2026-04-10)

| # | Issue | Resolution |
|---|-------|------------|
| M1 | Missing **Risks and trade-offs** | Added section in [ep-system-design.md](ep-system-design.md). |
| M2 | REQ-02.015 queue integration | Documented **priority values** (0 / 5 / 10), single queue, handler enqueue at 0; [ep-implementation-plan.md](ep-implementation-plan.md) task 2. |
| M3 | Tool auth/session | **`read_memory`** trust boundary = same as other native tools in tool-calling requests; always registered when memory is configured; redaction note in design. |
| M4 | Reject vs truncate | **Reject** baseline; REQ-02.010, AC-02.011, design aligned. |

### 3.3 Minor (addressed or accepted)

| # | Issue | Resolution |
|---|-------|------------|
| n1 | REQ-02.004 traceability | Traceability row expanded in design. |
| n2 | Catch-up ordering | **Priority 5** before timer **10**; AC-02.005 updated. |
| n3 | Package name TBD | Fixed **`internal/memoryjob`**. |
| n4 | ep-scope link in overview | Added in design Overview. |
| ext | REQ-02.016 vs catch-up | **Vector reconciliation** job on startup/cycle; REQ / AC / design / plan updated. |
| ext | EP-001 glossary link | Fixed in [ep-requirements.md](ep-requirements.md). |

---

## 4. Architectural Decisions

### 4.1 Justified Trade-offs

| Decision | Justification |
|----------|---------------|
| Single internal job queue with interactive-first ordering | Matches [ep-scope.md](ep-scope.md) “single queue with priority”; keeps KISS vs multiple schedulers. |
| File write before vector index; retry on later run | Matches scope persistence ordering and [REQ-02.016](ep-requirements.md#non-functional); avoids tight retry loops. |
| Skill (EP-013) owns phrase policy; core registers native tool only | Preserves modularity and scope boundary (no hybrid rule engine in core). |
| Dedicated `summarize` timeout context for background jobs | Limits resource capture from live traffic; pairs with queue priority. |

### 4.2 Potential Improvements (post-MVP)

1. Observability: metrics for job latency, catch-up depth, vector upsert failures (without logging raw memory content).
2. Explicit deduplication or idempotency keys if double scheduling becomes costly operationally.

---

## 5. NFR Coverage

| NFR | Coverage | Status |
|-----|----------|--------|
| [REQ-02.004](ep-requirements.md#automatic-summarization-schedule) | Built-in scheduling; `cmd/pa` + memoryjob; no external cron | OK |
| [REQ-02.014](ep-requirements.md#non-functional) | Testing strategy; `make check` / AC-02.015 | OK |
| [REQ-02.015](ep-requirements.md#non-functional) | Job queue; priority 0/5/10; handler contract | OK |
| [REQ-02.016](ep-requirements.md#non-functional) | Summarize runner; **vector reconciliation** worker | OK |

---

## 6. Project Rules Compliance

| Rule | Compliance |
|------|------------|
| KISS | ✅ Single queue, dedicated package, no parallel rule engine in core |
| Fail fast | ✅ Startup config for `pa_timezone`; tool validation errors to model |
| Security | ✅ Path rejection, reject on limit, trust boundary; **`read_memory`** registered when **memory_dir** is configured (limits-only JSON) |
| Testability | ✅ Unit/integration/manual split; AC mapping stated |
| Modularity | ✅ memoryjob / summarize / core / vector / EP-013 skill boundary clear |

---

## 7. Summary

**Ready** — Follow-up edits closed M1–M4 and related stage-10 **Major** documentation findings (glossary link, REQ-02.016 reconciliation story, `read_memory` id, EP-013 `tools` list, queue testability).

**Optional post-merge:** Observability metrics for queue delay; bounded **day-only** reconciliation scan and non-deferred reconcile (§9.4) are documented in design risks.

---

## 8. Requirement traceability (REQ → design → AC)

| REQ | Design coverage (ep-system-design.md) | Acceptance criteria | Alignment |
|-----|----------------------------------------|---------------------|-----------|
| REQ-02.001 | Overview; Memory job scheduler; Summarize runner; LLM log reader | AC-02.001 | OK |
| REQ-02.002 | Memory job scheduler; Summarize runner | AC-02.002 | OK |
| REQ-02.003 | Memory job scheduler; Summarize runner | AC-02.003 | OK |
| REQ-02.004 | Overview; module boundaries; Built-in scheduling row | AC-02.004 | OK (minor: enrich traceability row — n1) |
| REQ-02.005 | Catch-up coordinator; LLM log reader | AC-02.005 | OK (minor: ordering vs tick — n2) |
| REQ-02.006 | Catch-up coordinator | AC-02.006 | OK |
| REQ-02.007 | Catch-up coordinator | AC-02.007 | OK |
| REQ-02.008 | Vector indexer; Data models | AC-02.008 | OK |
| REQ-02.009 | Retrieval assembler; Data models | AC-02.009 | OK |
| REQ-02.010 | Native **`read_memory`**; Data models | AC-02.010, AC-02.011 | OK |
| REQ-02.011 | Runtime skills; Tool + skill wiring | AC-02.012 | OK (EP-013 deliverable) |
| REQ-02.012 | Components: retrieval independent of tool; core vector path | AC-02.013 | OK |
| REQ-02.013 | Summarize + vector; Vector indexer (stable ids, upsert) | AC-02.014 | OK |
| REQ-02.014 | Testing strategy | AC-02.015 | OK |
| REQ-02.015 | Job queue; module boundaries | AC-02.016 | OK with Medium: integration detail (M2) |
| REQ-02.016 | Summarize runner; Error handling; Vector upsert | AC-02.017 | OK |

---

## 9. Implementation / code review (stage 10)

**Review date:** 2026-04-10  
**Scope:** Product code under `cmd/pa`, `internal/memoryjob`, `internal/tools/read_memory.go`, `internal/core` (retrieval, user-turn guard), `internal/summarize`, `internal/config` — against [ep-system-design.md](ep-system-design.md) and EP-002 requirements.  
**Verification:** `make check` (fmt, vet, govulncheck, golangci-lint, `go test -race -tags=integration ./...`, coverage) — **pass** at time of review.

### 9.1 Overall

Implementation matches the epic’s technical intent: dedicated `internal/memoryjob` with code constants for schedule and scan depth, file-then-vector flow with `ErrVectorIndexAfterFileWrite` and reconciliation enqueue, native `read_memory` with reject-on-limit semantics, `pa_timezone` validation at config load, chunk labels and stable vector ids in `internal/summarize`, EP-013 native allowlist including `read_memory`.

**Verdict:** **Shippable** with documentation and optional hardening items below (no blocking defects found in code review).

### 9.2 Findings

| Severity | Id | Topic | Observation | Recommendation |
|----------|-----|--------|-------------|----------------|
| Low (doc) | C1 | Interactive vs background ([REQ-02.015](ep-requirements.md#non-functional)) | Design and plan still describe enqueueing interactive work at **priority 0**. Code does **not** enqueue handler work on `memoryjob.Runner`; it uses `core.EnterUserTurn` / `UserTurnInProgress` and defers jobs with `priority >= PriorityCatchUp` only. | Update [ep-system-design.md](ep-system-design.md) Components table / job queue row and [ep-implementation-plan.md](ep-implementation-plan.md) to describe the **user-turn deferral** model explicitly; remove or reword “enqueue at 0” if no code path will use it. |
| Low | C2 | Reconciliation during user turn | `PriorityReconcile` (**4**) is **not** deferred when `UserTurnInProgress`; only priorities **≥ 5** re-queue. Reconciliation can run concurrently with an active LLM turn (embedder contention). | Either extend deferral to reconciliation while a user turn is active, or document as an intentional trade-off (faster vector consistency vs strict interactive isolation). |
| Low | C3 | Startup reconciliation scan | `runReconciliationScan` walks **day** summaries only over the last **90** days; month/year files are not scanned for missing vector rows. | Accept as bounded MVP (align wording in design **Risks**), or add a bounded month/year scan later if operational data shows gaps after crashes. |
| Low | C4 | Duplicate chunk type labels ([REQ-02.009](ep-requirements.md#date-and-chunk-labels-in-vector-memory)) | Stored vector text already includes `[turn]` or `[summary:*]`; `gatherRetrievedChunkTexts` prepends `[` + label + `]\n` again, so the model may see the label twice. | Optional cleanup: skip outer prefix when body already starts with the same label, or store without inline label and rely on retrieval prefix only (larger change). |
| Low | C5 | `paLocationOrUTC` in `cmd/pa` | After successful `config.Load`, IANA name is validated; `LoadLocation` failure still falls back to UTC in `paLocationOrUTC`. Unlikely in practice post-validation. | Prefer `config.PALocation(cfg)` and fail startup if location resolution errors after load, for strict alignment with fail-fast wording. |
| Info | C6 | Daily fire window | Daily/month/year enqueue uses local hour **01** with `last*FireKey` deduplication; any tick within **01:xx** can fire once per key, not only minute `00`. | Document as implementation detail (acceptable); or narrow to `Minute() == 0` if operators expect a single-minute window. |

### 9.3 Strengths (code)

- Clear separation: `memoryjob` owns queue, tick, catch-up, reconciliation scan, and timeout-wrapped job execution.
- `read_memory`: ISO parsing in store location, inclusive range day count, output byte cap before assembly, path confinement via `filepath` + root prefix check.
- `IsVectorIndexAfterFileWrite` gives a single, testable signal for post-write vector failure.
- Tests reference AC ids where relevant (`internal/tools/read_memory_test.go`, `internal/core/handler_test.go`, `internal/runtimeskills/load_test.go`, integration).

### 9.4 Follow-up — documentation and code (2026-04-10)

The items in §9.2 (C1–C6) were closed as follows:

| Id | Resolution |
|----|------------|
| **C1** | [ep-system-design.md](ep-system-design.md) **Job queue** and **`internal/memoryjob`** module rows updated: interactive precedence via **`UserTurnInProgress`** deferral for priorities **≥ 5**; no enqueue of interactive work at priority **0**. [ep-implementation-plan.md](ep-implementation-plan.md) task **2** aligned. |
| **C2** | Design documents **intentional** behaviour: **reconciliation (4)** not deferred during user turn; trade-off (embedder contention vs faster vector heal) stated in [ep-system-design.md](ep-system-design.md) Job queue + Error handling. |
| **C3** | **Vector reconciliation worker** and **Risks** updated: periodic scan is **day-only**, bounded window; month/year recovery path described. |
| **C4** | **Product:** `gatherRetrievedChunkTexts` uses **`retrievalChunkWithLabel`** so stored bodies that already contain `\n[label]\n` are not prefixed twice. **Docs:** Retrieval assembler + Data models in [ep-system-design.md](ep-system-design.md). Tests: `internal/core/handler_test.go`. |
| **C5** | **Product:** `cmd/pa` uses **`config.PALocation(cfg)`** for memory job and **`-summarize`** CLI paths; startup fails if location resolution errors after load (no silent UTC fallback). |
| **C6** | **Memory job scheduler** row documents **01:xx** fire window and dedup keys (implementation detail). |

---

## Traceability

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- **Scope:** [ep-scope.md](ep-scope.md)
