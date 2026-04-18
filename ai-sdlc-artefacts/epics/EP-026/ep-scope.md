# Epic scope — EP-026 Core refactor: tier builders in conversation handler

| Field | Content |
|-------|---------|
| **ID** | EP-026 |
| **Status** | DONE |
| **Title** | Core refactor: tier builders in conversation handler |
| **Description** | Refactor the conversation handler so each prompt tier (simple, full-lite, full) is produced by an explicit builder with a single entry point, removing duplicated branches and making the tier contract testable on its own. |
| **First version date** | 2026-04-17 |

## Glossary

- **Tier**: one of the prompt tiers selected by the intent classifier (simple, full-lite, full).
- **Tier builder**: a function that produces the prompt options, tools, and system tail for exactly one tier.
- **Conversation handler**: the core component that orchestrates a single inbound message through classification, prompt assembly, LLM completion, and tool loop.

## Scope (features/capabilities)

- Each tier has a dedicated builder with a clear input and output contract; duplicated branches in the conversation handler are removed.
- The main handler entry point reads as a linear orchestrator: pick tier, call builder, call router, run tool loop, finalize.
- Each tier builder is covered by unit tests that do not require the full handler graph.
- The lint exceptions previously applied to the handler for cyclomatic complexity are removed or reduced.

## Success criteria

- The main handler entry point no longer carries its previous cyclomatic exception.
- Each tier builder has unit tests exercising its contract without wiring a full process.
- No externally observable behaviour changes for any tier compared to the baseline.
- Full quality gate passes on the change set.

## Traceability

- **Scope:** Evolvability criterion in [scope.md](../../scope.md) — "architecture is designed to evolve without radical redesign".
- **Strategy:** Regression guarantees in [strategy.md](../../strategy.md) §1.2.
- **Related:** Recommendations §10.1 and weakness "god handler" in [pa-architecture-review.md](../../pa-architecture-review.md).
