# EP-002 — Implementation plan

**Pipeline:** Stage 8.  
**Purpose:** Ordered coding tasks for automatic summarization, vector text/chunk labels, native memory tool, and skill wiring.

**Inputs:** [ep-scope.md](ep-scope.md), [ep-requirements.md](ep-requirements.md), [ep-acceptance-criteria.md](ep-acceptance-criteria.md), [ep-system-design.md](ep-system-design.md).  
**Recommended before stage 9:** [ep-system-design-review.md](ep-system-design-review.md) produced by a **delegated subagent** per [pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md) §3 (stage 7).

**Checkpoints**

- Run `make check` after each major task group before merging.
- Stop and ask the user if `pa_timezone` log slicing rules disagree with existing `llmlog` layout.

---

## Task list

- [ ] **1. Time and calendar helpers (`pa_timezone`)**  
  - Add helpers: “previous calendar day/month/year in zone”, first local day of month/year detection; unit tests with fixed `time.LoadLocation`. (Actual fire detection uses fixed **01:00** rules in **`internal/memoryjob`**.)  
  - Wire inputs for summarization job enqueue (no behaviour change to CLI yet).  
  - _Requirements:_ [REQ-02.001](ep-requirements.md#automatic-summarization-schedule)–[REQ-02.003](ep-requirements.md#automatic-summarization-schedule), [REQ-02.005](ep-requirements.md#startup-catch-up)–[REQ-02.007](ep-requirements.md#startup-catch-up)  
  - _Acceptance Criteria:_ [AC-02.001](ep-acceptance-criteria.md#ac-02-001)–[AC-02.003](ep-acceptance-criteria.md#ac-02-003) (test harness)  
  - **Verification:** Unit tests pass; helpers covered for DST edge (at least one zone with DST in test table).

- [ ] **2. Package `internal/memoryjob` (queue + priority + timeout)**  
  - Single **priority queue**: **lower number runs first** — priorities **4** (reconcile), **5** (catch-up), **10** (scheduled) in code. **Interactive precedence:** while a user turn is active, **`core.UserTurnInProgress()`** causes jobs with priority **≥ 5** to be re-queued (backoff); **reconciliation (4) is not deferred** by that guard (see [ep-system-design.md](ep-system-design.md) Job queue row). Do **not** require enqueueing interactive work into `memoryjob` at priority **0** unless you explicitly add that path.  
  - Context with timeout per summarization invocation.  
  - _Requirements:_ [REQ-02.015](ep-requirements.md#non-functional)  
  - _Acceptance Criteria:_ [AC-02.016](ep-acceptance-criteria.md#ac-02-016)  
  - **Verification:** Unit test proves dequeue order (e.g. lower numeric priority before higher when both queued); deferral behaviour covered by design + code comments.

- [ ] **3. Schedule + startup catch-up wiring**  
  - From `cmd/pa` bot mode: start memory job runner; on tick run day/month/year jobs; on startup enqueue catch-up per REQ-02.005–007.  
  - Ensure [REQ-02.004](ep-requirements.md#automatic-summarization-schedule): no dependency on external cron.  
  - _Requirements:_ [REQ-02.001](ep-requirements.md#automatic-summarization-schedule)–[REQ-02.007](ep-requirements.md#startup-catch-up), [REQ-02.004](ep-requirements.md#automatic-summarization-schedule)  
  - _Acceptance Criteria:_ [AC-02.004](ep-acceptance-criteria.md#ac-02-004)–[AC-02.007](ep-acceptance-criteria.md#ac-02-007)  
  - **Verification:** Integration test with fake clock or short interval; manual smoke optional per ep-scope E2E.

- [ ] **4. Align `summarize` + LLM log selection with `pa_timezone`**  
  - Day transcript selection uses same calendar-day definition as `memory_dir` paths.  
  - File write before vector update; on vector error, enqueue **vector reconciliation** job (read existing file → embed → upsert, no LLM). Startup reconciliation pass scans **day** summaries only over a **bounded day window** (see design).  
  - _Requirements:_ [REQ-02.001](ep-requirements.md#automatic-summarization-schedule), [REQ-02.016](ep-requirements.md#non-functional)  
  - _Acceptance Criteria:_ [AC-02.017](ep-acceptance-criteria.md#ac-02-017)  
  - **Verification:** Test simulates embed failure after file write; reconciliation completes without re-summarizing.

- [ ] **5. Vector: date line + chunk type labels**  
  - Extend `indexTurn` and summary indexing to include `Date:` (or equivalent) and type prefix for retrieval assembly.  
  - _Requirements:_ [REQ-02.008](ep-requirements.md#date-and-chunk-labels-in-vector-memory), [REQ-02.009](ep-requirements.md#date-and-chunk-labels-in-vector-memory)  
  - _Acceptance Criteria:_ [AC-02.008](ep-acceptance-criteria.md#ac-02-008), [AC-02.009](ep-acceptance-criteria.md#ac-02-009)  
  - **Verification:** Unit/integration tests on stored text and assembled context.

- [ ] **6. Native tool `read_memory`**  
  - Implement handler id **`read_memory`**: ISO `date` or `from`/`to`, path validation, max span / max output bytes; **reject** over-limit (no truncation).  
  - Register on native registry when memory is configured; add id to EP-013 **native allowlist** used by `ValidateToolRefs`; JSON schema for LLM. Optional JSON **limits** for span and output size only.  
  - _Requirements:_ [REQ-02.010](ep-requirements.md#memory-retrieval-skill-and-native-tool)  
  - _Acceptance Criteria:_ [AC-02.010](ep-acceptance-criteria.md#ac-02-010), [AC-02.011](ep-acceptance-criteria.md#ac-02-011)  
  - **Verification:** Unit tests for rejection cases; integration read after write.

- [ ] **7. Memory retrieval skill package (repo template + docs)**  
  - Add sample skill under operator `skills_dir` with `SKILL.md` listing `tools: ["read_memory"]`, policy for calls and `pa_timezone` phrasing; align with EP-013 startup (`vec_skills` build, tool union).  
  - _Requirements:_ [REQ-02.011](ep-requirements.md#memory-retrieval-skill-and-native-tool)  
  - _Acceptance Criteria:_ [AC-02.012](ep-acceptance-criteria.md#ac-02-012)  
  - **Verification:** Config load test: skill validates; manual E2E Part C.

- [ ] **8. Vector retrieval without tool**  
  - Confirm handler still runs semantic search when memory tool not called.  
  - _Requirements:_ [REQ-02.012](ep-requirements.md#memory-retrieval-skill-and-native-tool)  
  - _Acceptance Criteria:_ [AC-02.013](ep-acceptance-criteria.md#ac-02-013)  
  - **Verification:** Integration test.

- [ ] **9. Upsert idempotence**  
  - Re-run same day summarization twice; assert single vector id for `summary:day:…`.  
  - _Requirements:_ [REQ-02.013](ep-requirements.md#upsert-semantics)  
  - _Acceptance Criteria:_ [AC-02.014](ep-acceptance-criteria.md#ac-02-014)  
  - **Verification:** Test passes.

- [ ] **10. Config + documentation**  
  - Document **`pa_timezone`**, paths, embedding/vector prerequisites for the summarization worker, and optional **`read_memory`** limits in `docs/configuration.md`. Document that automatic summarization timing is **not** operator-configurable (fixed in **`internal/memoryjob`**).  
  - _Requirements:_ [REQ-02.014](ep-requirements.md#non-functional)  
  - _Acceptance Criteria:_ [AC-02.015](ep-acceptance-criteria.md#ac-02-015)  
  - **Verification:** `make check`; link from README if required by house style.

---

## Traceability summary

| REQ | Tasks |
|-----|--------|
| 02.001–02.004 | 1, 3, 4 |
| 02.005–02.007 | 1, 3 |
| 02.008–02.009 | 5 |
| 02.010–02.012 | 6, 7, 8 |
| 02.013 | 4, 9 |
| 02.014–02.016 | 2, 4, 10 |
