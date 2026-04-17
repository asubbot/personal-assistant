# Epic scope — EP-022 Reliability hardening for local SQLite stores and outbound HTTP timeouts

| Field | Content |
|-------|---------|
| **ID** | EP-022 |
| **Status** | NEW |
| **Title** | Reliability hardening for local SQLite stores and outbound HTTP timeouts |
| **Description** | Tighten reliability of local state (sqlite-vec and jobs DB) and outbound HTTP by setting explicit SQLite PRAGMAs, documenting the single-writer expectation, and auditing every outbound HTTP client for a bounded timeout. |
| **First version date** | 2026-04-17 |

## Glossary

- **Local store**: the set of sqlite files used by the assistant (memory and vector tables, scheduled jobs store).
- **PRAGMA policy**: set of `journal_mode`, `busy_timeout`, `synchronous`, and related settings applied at every sqlite connection open.
- **Outbound HTTP**: every HTTP client inside the process that talks to Telegram, LLM providers, embedding providers, and web tools.

## Scope (features/capabilities)

- Single, explicit PRAGMA policy applied on every sqlite connection open inside the process (vector tables and jobs store), documented in operator configuration.
- Concurrency test that exercises writer paths concurrently (background summarization worker, foreground chat handler, background vector index build) under race detection without busy errors.
- Audit of every outbound HTTP client: each must carry an operator-configurable timeout (no indefinite zero timeout) for connect, read, and total request.
- Operator-facing note on the single-writer expectation for each sqlite file, and on the configured HTTP timeouts and their defaults.

## Success criteria

- Running the race-enabled concurrency test does not produce busy errors on any sqlite path used by the product.
- Every place that creates an outbound HTTP client in the codebase is either covered by a configured timeout or explicitly listed as exempt in operator docs.
- All PRAGMAs and HTTP timeouts are explicit in configuration; no hidden defaults at load.
- Full quality gate passes on the change set.

## Traceability

- **Scope:** Reliability and security focus called out in [scope.md](../../scope.md).
- **Strategy:** Delivery strategy in [strategy.md](../../strategy.md) — "existing behaviour still works; new behaviour is testable".
- **Related:** Recommendations §10.7 and risk set in [pa-architecture-review.md](../../pa-architecture-review.md); residual risks in [threat-model.md](../../threat-model.md) §7.
