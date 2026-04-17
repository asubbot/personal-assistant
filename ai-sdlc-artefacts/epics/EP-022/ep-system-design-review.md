# Architecture Review — EP-022 Reliability hardening for local SQLite stores and outbound HTTP timeouts

**Reviewer:** AI Code Reviewer (stage 7 delegated)

---

## Review iteration 1

**Review date:** 2026-04-17
**Stage 7 iteration:** 1 of max 5
**Document reviewed:** [ep-system-design.md](ep-system-design.md)
**Iteration summary — open counts:** Blocker: 0 | Major: 2 | Medium: 1 | Minor: 4
**Gate:** Fail (Major > 0)

### Overall assessment

The design is appropriately narrow for a reliability epic and maps cleanly to the requirements: a small shared `internal/sqlitepragma` seam, typed HTTP timeout fields at the composition root, and fail-fast config validation. Two gaps prevent passing this gate: the web-tools HTTP timeout config key name contradicts AC-22.006, and the stated PRAGMA-application contract (`ApplyOnOpen(db *sql.DB, …)` "applied once per `*sql.DB`") does not obviously satisfy REQ-22.001 ("on every connection open") for per-connection PRAGMAs under Go's `database/sql` pool.

**Verdict:** Fail gate — return to stage 6.

### Strengths

- Clear, single-sourced PRAGMA policy via a tiny new package (`internal/sqlitepragma`) with no config coupling — matches KISS and keeps both callers testable.
- Explicit, typed `time.Duration` fields at the composition root (`internal/llm`, `internal/embedding`, `cmd/pa`) replace the hard-coded `defaultTimeout` constants and `Timeout: 0` web-tools client flagged in pa-architecture-review §8.6 / §10.7.
- Correctly upholds the repository principle "no hidden defaults at load" by keeping defaults documentary only (loader rejects missing/empty/zero values).
- Fail-fast error model is specific: config loader names the field path and rejected value; provider constructors refuse zero timeouts; store constructors close the already-open `*sql.DB` on PRAGMA failure.
- Scope boundaries are stated (Telegram long-poll, SSH transport out of scope) which matches scope and threat-model §7.
- Testing strategy exercises writer contention on real SQLite files under `-race`, directly tracing AC-22.010.

### Issues and recommendations

#### Blocker

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

