---
artefact: ep-system-design
epic_id: EP-042
status: draft
updated_at: 2026-05-31
---

# EP-042 — Composition root refinement — System design

## Overview

Extract `cmd/pa/wire` package with `Build(cfg, configPath, logger) (Application, error)` where **`Application`** is an exported interface implemented by unexported `*paApplication` in the wire package (avoids cross-package unexported type issue). Thin `main.go`. Explicit jobs runtime state enum (initializing/ready/failed) aligned with readiness ([ep-scope.md](ep-scope.md)).

## Components

| Component | Change |
|-----------|--------|
| `cmd/pa/wire/build.go` | `Build`, move `buildPAInfrastructure`, LLM, classifier, handler wiring from `main.go` |
| `cmd/pa/main.go` | flags, load config, `wire.Build`, run, shutdown |
| `cmd/pa/jobs_runtime.go` | Document state enum; ensure messages stable |
| `cmd/pa/readiness.go` | `scheduled_jobs` check uses same state |
| `docs/architecture.md` or `docs/development.md` | Subsystem insertion checklist |

## Jobs state contract

```go
type jobsInitState int // initializing, ready, failed
```

Readiness: initializing → not OK; failed → not OK with detail; ready → OK.

## REQ traceability

REQ-42.001–012 covered by wire package, tests in jobs_runtime_test.go, readiness tests.
