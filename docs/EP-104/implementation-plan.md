# Implementation Plan — EP-104 PersonalAssistant MVP

**Epic:** EP-104  
**Design:** [system-design.md](system-design.md)  
**Research:** [research.md](research.md) (MVI, iteration plan)  
**Requirements:** Spexus REQ-642–REQ-657; AC-1274–AC-1300  
**Testing reference:** [testing-coverage.md](testing-coverage.md)

Tasks are ordered for incremental progress; each step builds on the previous. Optional test sub-tasks are marked with `*` and can be skipped for a faster MVP.

---

## 1. Project skeleton and config

- [ ] 1. Set up Go module and package structure
  - Create `go.mod` (Go 1.26+), directories: `cmd/pa`, `internal/config`, `internal/telegram`, `internal/core`, `internal/memory`, `internal/vector`, `internal/llm`, `internal/scheduler`, `internal/tools`, `internal/ssh`, `internal/logging`
  - Minimal `cmd/pa/main.go` that loads config and exits
  - _Requirements: REQ-643, REQ-653_

- [ ] 1.1 Implement config load and validation
  - Define config struct (nodes: host, dedicated_user, auth, command_allowlist; LLM: type, endpoint, api_key_path; paths: memory_dir, log_path, vector_index_path; scheduled_tasks)
  - Load YAML/JSON from path; validate required fields and node/LLM/path consistency
  - On validation failure: log clear error and exit non-zero (do not start serving)
  - _Requirements: REQ-644, REQ-645_
  - _Validates: AC-1278_

- [ ]* 1.2 Write unit tests for config validation
  - Invalid host or missing authentication → validator returns error
  - Valid config → no error
  - _Validates: AC-1278_

- [ ] 2. Checkpoint — Ensure all tests pass, ask the user if questions arise.

---

## 2. Config and node security model

- [ ] 2.1 Implement per-node allowlist model
  - Per-node command/tool allowlist (patterns or regex from config); data structure and lookup
  - Function: given node ID and requested command/action, return allowed or denied
  - _Requirements: REQ-646_
  - _Validates: AC-1280, AC-1281_

- [ ] 2.2 Enforce dedicated SSH user per node
  - Node config exposes exactly one user identity per node; SSH client must use only that identity
  - No shared or alternate account for that node
  - _Requirements: REQ-654_
  - _Validates: AC-1282, AC-1283_

- [ ]* 2.3 Write unit tests for allowlist and dedicated user
  - Allowlist: only allowlisted commands return allowed; others denied
  - Dedicated user: node config yields single user; multi-node yields correct user per node
  - _Validates: AC-1280, AC-1281, AC-1282_

- [ ] 3. Checkpoint — Ensure all tests pass, ask the user if questions arise.

---

## 3. Telegram adapter and core (first conversation flow)

- [ ] 3.1 Implement LLM provider interface and one implementation
  - Interface: e.g. `Complete(ctx, messages, opts) (response, usage, err)`
  - One implementation: OpenAI-compatible HTTP or Ollama; provider and params from config
  - _Requirements: REQ-649_
  - _Validates: AC-1288, AC-1289_

- [ ] 3.2 Implement Telegram adapter (polling)
  - Use go-telegram/bot; config: bot token, optional allowed user_id list
  - Map incoming text messages to core input; send text replies from core output
  - _Requirements: REQ-642_
  - _Validates: AC-1274_

- [ ] 3.3 Implement minimal core orchestration
  - Single entry: receive user message → call LLM provider → return reply (no memory/vector/tools yet)
  - Wire Telegram adapter to core and LLM provider
  - _Requirements: REQ-642, REQ-649_

- [ ] 3.4 Message validation (empty / max length)
  - Reject or truncate empty message or message exceeding configured max length; clear behaviour documented
  - _Requirements: REQ-642_
  - _Validates: AC-1275_

- [ ]* 3.5 Write integration tests for Telegram → core → LLM → reply
  - Mock Telegram updates and LLM; assert reply returned within timeout
  - _Validates: AC-1274_

- [ ] 4. Checkpoint — Ensure all tests pass, ask the user if questions arise.

---

## 4. Memory store and vector index

- [ ] 4.1 Implement long-term memory store (markdown files)
  - Read/write markdown files under configured memory_dir; structure (e.g. by user, topic, date) from config
  - _Requirements: REQ-647_
  - _Validates: AC-1284, AC-1285_

- [ ] 4.2 Implement pluggable vector store interface and default implementation
  - Interface: add documents (with embeddings), search by query vector (top-k or threshold)
  - Default: chromem-go or vecgo; persist index to configured path where supported
  - _Requirements: REQ-648_
  - _Validates: AC-1286, AC-1287_

- [ ] 4.3 Wire memory and vector into core
  - On conversation: read relevant memory, optionally update memory; index content; semantic search and inject context into LLM call
  - _Requirements: REQ-647, REQ-648_

