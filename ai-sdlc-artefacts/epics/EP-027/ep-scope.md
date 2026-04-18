# Epic scope — EP-027 Composition root and application lifecycle

| Field | Content |
|-------|---------|
| **ID** | EP-027 |
| **Status** | DONE |
| **Title** | Composition root and application lifecycle |
| **Description** | Split the main binary startup wiring into subsystem constructors and introduce an explicit application type that owns lifecycle and teardown, so adding a new subsystem no longer grows a single wiring function. |
| **First version date** | 2026-04-17 |

## Glossary

- **Composition root**: the single place where all subsystems are constructed and wired together.
- **Application type**: a struct returned by the composition root that exposes the runnable handler and a close method.
- **Subsystem constructor**: a dedicated function that builds one subsystem (memory, tools, jobs, LLM, runtime skills) from configuration.

## Scope (features/capabilities)

- Replace the monolithic wiring function with a small set of subsystem constructors returning typed values.
- The main binary constructs an application type that owns the handler and a single close method covering all subsystems.
- The jobs subsystem initializes through the same application type; the hand-off between the asynchronous jobs runtime readiness and the user-facing tool produces a clear not-ready response instead of a generic error.
- Lint exceptions for cyclomatic complexity on the startup path are removed.

## Success criteria

- Adding a new subsystem touches at most its own constructor and one line in the composition root.
- Teardown runs through a single close path with no goroutine leaks in the standard test suite.
- The jobs not-ready path is covered by a deterministic test.
- Full quality gate passes on the change set.

## Traceability

- **Scope:** Evolvability criterion in [scope.md](../../scope.md).
- **Strategy:** Incremental delivery in [strategy.md](../../strategy.md) §1.1.
- **Related:** Recommendations §10.2 and weaknesses on composition root and jobs integration in [pa-architecture-review.md](../../pa-architecture-review.md).
