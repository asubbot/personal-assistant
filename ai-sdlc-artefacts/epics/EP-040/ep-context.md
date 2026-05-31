---
artefact: ep-context
epic_id: EP-040
status: draft
source_of_truth: false
updated_at: 2026-05-31
git_branch: epic/EP-040-handler-dependency-grouping
---

# Epic Context — EP-040 Handler dependency grouping

## Purpose

Group `conversationHandler` flat fields into sub-structs for readability and simpler construction; no behaviour change.

## Current Scope

Four dependency groups + constructor refactor in `run.go`; update all handler files and tests.

## Key Requirements

- REQ-40.001–004: define tool/memory/session/LLM groups
- REQ-40.005–007: migrate accessors, simplify constructor, preserve public API
- REQ-40.008–010: parity tests, scope guards, make check

## Acceptance Signals

- Flat field list gone; tests green; no config or API changes.

## Design Decisions

- Pending stage 6. Default: unexported struct types in `internal/core`, no getters.

## Open Questions

- Whether `classifier` belongs in LLM group or a small `handlerIntentDeps` (default: LLM group).

## Links

- [ep-scope.md](ep-scope.md) · Branch: `epic/EP-040-handler-dependency-grouping`
