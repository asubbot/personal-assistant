# EP-029 — Implementation plan

## Goal

Ship optional observability HTTP (health + readiness), readiness composition aligned with `paApplication` startup, structured lifecycle logs for memory jobs / jobs runtime / tool index build, and operator documentation.

## Task order

1. **Config schema** — Add `ObservabilityHTTPConfig` to `internal/config`; validate paths, listen address non-empty, distinct paths; add positive/negative test fixtures (`REQ-29.001`, `REQ-29.006`, `AC-29.001`, `AC-29.006`).
2. **Lifecycle helper** — Add `internal/lifecyclelog` with `Info`/`Error` (`REQ-29.005`, `AC-29.005`).
3. **Instrument workers** — `internal/memoryjob` per-job boundaries; `cmd/pa/jobs_runtime.go` init; `cmd/pa/main.go` tool index goroutine duration + `LogBuildOutcome` signature (`REQ-29.005`).
4. **Readiness engine** — `cmd/pa/readiness.go`: `evalReadiness`, optional LLM probe, vector/tool/jobs/memory checks (`REQ-29.003`, `REQ-29.004`, `AC-29.003`, `AC-29.004`).
5. **HTTP surface** — `cmd/pa/observability_http.go`: handler mux + background `http.Server`; integrate in `runServer` after handler wiring; graceful shutdown on process context (`REQ-29.002`, `REQ-29.003`, `AC-29.002`).
6. **Tests** — `cmd/pa/ep029_observability_test.go` with `httptest` (`AC-29.002`, `AC-29.003`).
7. **Documentation** — `docs/observability-http.md`, index + configuration cross-links (`REQ-29.007`, `AC-29.007`).
8. **Verification** — `make check`; `make build && ./bin/validate EP-029` for release-style validation (`REQ-29.008`, `AC-29.008`; automated path includes `TestEP029_validateCommandExitZero`).

## Verification commands

```bash
make check
make build && ./bin/validate EP-029
```

## Notes

- Stages 7 (design review) and 10 (code review) follow pipeline delegation; artefacts updated before merge.
