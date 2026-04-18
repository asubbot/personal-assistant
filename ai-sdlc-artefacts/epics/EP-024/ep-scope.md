# Epic scope — EP-024 Operator documentation for provider roles and safe logging defaults

| Field | Content |
|-------|---------|
| **ID** | EP-024 |
| **Status** | DONE |
| **Title** | Operator documentation for provider roles and safe logging defaults |
| **Description** | Close two operator-visible gaps: document the mapping between the LLM provider pool and the roles it serves (main chat, escalation, summarize, intent classifier), and set safe logging defaults for production so that verbose LLM I/O is opt-in and explicit. |
| **First version date** | 2026-04-17 |

## Glossary

- **Provider pool**: the ordered list of LLM providers declared in configuration.
- **Provider role**: the function a provider serves in the process (main chat reply, tool-path escalation, summarization, intent classification).
- **Verbose LLM logging**: the log level at which the process logs full LLM request and response bodies.
- **Operator docs**: operator-facing markdown under the product docs tree.

## Scope (features/capabilities)

- Operator doc section that describes each provider role, how the role is selected from the pool, and what reordering the pool implies.
- Examples showing minimal pool shapes for single-provider, escalation-enabled, and classifier-enabled deployments.
- Explicit production defaults for log level in the Dockerfile and compose files, with verbose LLM logging clearly marked as diagnostic-only.
- Startup warning or gate when verbose LLM logging is enabled outside a development indicator; wording agreed with the operator docs.

## Success criteria

- Operator reading the new section can reproduce the mapping between their pool and the roles without reading the source.
- Production Docker artefacts do not set verbose LLM logging; searching them for the verbose value returns nothing.
- Startup clearly indicates when verbose LLM logging is active and why it is risky.
- Full quality gate passes on the change set.

## Traceability

- **Scope:** Reliability and security focus in [scope.md](../../scope.md).
- **Strategy:** Manual and documentation review in [strategy.md](../../strategy.md) §2.3.
- **Related:** Recommendations §10.3 and §10.10, risks R5 and R13 in [pa-architecture-review.md](../../pa-architecture-review.md); controls in [threat-model.md](../../threat-model.md).
