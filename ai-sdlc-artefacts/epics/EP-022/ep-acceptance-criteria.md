# EP-022 — Acceptance criteria

Introduction: Testable conditions for EP-022 reliability hardening — explicit SQLite PRAGMA policy on every local store, bounded operator-configurable timeouts on every outbound HTTP client, fail-fast rejection of zero timeout, operator documentation, and a concurrent-write test under the race detector. Traceability to [ep-requirements.md](ep-requirements.md).

## Acceptance criteria index

| AC ID | REQ | Summary |
|-------|-----|---------|
| [AC-22.001](#ac-22001) | [REQ-22.001](ep-requirements.md#database-reliability), [REQ-22.002](ep-requirements.md#database-reliability) | Vector store opens apply WAL and full PRAGMA policy |
| [AC-22.002](#ac-22002) | [REQ-22.001](ep-requirements.md#database-reliability), [REQ-22.002](ep-requirements.md#database-reliability), [REQ-22.005](ep-requirements.md#database-reliability) | Jobs store opens apply WAL, busy_timeout, synchronous, foreign_keys |
| [AC-22.003](#ac-22003) | [REQ-22.003](ep-requirements.md#database-reliability), [REQ-22.004](ep-requirements.md#database-reliability) | busy_timeout and synchronous reflect configured values |
| [AC-22.004](#ac-22004) | [REQ-22.006](ep-requirements.md#outbound-http-timeouts) | LLM provider HTTP client carries configured bounded timeout |
| [AC-22.005](#ac-22005) | [REQ-22.007](ep-requirements.md#outbound-http-timeouts) | Embedding provider HTTP client carries configured bounded timeout |
| [AC-22.006](#ac-22006) | [REQ-22.008](ep-requirements.md#outbound-http-timeouts) | Web tools HTTP client carries configured bounded timeout |
| [AC-22.007](#ac-22007) | [REQ-22.009](ep-requirements.md#outbound-http-timeouts) | Invalid timeout string fails config load with explicit error |
| [AC-22.008](#ac-22008) | [REQ-22.010](ep-requirements.md#outbound-http-timeouts) | Zero timeout fails startup with explicit error |
| [AC-22.009](#ac-22009) | [REQ-22.011](ep-requirements.md#operator-documentation), [REQ-22.012](ep-requirements.md#operator-documentation) | Operator docs describe PRAGMA policy, single-writer expectation, HTTP timeouts |
| [AC-22.010](#ac-22010) | [REQ-22.013](ep-requirements.md#testing) | Concurrent-write test passes under race detector with no busy error |
| [AC-22.011](#ac-22011) | [REQ-22.014](ep-requirements.md#testing) | `make check` succeeds on the change set |

## Acceptance criteria

### AC-22.001

**AC-22.001** (Trace: REQ-22.001, REQ-22.002)

Given an initialized vector SQLite store opened at a temporary path by `internal/vector/sqlite`  
When a test queries `PRAGMA journal_mode` and `PRAGMA synchronous` on a fresh connection from that store  
Then `journal_mode` returns `wal` and `synchronous` returns the configured mode (defaulting to the value declared for the vector store in configuration).

### AC-22.002

**AC-22.002** (Trace: REQ-22.001, REQ-22.002, REQ-22.005)

Given an initialized jobs store opened at a temporary path by `internal/jobs`  
When a test queries `PRAGMA journal_mode`, `PRAGMA busy_timeout`, `PRAGMA synchronous`, and `PRAGMA foreign_keys` on a fresh connection from that store  
Then `journal_mode=wal`, `busy_timeout` equals the configured value in milliseconds, `synchronous` equals the configured mode, and `foreign_keys=1`.

### AC-22.003

**AC-22.003** (Trace: REQ-22.003, REQ-22.004)

Given configuration that sets `busy_timeout=7500ms` and `synchronous=NORMAL` for a local SQLite store  
When that store is opened via its package constructor  
Then `PRAGMA busy_timeout` returns `7500` and `PRAGMA synchronous` returns `1` (NORMAL) on a fresh connection.

### AC-22.004

**AC-22.004** (Trace: REQ-22.006)

Given a configuration with `llm_providers[0].http_timeout="45s"`  
When the process constructs the LLM provider HTTP client through `internal/llm`  
Then the resulting `*http.Client` exposes `Timeout == 45s` (asserted by a unit test that accesses the constructed client).

### AC-22.005

**AC-22.005** (Trace: REQ-22.007)

Given a configuration with `embedding.http_timeout="20s"`  
When the process constructs the embedding provider HTTP client through `internal/embedding`  
Then the resulting `*http.Client` exposes `Timeout == 20s`.

### AC-22.006

**AC-22.006** (Trace: REQ-22.008)

Given a configuration with `web_tools.http_timeout="30s"` and `web_tools.enabled=true`  
When the process constructs the web tools HTTP client through the composition root  
Then the resulting `*http.Client` passed to `NewWebSearchTool` and `NewWebFetchTool` exposes `Timeout == 30s`.

### AC-22.007

**AC-22.007** (Trace: REQ-22.009)

Given a configuration file with `llm_providers[0].http_timeout="not-a-duration"`  
When `config.Load` runs at startup  
Then `config.Load` returns an error whose message names the field path and the rejected value.

### AC-22.008

**AC-22.008** (Trace: REQ-22.010)

Given a configuration file with `web_tools.http_timeout="0s"`  
When `config.Load` runs at startup  
Then `config.Load` returns an error whose message names the `web_tools` client role and states that zero timeout is not allowed.

### AC-22.009

**AC-22.009** (Trace: REQ-22.011, REQ-22.012)

Given the updated operator documentation under `docs/` on the epic branch  
When a reader searches the docs for the PRAGMA policy, the single-writer expectation, and the HTTP timeout fields  
Then a section exists that lists `journal_mode`, `busy_timeout`, `synchronous`, and `foreign_keys` for each local SQLite store, the single-writer expectation per file, and the HTTP timeout configuration fields for LLM, embedding, and web tools clients with their defaults.

### AC-22.010

**AC-22.010** (Trace: REQ-22.013)

Given an integration test that opens a real vector store and a real jobs store at temporary paths and spawns three concurrent writer goroutines simulating the summarization worker, the conversation handler turn indexing, and the tool vector index build  
When the test runs under `go test -race` for a bounded duration  
Then the test completes without any `database is locked`, `SQLITE_BUSY`, or data race report.

### AC-22.011

**AC-22.011** (Trace: REQ-22.014)

Given the repository on the epic branch after implementation  
When an operator runs `make check` from the repository root  
Then the command exits with status zero.
