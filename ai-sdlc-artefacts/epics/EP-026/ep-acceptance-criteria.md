# EP-026 — Acceptance criteria

## Introduction

Testable criteria for tier builders in the conversation handler, traceable to [ep-requirements.md](ep-requirements.md).

---

## Acceptance criteria index

| AC ID | REQ (trace) | Summary |
|-------|-------------|---------|
| [AC-26.001](#ac-26-001) | [REQ-26.001](ep-requirements.md#tier-builders) | Distinct tier builder entry points exist |
| [AC-26.002](#ac-26-002) | [REQ-26.002](ep-requirements.md#orchestrator) | Tier dispatch helper used from `HandleMessage` |
| [AC-26.003](#ac-26-003) | [REQ-26.003](ep-requirements.md#tests) | Unit tests exercise tier builders with minimal handler |
| [AC-26.004](#ac-26-004) | [REQ-26.004](ep-requirements.md#lint) | No `gocyclo` nolint on `HandleMessage` |
| [AC-26.005](#ac-26-005) | [REQ-26.005](ep-requirements.md#parity) | Existing `internal/core` tests pass |
| [AC-26.006](#ac-26-006) | [REQ-26.006](ep-requirements.md#verification) | `make check` and validate EP-026 pass |

---

## Acceptance criteria

### AC-26.001

**Trace:** [REQ-26.001](ep-requirements.md#tier-builders)

Given the `internal/core` source tree  
When a reviewer searches for `buildTierFullMainPrompt`, `buildTierFullLiteMainPrompt`, and `buildTierSimpleMainPrompt`  
Then all three identifiers are defined as methods on `*conversationHandler`.

### AC-26.002

**Trace:** [REQ-26.002](ep-requirements.md#orchestrator)

Given `handler.go`  
When a reviewer inspects `HandleMessage`  
Then the function delegates session key handling, classification, optional retrieval, and the base message stack to `buildMainTurnMessagesPreTail`, and delegates tier-specific prompt assembly to `assembleTierMainLLMParams` (and no longer contains the former duplicated full vs full_lite tail `switch` bodies).

### AC-26.003

**Trace:** [REQ-26.003](ep-requirements.md#tests)

Given package `core` tests  
When `go test ./internal/core -count=1 -run TierMainPromptBuilders` runs  
Then tests pass and exercise tier builder contracts with minimal fixtures.

### AC-26.004

**Trace:** [REQ-26.004](ep-requirements.md#lint)

Given `handler.go`  
When a reviewer reads the `HandleMessage` godoc and implementation  
Then there is no `//nolint:gocyclo` comment applied to `HandleMessage`.

### AC-26.005

**Trace:** [REQ-26.005](ep-requirements.md#parity)

Given the repository checkout  
When `go test -tags=integration -count=1 ./internal/core/...` runs  
Then the command exits with status zero.

### AC-26.006

**Trace:** [REQ-26.006](ep-requirements.md#verification)

Given the epic branch  
When `make check` and `make build && ./bin/validate EP-026` run from the repository root  
Then both commands exit with status zero.