- [ ]* 4.4 Write unit and integration tests for memory and vector
  - Memory: write then read from same structure; reader uses configured path
  - Vector: index content, search returns relevant chunks
  - _Validates: AC-1284, AC-1285, AC-1286, AC-1287_

- [ ] 5. Checkpoint — Ensure all tests pass, ask the user if questions arise.

---

## 5. SSH client and nodes

- [ ] 5.1 Implement SSH client
  - Use golang.org/x/crypto/ssh; connect using credentials from validated node config only (one dedicated user per node)
  - Execute only allowlisted commands; exec-style args, no shell with untrusted input
  - _Requirements: REQ-645, REQ-646, REQ-654_
  - _Validates: AC-1279, AC-1282, AC-1283_

- [ ] 5.2 Integrate SSH into core
  - When a tool or flow requires node action: resolve node from config, check allowlist, run via SSH client
  - On connection/exec failure: log and report to core; no fallback to other users
  - _Requirements: REQ-645, REQ-646, REQ-654_

- [ ]* 5.3 Write integration tests for SSH (mock or test container)
  - Valid config → SSH uses config host/user only; allowlist blocks disallowed command
  - _Validates: AC-1279, AC-1280, AC-1281_

- [ ] 6. Checkpoint — Ensure all tests pass, ask the user if questions arise.

---

## 6. Scheduler and tools

- [ ] 6.1 Implement tool contract and registry
  - Interface: Name, Description, ParamsSchema, Run(ctx, params); registry at startup; config can enable/parameterise tools
  - _Requirements: REQ-651, REQ-652_
  - _Validates: AC-1295, AC-1296_

- [ ] 6.2 Implement scheduler (cron)
  - Use robfig/cron/v3; tasks from config (cron or @every); execution invokes registered tool or sends Telegram notification within security model
  - _Requirements: REQ-650_
  - _Validates: AC-1293, AC-1294_

- [ ] 6.3 Wire tools and scheduler into core
  - Core invokes tools via single contract (validate input, call Run); scheduler runs tasks that call tools or notify
  - _Requirements: REQ-650, REQ-651_

- [ ] 6.4 Add node/tool via config without image rebuild
  - New node or tool in config (or designated extension); after restart (or hot-reload if supported), new entity loaded
  - _Requirements: REQ-652_
  - _Validates: AC-1297_

- [ ]* 6.5 Write unit and integration tests for tools and scheduler
  - Tool: valid input → result; invalid input → validation error, tool not run
  - Scheduler: task at schedule runs; task that would violate security model does not run
  - _Validates: AC-1293, AC-1294, AC-1295, AC-1296_

- [ ] 7. Checkpoint — Ensure all tests pass, ask the user if questions arise.

---

## 7. LLM logging

- [ ] 7.1 Implement LLM logging subsystem
  - On each LLM call: write request (input messages, model params, request_id) and response (output, token counts, duration/model id) to configurable destination
  - Format: JSON Lines; configurable path or directory
  - _Requirements: REQ-655, REQ-656_
  - _Validates: AC-1290, AC-1291_

- [ ] 7.2 Handle unavailable log destination
  - When destination is configured but unavailable (e.g. path not writable): fail-safe or fallback per documented behaviour
  - _Requirements: REQ-656_
  - _Validates: AC-1292_

- [ ]* 7.3 Write unit tests for LLM logging
  - Log entry contains request and response fields; entries written to configured path; parseable format
  - _Validates: AC-1290, AC-1291, AC-1292_

- [ ] 8. Checkpoint — Ensure all tests pass, ask the user if questions arise.

---

## 8. Docker and deploy (DS220+)

- [ ] 8.1 Add Dockerfile and docker-compose
  - Multi-stage build; final image linux/amd64 (Alpine or distroless); volumes for config, memory, logs
  - Single core service; target Synology DS220+ (x86_64)
  - _Requirements: REQ-643_
  - _Validates: AC-1276, AC-1277_

- [ ] 8.2 Verify container start and one conversation
  - Container starts with test config; one message in → reply out (e.g. via test bot or curl if API exposed for tests)
  - _Requirements: REQ-643_
  - _Validates: AC-1276_

- [ ] 9. Checkpoint — Ensure all tests pass, ask the user if questions arise.

---

## 9. Architecture boundaries and versioned state (optional for MVP)

- [ ] 9.1 Document and enforce clear module boundaries
  - Ensure ingestion adapters (Telegram), core, memory, vector, LLM, scheduler, tools are in separate packages; no circular deps
  - _Requirements: REQ-653_
  - _Validates: AC-1298_

- [ ] 9.2 Versioned state (git-backed config/memory) — scope TBD
  - If in scope: use git repo in deployment/data dir to track config, memory, or other paths; document tracked paths or mark TBD
  - _Requirements: REQ-657_
  - _Validates: AC-1299, AC-1300_

---

## 10. Final checkpoint

- [ ] 10. Final checkpoint — Ensure all acceptance criteria are met by reviewing the code and running unit and integration tests, ask the user if questions arise.
  - **Validates:** AC-1274–AC-1300 (see [testing-coverage.md](testing-coverage.md)).
