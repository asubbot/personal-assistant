---
artefact: ep-acceptance-criteria
epic_id: EP-041
status: draft
source_of_truth: true
updated_at: 2026-05-31
---

# EP-041 — Full-tier prompt pipeline — Acceptance criteria

**Git branch:** `epic/EP-041-full-tier-pipeline`

## Acceptance criteria index

| AC ID | REQ | Test level | Summary |
|-------|-----|------------|---------|
| [AC-41.001](#ac-41-001) | REQ-41.001, REQ-41.003 | Manual | Pipeline type and single entry exist |
| [AC-41.002](#ac-41-002) | REQ-41.002, REQ-41.006 | Manual | Five steps in documented order |
| [AC-41.003](#ac-41-003) | REQ-41.004 | Unit | Parity tests for full-tier assembly |
| [AC-41.004](#ac-41-004) | REQ-41.005 | Unit | Simple tier tests unchanged |
| [AC-41.005](#ac-41-005) | REQ-41.008 | Manual (make check) | make check passes |

## Acceptance criteria

<a id="ac-41-001"></a>

### AC-41.001

**Trace:** REQ-41.001, REQ-41.003  
**Test level:** Manual

Given the epic branch source under `internal/core`  
When searching for full-tier assembly  
Then a `fullTierAssembler` (or documented equivalent) SHALL be the sole entry from `buildTierFullMainPrompt`.

<a id="ac-41-003"></a>

### AC-41.003

**Trace:** REQ-41.004  
**Test level:** Unit

Given representative handler fixtures from EP-017/018/037 tier tests  
When full-tier assembly runs on pre- and post-refactor code paths  
Then merged tool ids, `dynamicRan` flag, and system tail content SHALL match baseline captures.

<a id="ac-41-005"></a>

### AC-41.005

**Trace:** REQ-41.008  
**Test level:** Manual (make check)

When `make check` runs  
Then exit code SHALL be zero.
