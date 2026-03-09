# EP-104: Testing Coverage and Acceptance Criteria

**Date:** 2026-03-09  
**Epic:** EP-104  
**Source:** PROMPT-009 (QA Engineer coverage planning); AC text applied in Spexus.

---

## 1. Summary

- **Acceptance criteria:** 27 AC (AC-1274–AC-1300) across 15 user stories; all in Gherkin (Given/When/Then). Four AC were updated in Spexus for stricter Gherkin (AC-1285, AC-1292, AC-1298, AC-1300).
- **Testing:** Each AC is assigned to one or more test levels (unit / integration / e2e). Pyramid: more unit, fewer integration, fewest e2e.

---

## 2. Applied AC Updates (in Spexus)

The following AC were updated in Spexus to the text below.

### AC-1285 (US-407)

**Given** the designated memory directory contains markdown files in the defined structure, **When** the assistant reads long-term memory, **Then** content is read from that directory and structure.

### AC-1292 (US-411)

**Given** the log destination is configured but unavailable (e.g. path not writable or disk full), **When** the logging subsystem attempts to write a log entry, **Then** the system handles the error (e.g. fail-safe or fallback) according to documented behaviour.

### AC-1298 (US-415)

**Given** the codebase, **When** an architect or developer reviews the module boundaries, **Then** ingestion adapters (e.g. Telegram), core, memory store, vector index, LLM abstraction, scheduler, and tools are clearly separated so that replacing or extending one part does not require a full redesign.

### AC-1300 (US-416)

**Given** the versioned state feature is implemented or in design, **When** the operator or developer consults the documentation, **Then** the exact set of tracked paths is documented or explicitly marked TBD until research is done.

---

## 3. Testing Strategy per AC

| AC     | User Story | Recommended test level(s) | Notes |
|--------|------------|---------------------------|--------|
| AC-1274 | US-402 | Integration, E2E | Happy path: mock Telegram → core → reply; E2E with test bot or mock API. |
| AC-1275 | US-402 | Unit, Integration | Unit: validation for empty/oversized message; integration: adapter rejects or truncates. |
| AC-1276 | US-403 | Integration, E2E | Integration: container start with test config; E2E: run on x86_64 (or CI emulation). |
| AC-1277 | US-403 | Integration, E2E | Build image and run (e.g. in CI); E2E on DS220+ or equivalent. |
| AC-1278 | US-404 | Unit, Integration | Unit: config validator with invalid/missing fields; integration: core fails to start or reports error. |
| AC-1279 | US-404 | Integration | SSH client uses only config-driven host/user; integration test with mock SSH or test container. |
| AC-1280 | US-405 | Unit, Integration | Unit: allowlist check logic; integration: only allowlisted commands run (mock or test node). |
| AC-1281 | US-405 | Unit, Integration | Unit: denial when action not in allowlist; integration: no execution + log/report. |
| AC-1282 | US-406 | Unit, Integration | Unit: node config → single user; integration: SSH connection uses that user only (mock SSH). |
| AC-1283 | US-406 | Integration | Multiple nodes → each connection with correct dedicated user. |
| AC-1284 | US-407 | Unit, Integration | Unit: memory writer uses directory/structure; integration: write → files on disk in expected layout. |
| AC-1285 | US-407 | Unit, Integration | Unit: reader reads from configured path/structure; integration: read returns content from that structure. |
| AC-1286 | US-408 | Unit, Integration | Unit: indexer builds index from content; integration: index updated when memory changes. |
| AC-1287 | US-408 | Unit, Integration | Unit: search returns top-k/threshold; integration: query → relevant chunks from index. |
| AC-1288 | US-409 | Unit, Integration | Unit: provider selected from config; integration: LLM call goes to configured endpoint (mock). |
| AC-1289 | US-409 | Integration | Restart/hot-reload with new config → new provider used (mock or stub). |
| AC-1290 | US-410 | Unit, Integration | Unit: logger records request/response fields; integration: after LLM call, log entry present and parseable. |
| AC-1291 | US-411 | Unit, Integration | Unit: log destination from config; integration: entries written to configured path/format. |
| AC-1292 | US-411 | Unit, Integration | Unit: error handling when write fails; integration: unavailable destination → documented behaviour. |
| AC-1293 | US-412 | Unit, Integration | Unit: scheduler triggers at time/interval; integration: task runs when schedule fires (mock time if needed). |
| AC-1294 | US-412 | Unit, Integration | Unit: task filtered by security model; integration: violating task not executed, log/report. |
| AC-1295 | US-413 | Unit, Integration | Unit: tool registry and single contract; integration: core invokes tool with validated input, gets result. |
| AC-1296 | US-413 | Unit, Integration | Unit: schema validation rejects invalid input; integration: core returns error, tool not run. |
| AC-1297 | US-414 | Integration | Add node/tool in config, restart (or hot-reload) → new entity loaded (no image rebuild). |
| AC-1298 | US-415 | Manual / Static | Architecture review (checklist or static layout); optional: module-boundary tests or dependency rules. |
| AC-1299 | US-416 | Integration | Enable versioned state, change config/memory → commits (or equivalent) in repo. |
| AC-1300 | US-416 | Manual / Static | Docs review: tracked paths documented or TBD. |

---

## 4. Test Pyramid Summary

| Level        | Count (AC covered) | Focus |
|-------------|--------------------|--------|
| Unit        | ~18 AC             | Validators, allowlist, schema, indexer, logger, scheduler, config. |
| Integration | ~25 AC             | Core + adapters, SSH, memory, LLM mock, scheduler, tools, git. |
| E2E         | 2–4 AC             | Telegram flow (AC-1274), Docker run (AC-1276, AC-1277). |
| Manual/Static | 2 AC (1298, 1300) | Architecture and documentation checks. |
