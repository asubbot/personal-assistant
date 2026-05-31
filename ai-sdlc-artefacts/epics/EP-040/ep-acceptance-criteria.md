---
artefact: ep-acceptance-criteria
epic_id: EP-040
status: draft
source_of_truth: true
updated_at: 2026-05-31
---

# EP-040 — Handler dependency grouping — Acceptance criteria

**Git branch:** `epic/EP-040-handler-dependency-grouping`

## Acceptance criteria index

| AC ID | REQ | Test level | Summary |
|-------|-----|------------|---------|
| [AC-40.001](#ac-40-001) | REQ-40.001–004 | Unit | conversationHandler uses grouped deps structs |
| [AC-40.002](#ac-40-002) | REQ-40.005 | Manual | No flat tool/memory field access outside group definitions |
| [AC-40.003](#ac-40-003) | REQ-40.006 | Manual | Constructor builds four groups |
| [AC-40.004](#ac-40-004) | REQ-40.007 | Unit | Public handler API unchanged |
| [AC-40.005](#ac-40-005) | REQ-40.008 | Unit | Handler test suite green, same assertions |
| [AC-40.006](#ac-40-006) | REQ-40.009 | Manual | No config schema diff |
| [AC-40.007](#ac-40-007) | REQ-40.010 | Manual (make check) | make check passes |

## Acceptance criteria

<a id="ac-40-001"></a>

### AC-40.001

**Trace:** REQ-40.001, REQ-40.002, REQ-40.003, REQ-40.004  
**Test level:** Unit

Given the epic branch `conversationHandler` type definition  
When inspected in `internal/core/handler.go`  
Then dependency fields SHALL be grouped into `handlerToolDeps`, `handlerMemoryDeps`, `handlerSessionDeps`, and `handlerLLMDeps` (or documented equivalents).

**Status:** AC-40.002 MANUAL ONLY — verified by reviewing handler*.go for absence of flat dependency field access outside group struct definitions.

<a id="ac-40-003"></a>

### AC-40.003

**Trace:** REQ-40.006  
**Test level:** Manual

Given `newRunConversationHandler` in `internal/core/run.go`  
When inspected  
Then four dependency group literals SHALL be constructed.

**Status:** AC-40.003 MANUAL ONLY — verified by inspecting `run.go` group construction.

<a id="ac-40-004"></a>

### AC-40.004

**Trace:** REQ-40.007  
**Test level:** Unit

Given `BuildMessageHandler` on the epic branch  
When called with minimal config  
Then it SHALL return a `MessageHandler` without signature change.

<a id="ac-40-005"></a>

### AC-40.005

**Trace:** REQ-40.008  
**Test level:** Unit

Given the full `internal/core/handler*` test package  
When `go test ./internal/core/...` runs  
Then all tests SHALL pass with unchanged expected strings, tool ids, and message shapes versus pre-epic baseline.

**Status:** AC-40.006 MANUAL ONLY — verified by reviewing branch diff for absence of config schema changes.

<a id="ac-40-007"></a>

### AC-40.007

**Trace:** REQ-40.010  
**Test level:** Manual (make check)

When `make check` runs on the epic branch  
Then exit code SHALL be zero.

**Status:** AC-40.007 MANUAL ONLY — verified by running `make check` (exit 0).
