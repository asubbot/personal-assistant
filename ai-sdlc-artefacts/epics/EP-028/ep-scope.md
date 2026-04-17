# Epic scope — EP-028 Per-user rate limiting and tier-aware tool round caps

| Field | Content |
|-------|---------|
| **ID** | EP-028 |
| **Status** | NEW |
| **Title** | Per-user rate limiting and tier-aware tool round caps |
| **Description** | Add configurable per-user rate limits on inbound messages and on tool rounds, and make the maximum tool round count a per-tier setting, so an allowed user cannot burn unbounded provider and node resources from a single message. |
| **First version date** | 2026-04-17 |

## Glossary

- **Rate limit**: a bounded-rate policy applied per user identifier.
- **Tool round**: one request-to-LLM plus subsequent tool execution cycle.
- **Tier-aware cap**: a maximum tool round count selected from configuration based on the resolved tier.

## Scope (features/capabilities)

- Configuration describes per-user rate limits for inbound messages and for tool rounds, with fail-fast validation at startup.
- Configuration describes per-tier tool round caps; the simple and full-lite tiers accept lower defaults than full.
- Rate limit breaches produce a deterministic user-visible reply without calling any LLM or tool.
- The new limits are observable through structured log events for operator analytics.

## Success criteria

- With rate limits configured, a burst from one user produces deterministic reply messages and no LLM or tool calls beyond the limit.
- The per-tier cap strictly bounds the tool loop for each tier according to configuration.
- All new settings are explicit in configuration; no hidden defaults at load.
- Full quality gate passes on the change set.

## Traceability

- **Scope:** Security focus in [scope.md](../../scope.md).
- **Strategy:** Security check alignment in [strategy.md](../../strategy.md) §2.1.
- **Related:** Recommendations §10.5 and §10.11, risk R2 in [pa-architecture-review.md](../../pa-architecture-review.md); denial-of-service residuals in [threat-model.md](../../threat-model.md) §7.
