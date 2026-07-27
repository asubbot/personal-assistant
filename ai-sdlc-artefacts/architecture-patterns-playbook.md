---
artefact: architecture-patterns-playbook
status: draft
source_of_truth: false
updated_at: 2026-07-27
---

# Architecture patterns playbook (PersonalAssistant)

**Purpose:** Product-local defaults for consulting the advisory catalog
`reference/architecture-patterns/` (ai-sdlc checkout; pin in `ai-sdlc.version`).
This file is **not** a substitute for card `when` / `when_not` / `kiss_default`.
Catalog cards stay generic; this playbook only says what usually applies to **this**
single-process personal assistant.

**Related:** [pa-architecture-review.md](pa-architecture-review.md) §11,
[threat-model.md](threat-model.md), [docs/architecture-ru.md](../docs/architecture-ru.md).

## How to use (agents and humans)

1. List architecturally significant decisions (ASD) for the change or epic.
2. For each ASD, check the tables below for a **default stance**, then open 1–3
   matching cards in the catalog and read `when_not` / `kiss_default` first.
3. Record in epic `ep-system-design.md` **Design decisions**:
   `architecture-pattern: <id>` or `architecture-pattern: n/a — <reason>`,
   plus chosen / rejected / why and upstream `https://` links.
4. Do not add relative links from artefacts into `ai-sdlc/`.

## Usually consider (PA)

| Pattern id | Default stance | Why for PA |
|------------|----------------|------------|
| `module-boundaries` | Adopt / maintain | Enforced by `scripts/check-module-boundaries.sh`; wire only in `cmd/pa` |
| `authn-boundary` | Partial adopt | Telegram allowlist + SSH/tool gates; not multi-tenant authn |
| `sync-vs-async` | Partial | Sync user turn; async jobs init, memoryjob, index builds |
| `retry-and-timeouts` | Partial | HTTP timeouts, LLM transport fallback, memoryjob retries |
| `health-liveness-readiness` | Adopt when ops HTTP enabled | EP-029 `observability_http`; null = disabled |
| `idempotency` | Partial | Turn SHA dedup; jobs overlap policies; no end-to-end request keys |

## Usually n/a or declined (PA)

| Pattern id | Stance | One-line reason |
|------------|--------|-----------------|
| `rate-limiting` | Declined | EP-028 canceled — single trusted user; revisit if multi-user |
| `circuit-breaker` | n/a | Transport fallback + timeouts suffice for one process |
| `bulkhead` | n/a | Single process; isolate via allowlists and tool caps instead |
| `transactional-outbox` | n/a | No broker; local FS + SQLite |
| `publisher-subscriber` | n/a | No broker fan-out |
| `saga-or-compensating` | n/a | No multi-service commit choreography |
| `dead-letter-queue` | n/a | No message broker; poison handling is logs + retries |
| `caching` | n/a as subsystem | Session window + vector retrieval only |
| `strangler-fig` | n/a | Not replacing a legacy dual-stack system |

## After an epic changes stance

When an epic adopts or rejects a pattern differently from this playbook, add one
bullet under **Playbook deltas** below (epic id, pattern id, new stance, why)
and update the tables in a follow-up commit.

## Playbook deltas

- (none yet)
