# Strategy — PersonalAssistant


This document combines **delivery strategy** and **test strategy** for the project. It is aligned with [scope.md](scope.md). Epic-level details are produced in stage 3 (epic planning).

---

## 1. Delivery strategy

### 1.1 Target increments

| Feature | Version | Definition |
|-------|---|---------|
|**MVP** | 0.01 |A working personal assistant the user can talk to via Telegram. It runs in Docker on Synology DS220+, uses long-term memory and optional remote nodes, supports multiple LLM backends, and is built so we can evolve it without breaking the core.|
| TBD | |

### 1.2 Success criteria (high level)

- After each increment: existing behaviour still works; new behaviour is testable (unit and/or integration).
- By end of 0.1: one E2E path — user message in Telegram → reply using memory and (where applicable) node/tool; runnable on target platform.

---

## 2. Test strategy

### 2.1 How we test

- **Traceability:** Every acceptance criterion (defined later) is covered by at least one test level.
- **Levels:** Unit, Integration, E2E, Manual (definitions and assignment in epic/story stages).
- **Security:** Explicit checks for allowlists, dedicated user, no secrets in LLM context or logs; prompt-injection / exfiltration scenarios.
- **Out of scope for 0.1:** Performance, load, stress testing; broad platform matrix (E2E target: x86_64 / DS220+).

### 2.2 Test pyramid (target shape)

| Level | Definition |
|-------|------------|
| **Unit** | Majority of tests. Single components in isolation, mocks, no real I/O. Fast feedback. Examples: config validation, allowlist, memory indexer, scheduler logic, context builder.|
| **Integration** | Meaningful subset. Several components together; may use real I/O or mocks. Examples: core + adapters, SSH + allowlist, memory + vector, tools + scheduler, logging.|
| **E2E** | Few. Full flow: Telegram → core → LLM (or mock) → reply; optional memory and node/tool. Docker run for deploy check.|
| **Manual** | Where automation is not practical: node/CLI checks, docs and architecture review, optional log inspection.|

Pyramid: **many unit → fewer integration → few E2E → manual as needed**. Exact mapping (which AC at which level) is defined in acceptance-criteria and implementation-plan stages.

### 2.3 Manual testing (high level)

- Verify node access and allowlist behaviour via CLI or documented steps.
- Optional: check LLM log format and completeness.
- Review architecture and documentation where required by acceptance criteria.