#### Major

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| M1 | **Config key for web tools HTTP timeout contradicts AC-22.006.** Design Architecture §2 and Data models declare `web_tools.http_client_timeout` (`WebToolsConfig.HTTPClientTimeout`), but [AC-22.006](ep-acceptance-criteria.md#ac-22006) specifies `web_tools.http_timeout="30s"`. A test written against AC-22.006 will fail against the designed schema. | §Architecture "Explicit HTTP timeout fields in configuration"; §Data models `WebToolsConfig`; contrast with [AC-22.006](ep-acceptance-criteria.md#ac-22006). `internal/config/webtools.go` already carries per-operation `search.timeout_seconds` / `fetch.timeout_seconds`, so a distinct `http_timeout` is not name-collision-critical. | Either rename the config key and Go field to `http_timeout` / `HTTPTimeout` to match the AC (and align with `llm_providers[i].http_timeout`, `embedding.http_timeout` — consistent name for the same concept), or update AC-22.006 and REQ-22.008 trace text to `http_client_timeout` if there is a deliberate naming reason. Pick one and make all three places (design, AC, requirements notes) agree. |
| M2 | **PRAGMA-per-connection semantics are under-specified and may not satisfy REQ-22.001.** The design states `ApplyOnOpen(db *sql.DB, p Policy)` "applies it once per `sql.DB` after `sql.Open`". In the `mattn/go-sqlite3` driver used by both `internal/vector/sqlite/store.go` and `internal/jobs/store.go`, `busy_timeout`, `foreign_keys`, and `synchronous` are per-connection settings. A single `db.Exec("PRAGMA …")` runs on whatever pooled connection the driver hands out and is **not** replayed when the pool opens further connections. REQ-22.001 says "on every connection open", and AC-22.001 / AC-22.002 explicitly query PRAGMAs on a **fresh connection**. | [REQ-22.001](ep-requirements.md#database-reliability), [AC-22.001](ep-acceptance-criteria.md#ac-22001), [AC-22.002](ep-acceptance-criteria.md#ac-22002), [AC-22.003](ep-acceptance-criteria.md#ac-22003); current code at `internal/vector/sqlite/store.go:63` (`sql.Open("sqlite3", dbPath)`) and `internal/jobs/store.go:83` (plus `PRAGMA foreign_keys = ON` at line 106 run once). | Commit to a mechanism that makes the policy actually per-connection. Two KISS options consistent with the repo stack: (a) encode PRAGMAs as DSN query params supported by `mattn/go-sqlite3` — `_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&_foreign_keys=on` — which the driver applies on every new connection; or (b) constrain the pool to a single writer connection (`db.SetMaxOpenConns(1)`) and document the read path, then apply PRAGMAs once. Whichever is chosen, state it explicitly in §Architecture and §Components (signature of the helper: does it take a DSN builder, a connection hook, or a `*sql.DB` where the caller has already configured the pool?). Also clarify whether `WAL` is sticky-once or reapplied (it is DB-level and persistent, unlike `busy_timeout`). |

#### Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| D1 | **Concurrent-Write Test coverage is narrower than REQ-22.013.** REQ-22.013 names three writer paths: summarization worker, conversation handler, **tool vector index build**. §Testing strategy lists "summarization worker, turn indexing, tool vector build" but does not say which SQLite files are touched by each or how the test doubles LLM/embedding without losing real writer contention. The vector store and tools live in the same file, jobs in another — the test must exercise both files to match REQ-22.013 and AC-22.010. | [REQ-22.013](ep-requirements.md#testing), [AC-22.010](ep-acceptance-criteria.md#ac-22010); design §Testing strategy bullet 2. | Spell out, in one sentence each, (1) which goroutine writes to which file (vector DB vs jobs DB), (2) the exact stubs for embedding/LLM so writer contention is preserved while calls are deterministic, and (3) the bounded wall-clock budget for the test. A 3-line table under §Testing strategy is enough. |

#### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| N1 | **Structural: no explicit "Requirement traceability" table.** Skill §Step 2 expects one; traceability is currently inline anchors throughout §Overview and §Testing. Coverage is present for all 14 REQs, so this is cosmetic. | Skill 07 §Step 2; current design has `### Module boundaries` but no dedicated traceability table. | Add a small REQ → component/section table at the end of the design. |
| N2 | **Structural: no "Risks and trade-offs" section.** Skill §Step 2 lists it. | Skill 07 §Step 2; pa-architecture-review §9 R6/R7. | One short subsection listing residual risks (busy_timeout too large, WAL sidecar files, SetMaxOpenConns reducing read throughput). |
| N3 | **`ForeignKeys` field on vector-store policy is dead config.** §Data models notes "ignored for vector store". | §Data models `SQLiteStoreReliability`; REQ-22.005 is jobs-only. | Drop the field from the shared `Policy` or require it explicitly per store — avoid the "ignored" path. |
| N4 | **Operator-doc default values vs "no hidden defaults at load".** Struct comments say `// "WAL" by default`. | §Data models struct comments. | Replace with `// required; operator docs recommend "WAL"`. |

### NFR coverage

| NFR | Coverage | Status |
|-----|----------|--------|
| REQ-22.009 fail-fast on unparseable/empty HTTP timeout | §Error handling bullet 1; §Components `internal/config` | OK |
| REQ-22.010 reject zero timeout | §Error handling bullets 1 and 3 | OK |
| REQ-22.011 docs cover PRAGMA + single-writer | §Components `docs/` row | OK (content list deferred to stage 8) |
| REQ-22.012 docs list timeouts + defaults | §Components `docs/` row; §Design decisions "Default values" | OK |
| REQ-22.014 `make check` passes | §Testing strategy "Quality gate" | OK |

### Project rules compliance

| Rule | Compliance |
|------|------------|
| KISS | OK (tiny helper package, no new abstractions beyond one policy struct) |
| Fail fast | OK for config; Needs work for DB open (dependent on M2 resolution) |
| Explicit JSON configuration (no hidden defaults at load) | OK stated in §Design decisions; Needs work in struct-comment wording (N4) |
| Security / no weakening | OK (no change to SSH, Telegram, or log redaction paths) |
| Testability | OK for unit layers; Needs work in concurrent-write test scope (D1) |

### Traceability (this iteration)

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- **Scope:** [ep-scope.md](ep-scope.md)
- **Source recommendation:** [pa-architecture-review.md](../../pa-architecture-review.md) §10.7

---

## Review iteration 2

**Review date:** 2026-04-17
**Stage 7 iteration:** 2 of max 5
**Document reviewed:** [ep-system-design.md](ep-system-design.md)
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0
**Gate:** Pass

### Overall assessment

The revised design resolves every iteration-1 finding. The PRAGMA policy now commits to a concrete per-connection mechanism — DSN query parameters consumed by `mattn/go-sqlite3` on every new pool connection, plus a fresh-connection `VerifyOnOpen` fail-fast self-check — which maps directly to REQ-22.001 and to the "fresh connection" assertions in AC-22.001 / AC-22.002. The web-tools HTTP timeout key is unified to `http_timeout`, aligning the schema with AC-22.006. The Concurrent-Write Test is now spelled out as a four-goroutine table with explicit writer paths, SQLite files, workloads, test doubles, and a bounded wall-clock budget (≤ 5 s), which traces cleanly to REQ-22.013 and AC-22.010. The document also adds the missing "Risks and trade-offs" and "Requirement traceability" sections required by the skill template, and struct comments are rephrased to "required; operator docs recommend …" consistent with the repo's no-hidden-defaults rule. The `ForeignKeys` field remains on the shared `Policy` but is now required per store (jobs=`true`, vector=`false`, fail-fast otherwise), which is a clean, KISS resolution of N3.

**Verdict:** Pass gate — proceed to stage 8 (implementation planning).

### Strengths

- Clear commitment to DSN-encoded PRAGMAs (`_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&_foreign_keys=on`) as the per-connection mechanism, with `VerifyOnOpen` as a documented fail-fast guard against driver regressions — directly satisfies REQ-22.001.
- Unified config key `http_timeout` across `llm_providers[i]`, `embedding`, and `web_tools`, with the pre-existing per-operation `search.timeout_seconds` / `fetch.timeout_seconds` kept distinct and untouched.
- Concurrent-Write Test table lists four goroutines with exact writer paths, target SQLite files (vector vs jobs), workloads, and stubs — traceability to REQ-22.013 and AC-22.010 is now self-evident.
- `internal/jobs.Open` explicitly drops the one-shot `PRAGMA foreign_keys = ON` from `initSchema` because the DSN now carries `_foreign_keys=on` on every connection — avoids redundant, misleading code after the change.
- `ForeignKeys` on the shared policy is required per store with a stated fail-fast rule (vector=`false`, jobs=`true`); no silent "ignored" field remains.
- New Risks and trade-offs section calls out operator-set large `busy_timeout` masking contention, WAL sidecar backup implications, and driver-dependency risk — each with a stated mitigation.
- `internal/sqlitepragma` module boundaries are explicit: stdlib-only imports, no `internal/config` dependency; callers pass a `Policy` value. Keeps the seam trivially fakeable and reusable.
- Requirement traceability table covers all 14 REQs with primary components and primary ACs.

### Issues and recommendations

#### Blocker

_None._

#### Major

_None._

#### Medium

_None._

#### Minor

_None._

### Iteration-1 finding resolution

| ID | Prior finding | Resolution in iteration 2 | Status |
|----|---------------|---------------------------|--------|
| M1 | `web_tools.http_client_timeout` contradicted AC-22.006 (`http_timeout`). | §Architecture and §Data models now declare `web_tools.http_timeout` and `WebToolsConfig.HTTPTimeout`, matching AC-22.006 and the unified key across LLM / embedding / web tools. | Resolved |
| M2 | PRAGMA-per-connection semantics under-specified; `ApplyOnOpen(*sql.DB)` did not match REQ-22.001. | Mechanism is now DSN query parameters (`BuildDSN(path, Policy) (string, error)`) which `mattn/go-sqlite3` applies on every new connection, plus a fresh-connection `VerifyOnOpen(ctx, db, p)` fail-fast check. Design Decisions section explicitly chooses this over `SetMaxOpenConns(1)` and cites the driver-supported query parameters. | Resolved |
| D1 | Concurrent-Write Test scope not spelled out (which file each goroutine touches, stubs, budget). | §Testing strategy now has a four-row table (goroutine / writer path / SQLite file / workload / stubs) covering vector and jobs files, with a stated ≤ 5 s wall-clock budget and explicit "no `database is locked` / `SQLITE_BUSY`" assertion. | Resolved |
| N1 | No dedicated Requirement traceability table. | §Requirement traceability table added at the end of the design, covering all 14 REQs → primary components → primary ACs. | Resolved |
| N2 | No Risks and trade-offs section. | §Risks and trade-offs section added with five entries and mitigations. | Resolved |
| N3 | `ForeignKeys` as "ignored" dead config on vector store. | Policy keeps the field but loader rejects other combinations: jobs MUST be `true`, vector MUST be `false`. | Resolved |
| N4 | Struct comments said `// "WAL" by default` implying hidden defaults. | §Data models comments rephrased to `// required; operator docs recommend "WAL"` / `"NORMAL"`, matching the explicit-JSON-config principle. | Resolved |

### Project rules compliance

| Rule | Compliance |
|------|------------|
| KISS | OK — DSN-query approach is the smallest change that satisfies REQ-22.001; one new stdlib-only package. |
| Fail fast | OK — config validation; `VerifyOnOpen` on fresh connection; provider constructors refuse zero timeouts. |
| Explicit JSON configuration | OK — "No loader-side defaults" stated; loader rejects empty/zero/unparseable and the bad `foreign_keys` combination. |
| Security / no weakening | OK — no change to SSH, Telegram long-poll, or log redaction paths. |
| Testability | OK — unit tests + real-DB concurrent-write test under `-race` with bounded budget. |

### Traceability (this iteration)

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- **Scope:** [ep-scope.md](ep-scope.md)
- **Prior iteration:** §Review iteration 1 above
- **Source recommendation:** [pa-architecture-review.md](../../pa-architecture-review.md) §10.7
