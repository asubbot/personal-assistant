# EP-029 — System design review

## Review iteration 1

**Inputs reviewed:** `ep-scope.md`, `ep-requirements.md`, `ep-acceptance-criteria.md`, `ep-system-design.md` (initial).

**Structural note:** The design included overview, C4 diagrams, module boundaries, components/interfaces, data models, error handling, testing strategy, risks, and REQ traceability.

### Findings

- **F-7.1 — Major** — **Location:** `ep-system-design.md` — components (`evalReadiness`), error handling (LLM probe). **Description:** REQ-29.004 requires a **bounded** completion probe when `probe_llm` is true; the design did not spell out provider selection, timeout budget, cancellation, or mapping to `checks[].ok` / `detail`. **Recommendation:** Document probe target, hard timeout, context behaviour, and error mapping.

- **F-7.2 — Medium** — **Location:** `ep-system-design.md` — readiness JSON / REQ-29.004. **Description:** Conditional inclusion of `memory_summarization` vs `scheduled_jobs` was asymmetric in the narrative. **Recommendation:** Document gating and whether checks are omitted vs always present.

- **F-7.3 — Minor** — **Location:** `ep-system-design.md` — health JSON. **Description:** Health body not concretely specified vs AC-29.002. **Recommendation:** Add minimal JSON example.

- **F-7.4 — Nit** — **Location:** `ep-system-design.md` — `vector_stores`. **Description:** Three tables collapsed without stating aggregation semantics. **Recommendation:** Clarify single check aggregates all three.

- **F-7.5 — Suggestion** — **Location:** risks / error handling. **Description:** No statement on synchronous re-evaluation per request vs caching. **Recommendation:** State evaluation model for operators tuning scrape intervals.

**Iteration summary — open counts:** Blocker: 0, Major: 1, Medium: 1, Minor: 1

---

## Review iteration 2

**Inputs reviewed:** `ep-system-design.md` after addressing iteration 1 (health JSON example, readiness inclusion rules, `vector_stores` aggregation note, synchronous per-request evaluation, LLM probe contract with 5s deadline and first-provider selection).

### Findings

No Blocker, Major, Medium, or Minor findings.

**Iteration summary — open counts:** Blocker: 0, Major: 0, Medium: 0, Minor: 0
