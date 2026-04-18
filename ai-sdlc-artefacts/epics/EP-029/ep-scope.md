# Epic scope — EP-029 Health, readiness and operator observability surface

| Field | Content |
|-------|---------|
| **ID** | EP-029 |
| **Status** | DONE |
| **Title** | Health, readiness and operator observability surface |
| **Description** | Expose an optional HTTP surface with health and readiness endpoints that depend on real process state, and emit structured lifecycle events from background workers so operators can reason about liveness without grepping logs. |
| **First version date** | 2026-04-17 |

## Glossary

- **Health endpoint**: an HTTP path that returns a liveness verdict for the running process.
- **Readiness endpoint**: an HTTP path that returns a verdict combining reachability of critical dependencies and background subsystem readiness.
- **Lifecycle event**: a structured log record emitted by a background worker at start, success, and error boundaries with standard field names.

## Scope (features/capabilities)

- An optional HTTP listener exposes health and readiness endpoints; the listener is enabled only when the corresponding configuration field is present.
- Readiness composes checks across critical subsystems (provider reachability, vector store open, jobs runtime ready) into a single verdict.
- Background workers (memory summarization, jobs runtime, tool vector index build) emit structured lifecycle events with consistent field names and durations.
- Operator documentation explains how to consume both the HTTP endpoints and the lifecycle events in a typical Docker deployment.

## Success criteria

- A Docker healthcheck can rely only on the health endpoint to detect process death.
- The readiness endpoint reports not-ready while any critical subsystem is still initializing and ready once all are available.
- Lifecycle events from each covered worker appear in logs with the documented schema and duration field.
- Full quality gate passes on the change set.

## Traceability

- **Scope:** Reliability focus and target platform in [scope.md](../../scope.md).
- **Strategy:** Delivery strategy in [strategy.md](../../strategy.md) §1.1.
- **Related:** Recommendations §10.6 and §10.9, risks R10 and R12 in [pa-architecture-review.md](../../pa-architecture-review.md).
