# EP-022 — Implementation plan

**Pipeline:** [pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md) §2.1 stage 8

**Related artefacts:**
- [ep-scope.md](ep-scope.md)
- [ep-requirements.md](ep-requirements.md)
- [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- [ep-system-design.md](ep-system-design.md)
- [ep-system-design-review.md](ep-system-design-review.md) — stage 7 Pass on iteration 2
- [strategy.md](../../strategy.md)

## Goal

Deliver REQ-22.001–REQ-22.014 and AC-22.001–AC-22.011: introduce `internal/sqlitepragma` (DSN-based per-connection PRAGMA policy + `VerifyOnOpen`), rewire `internal/vector/sqlite` and `internal/jobs` to use it, add unified `http_timeout` config fields for LLM / embedding / web tools with fail-fast validation, add a `-race` concurrent-write test across both SQLite files, and update operator docs. **Critical path:** sqlitepragma → config → store call-sites → HTTP clients → test → docs.

## Tasks (order)

### 1. New shared package `internal/sqlitepragma`

- [ ] **1.1 Create `internal/sqlitepragma` package skeleton**
  - Add `internal/sqlitepragma/policy.go` with exported `Policy` struct: `JournalMode string`, `BusyTimeout time.Duration`, `Synchronous string`, `ForeignKeys bool`. Stdlib-only imports.
  - _Requirements:_ [REQ-22.001](ep-requirements.md#database-reliability), [REQ-22.002](ep-requirements.md#database-reliability)
  - _Acceptance Criteria:_ —
  - **Verification:** `go build ./internal/sqlitepragma` passes.

- [ ] **1.2 Implement `BuildDSN`**
  - `func BuildDSN(path string, p Policy) (string, error)`: validates each field (journal_mode / synchronous non-empty, busy_timeout > 0); builds a `file:<path>?_journal_mode=…&_busy_timeout=<ms>&_synchronous=…&_foreign_keys=on|off` DSN using `net/url`.
  - _Requirements:_ [REQ-22.001](ep-requirements.md#database-reliability), [REQ-22.002](ep-requirements.md#database-reliability), [REQ-22.003](ep-requirements.md#database-reliability), [REQ-22.004](ep-requirements.md#database-reliability), [REQ-22.005](ep-requirements.md#database-reliability)
  - _Acceptance Criteria:_ [AC-22.001](ep-acceptance-criteria.md#ac-22001), [AC-22.002](ep-acceptance-criteria.md#ac-22002), [AC-22.003](ep-acceptance-criteria.md#ac-22003)
  - **Verification:** Table-driven unit test covers good policy, empty `JournalMode`, zero `BusyTimeout`, empty `Synchronous`.

- [ ] **1.3 Implement `VerifyOnOpen`**
  - `func VerifyOnOpen(ctx context.Context, db *sql.DB, p Policy) error`: opens a fresh connection via `db.Conn(ctx)`; runs `PRAGMA journal_mode;`, `PRAGMA busy_timeout;`, `PRAGMA synchronous;`, `PRAGMA foreign_keys;`; compares each value against `p`; returns a wrapped error naming the mismatched PRAGMA.
  - _Requirements:_ [REQ-22.001](ep-requirements.md#database-reliability)
  - _Acceptance Criteria:_ [AC-22.001](ep-acceptance-criteria.md#ac-22001), [AC-22.002](ep-acceptance-criteria.md#ac-22002), [AC-22.003](ep-acceptance-criteria.md#ac-22003)
  - **Verification:** Unit test on a temp SQLite file covers matching and mismatched pragma paths.

### 2. Config types and validation

- [ ] **2.1 Add typed reliability and timeout fields to config model**
  - In `internal/config` add `SQLiteReliabilityConfig` (maps to `Policy` — same four fields, `BusyTimeout` as string parsed to `time.Duration`).
  - Attach reliability blocks to the vector-store config and the jobs-store config in `config.json` schema.
  - Add `HTTPTimeout time.Duration` to `LLMProviderConfig`, `EmbeddingConfig`, and `WebToolsConfig`; JSON key `http_timeout` in all three.
  - _Requirements:_ [REQ-22.006](ep-requirements.md#outbound-http-timeouts), [REQ-22.007](ep-requirements.md#outbound-http-timeouts), [REQ-22.008](ep-requirements.md#outbound-http-timeouts)
  - _Acceptance Criteria:_ —
  - **Verification:** `go build ./...` passes.

- [ ] **2.2 Implement fail-fast validation**
  - Loader rejects (a) empty / zero / unparseable `http_timeout` for each client; (b) empty / zero `busy_timeout`; (c) empty `journal_mode` or `synchronous`; (d) `foreign_keys=true` on the vector-store block; (e) `foreign_keys=false` on the jobs-store block.
  - Error messages name the JSON field path and the rejected value.
  - _Requirements:_ [REQ-22.005](ep-requirements.md#database-reliability), [REQ-22.009](ep-requirements.md#outbound-http-timeouts), [REQ-22.010](ep-requirements.md#outbound-http-timeouts)
  - _Acceptance Criteria:_ [AC-22.007](ep-acceptance-criteria.md#ac-22007), [AC-22.008](ep-acceptance-criteria.md#ac-22008)
  - **Verification:** Table-driven config test covers each rejection case and a happy-path load.

- [ ] **2.3 Update shipped `.config/config.json` and `config.example.json` (if present)**
  - Add explicit `http_timeout` fields (`"60s"` for LLM, `"30s"` for embedding, `"30s"` for web tools) and reliability blocks (`journal_mode="WAL"`, `busy_timeout="5s"`, `synchronous="NORMAL"`, `foreign_keys` per store).
  - No hidden defaults: absent fields fail startup.
  - _Requirements:_ [REQ-22.009](ep-requirements.md#outbound-http-timeouts), [REQ-22.010](ep-requirements.md#outbound-http-timeouts)
  - _Acceptance Criteria:_ [AC-22.007](ep-acceptance-criteria.md#ac-22007)
  - **Verification:** `./bin/pa` starts with the updated config; `make check` passes.

### 3. Wire sqlitepragma into stores

- [ ] **3.1 Update `internal/vector/sqlite` to use `sqlitepragma`**
  - Change `NewWithTable` (and related constructors) to accept a `sqlitepragma.Policy`.
  - Build DSN via `sqlitepragma.BuildDSN(path, policy)`; open with `sql.Open("sqlite3", dsn)`; call `sqlitepragma.VerifyOnOpen`; close the `*sql.DB` and return a wrapped error on failure.
  - Loader must pass `ForeignKeys=false` here (enforced by 2.2).
  - _Requirements:_ [REQ-22.001](ep-requirements.md#database-reliability), [REQ-22.002](ep-requirements.md#database-reliability), [REQ-22.003](ep-requirements.md#database-reliability), [REQ-22.004](ep-requirements.md#database-reliability)
  - _Acceptance Criteria:_ [AC-22.001](ep-acceptance-criteria.md#ac-22001)
  - **Verification:** Existing vector-store tests pass; new test asserts PRAGMAs read back on a fresh connection.

- [ ] **3.2 Update `internal/jobs.Open` to use `sqlitepragma`**
  - Change `Open(path)` → `Open(path string, p sqlitepragma.Policy)`.
  - Build DSN; open; verify.
  - **Remove** the one-shot `PRAGMA foreign_keys = ON` currently in `initSchema` (`internal/jobs/store.go:106`) — the DSN now carries it and the policy requires it.
  - _Requirements:_ [REQ-22.001](ep-requirements.md#database-reliability), [REQ-22.005](ep-requirements.md#database-reliability)
  - _Acceptance Criteria:_ [AC-22.002](ep-acceptance-criteria.md#ac-22002)
  - **Verification:** Existing jobs tests pass; new test asserts `foreign_keys=ON` on a fresh connection.

### 4. Wire HTTP timeouts

- [ ] **4.1 Update `internal/llm` constructor signature**
  - Provider constructor accepts `httpTimeout time.Duration`; returns `errors.New("llm: http timeout must be positive")` when zero; builds `&http.Client{Timeout: httpTimeout}` (no more `defaultTimeout` constant).
  - _Requirements:_ [REQ-22.006](ep-requirements.md#outbound-http-timeouts), [REQ-22.010](ep-requirements.md#outbound-http-timeouts)
  - _Acceptance Criteria:_ [AC-22.004](ep-acceptance-criteria.md#ac-22004)
  - **Verification:** Unit test asserts `client.Timeout == configured`; zero timeout test hits error.

- [ ] **4.2 Update `internal/embedding` constructor signature**
  - Same pattern as 4.1.
  - _Requirements:_ [REQ-22.007](ep-requirements.md#outbound-http-timeouts), [REQ-22.010](ep-requirements.md#outbound-http-timeouts)
  - _Acceptance Criteria:_ [AC-22.005](ep-acceptance-criteria.md#ac-22005)
  - **Verification:** Unit test mirrors 4.1.

- [ ] **4.3 Update `cmd/pa/main.go::registerWebToolsIfEnabled`**
  - Replace `webHTTP` construction: take `http_timeout` from `WebToolsConfig.HTTPTimeout`; pass a non-zero `Timeout` to `&http.Client`; no change to per-operation `search.timeout_seconds` / `fetch.timeout_seconds` semantics.
  - _Requirements:_ [REQ-22.008](ep-requirements.md#outbound-http-timeouts)
  - _Acceptance Criteria:_ [AC-22.006](ep-acceptance-criteria.md#ac-22006)
  - **Verification:** Focused test exercising the composition root asserts non-zero client timeout.

- [ ] **4.4 Propagate new constructor args at the composition root**
  - Wire config-loaded `HTTPTimeout` values into LLM and embedding providers in `cmd/pa/main.go`.
  - _Requirements:_ [REQ-22.006](ep-requirements.md#outbound-http-timeouts), [REQ-22.007](ep-requirements.md#outbound-http-timeouts)
  - _Acceptance Criteria:_ [AC-22.004](ep-acceptance-criteria.md#ac-22004), [AC-22.005](ep-acceptance-criteria.md#ac-22005)
  - **Verification:** `go build ./...`; startup with updated config succeeds.

### 5. Concurrent-write integration test

- [ ] **5.1 Add test under `internal/reliability` (new test-only package)**
  - Test function opens a real vector store and a real jobs store at `t.TempDir()` paths using the new `Policy`.
  - Spawns four goroutines per [ep-system-design.md §Testing strategy](ep-system-design.md#testing-strategy) table: summarization writer → `vec_summaries`, handler turn writer → `vec_turns`, tool index rebuilder → `vec_tools`, jobs runtime → jobs DB.
  - Uses a deterministic embedding stub returning a fixed vector; no real LLM call.
  - Wall-clock budget ≤ 5 s; asserts no `SQLITE_BUSY` / `database is locked` from any goroutine.
  - _Requirements:_ [REQ-22.013](ep-requirements.md#testing)
  - _Acceptance Criteria:_ [AC-22.010](ep-acceptance-criteria.md#ac-22010)
  - **Verification:** `go test -race ./internal/reliability/...` passes locally.

### 6. Operator documentation

- [ ] **6.1 Document PRAGMA policy and single-writer expectation**
  - In `docs/configuration.md` (or a sibling `docs/reliability.md` if cleaner), add a short section listing, per SQLite file (vector + jobs): path, required PRAGMA fields, recommended values, and the single-writer expectation note.
  - Document the operator rule: `foreign_keys=true` jobs / `false` vector.
  - _Requirements:_ [REQ-22.011](ep-requirements.md#operator-documentation)
  - _Acceptance Criteria:_ [AC-22.009](ep-acceptance-criteria.md#ac-22009)
  - **Verification:** Manual read-through; links render.

- [ ] **6.2 Document HTTP timeout fields**
  - Same docs page: list each `http_timeout` field, which client it governs, and the recommended default (`"60s"` LLM, `"30s"` embedding, `"30s"` web tools). Note that missing / zero values fail startup.
  - _Requirements:_ [REQ-22.012](ep-requirements.md#operator-documentation)
  - _Acceptance Criteria:_ [AC-22.009](ep-acceptance-criteria.md#ac-22009)
  - **Verification:** Manual read-through.

### 7. Checkpoints

- [ ] **7.1 Intermediate checkpoint after Task 3** — Run `go test ./internal/sqlitepragma/... ./internal/vector/... ./internal/jobs/... -race`. Fix before moving on. _Requirements:_ —; _Acceptance Criteria:_ —.
- [ ] **7.2 Intermediate checkpoint after Task 4** — Run `go build ./...`; smoke-start `./bin/pa` with updated config. _Requirements:_ —; _Acceptance Criteria:_ —.
- [ ] **7.3 Final checkpoint — `make check`**
  - `make check` must exit zero.
  - _Requirements:_ [REQ-22.014](ep-requirements.md#testing)
  - _Acceptance Criteria:_ [AC-22.011](ep-acceptance-criteria.md#ac-22011)
  - **Verification:** Command output captured in commit log.

## Non-goals

- Telegram long-poll HTTP behaviour (owned by `go-telegram/bot`; out of scope per [ep-scope.md](ep-scope.md)).
- SSH transport timeouts (owned by `internal/noderunner`; out of scope).
- Schema migrations; business-logic behaviour; per-operation context-based timeouts already in place for web tools.
