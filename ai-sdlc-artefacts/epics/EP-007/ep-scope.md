# Epic scope — EP-007 Observability: correlation, local analytics, and metrics

| Field | Content |
|-------|---------|
| **ID** | EP-007 |
| **Status** | NEW |
| **Title** | Observability: correlation, local analytics, and metrics |
| **Description** | Strengthen operator visibility without mandatory external LLM SaaS: stable correlation across a user turn, first-class use of existing JSONL LLM logs, and optional in-process metrics export. Aligns with existing `slog` + `internal/llmlog` and NFRs on redaction and no secrets in logs. |
| **First version date** | 2026-03-19 (date of first approved write of this document; start of work on the epic idea) |

## Glossary

- **Turn / user message handling:** One inbound user message through final assistant reply (may include multiple LLM `Complete` calls and tool rounds).
- **Correlation ID:** Identifier carried across all log lines and LLM log entries for a single turn (or session subtree), so operators can filter one story end-to-end.
- **LLM JSONL log:** Daily files under configured `llm_log_dir` (`llm-YYYY-MM-DD.jsonl`) as written by `internal/llmlog`.
- **Metrics export:** Counters/histograms (e.g. LLM latency, tool outcomes, escalation counts) exposed in a standard form (e.g. Prometheus text format or OpenTelemetry metrics), optional and off by default if not configured.

## Scope (features/capabilities)

1. **Structured logs and correlation**  
   - Introduce or standardize a **correlation ID** (and related fields) for the full turn, propagated to `slog` and to `llmlog.Entry` where feasible.  
   - Align **field names and semantics** with existing requirements (tool traces, escalation classification, provider before/after, optional `tried_providers`) so logs answer operator questions without ad-hoc grep patterns.  
   - Preserve **redaction** and **no secrets**; no new sinks that bypass existing redactors.

2. **Local analytics for `llm_logs`**  
   - **Documented** workflows and/or small **maintained utilities** (e.g. `jq` recipes, optional script under `cmd/` or `scripts/`) to summarize: latency, token usage, model mix, error/escalation rates from JSONL.  
   - Optional: **human-readable report** (markdown/stdout) for a date range or request id.  
   - No requirement to ship data to third parties.

3. **Metrics**  
   - **Optional** metrics surface (feature-flagged or config-gated): e.g. histogram of LLM call duration, counters for tool success/failure, escalation events, provider index usage.  
   - Prefer **standard formats** (Prometheus scrape or OTel metrics) and **fail-soft** behaviour if the metrics backend is down (must not break request handling).  
   - Labels must not include raw user content or secrets.

## Success criteria

- Operators can take a single **correlation ID** from user-facing or app logs and find **all** related `slog` lines and the matching **JSONL LLM record(s)** for that turn (where logging is enabled).  
- README or ops doc describes **how to run** at least one documented analysis path over `llm_logs` without reading Go code.  
- With metrics enabled in a test configuration, scrapers or tests can observe **non-zero** samples for at least LLM duration and one business counter (e.g. tool or escalation), without leaking PII in label values.  
- Existing tests pass; new behaviour covered by unit/integration tests as appropriate.

## Traceability

- **Scope:** [scope.md](../../scope.md) — PersonalAssistant focuses on reliability; observability supports operations and regression analysis.  
- **Strategy:** [strategy.md](../../strategy.md) — Testable increments; security and redaction remain explicit.  
- **Builds on:** [EP-001](../EP-001/ep-scope.md) (logging/redaction, `llmlog`), [EP-004](../EP-004/ep-scope.md) (tool traceability), [EP-006](../EP-006/ep-scope.md) (escalation observability fields).

## Out of scope (this epic)

- Integration with **Langfuse** or other LLM-specific SaaS (separate decision/epic).  
- **Full** MLOps / prompt registry / hosted eval UI.  
- Replacing file-based `llm_logs` with-only remote storage.  
- **Golden-dataset CI eval** as a mandatory deliverable (may be a **follow-up** epic: versioned fixtures + runner + threshold gates).

---

*Next pipeline stages:* [ep-requirements.md](ep-requirements.md), [ep-system-design.md](ep-system-design.md), [ep-implementation-plan.md](ep-implementation-plan.md), [ep-acceptance-criteria.md](ep-acceptance-criteria.md) per [pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md).
