# Architecture Review — EP-017 Intent Classifier for Prompt Optimization

**Reviewer:** AI Agent (delegated stage 7)

---

## Review iteration 1

**Review date:** 2026-04-15
**Stage 7 iteration:** 1 of max 5
**Document reviewed:** [ep-system-design.md](ep-system-design.md)
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 3 | Minor: 3
**Gate:** Fail (Medium and Minor > 0)

### Overall assessment

The design is well-structured, concise, and directly grounded in the requirements. All 20 requirements have explicit traceability entries with meaningful design-section references. The new `internal/intent` package introduces a clean `Classifier` interface with three focused implementations (`HeuristicClassifier`, `ModelClassifier`, `CascadeClassifier`) and reuses the existing `llm.Provider` infrastructure — no unnecessary abstractions. Three medium issues need resolution before implementation: the model-stage token logging mechanism is mapped but not specified at the component level (REQ-17.018), model-stage timeout handling is mentioned in the error table but not designed, and the environment-variable aspect of REQ-17.016 is unaddressed.

**Verdict:** Fail gate

### Strengths

- **Clean module boundary:** New `internal/intent` package is self-contained with a single `Classifier` interface, `Tier` type, and three implementations — no coupling to prompt assembly logic.
- **Dependency injection via nil:** When the classifier is disabled, `conversationHandler.classifier` is nil and `HandleMessage` skips classification entirely — zero overhead, zero code path change.
- **Full requirement traceability:** All 20 requirements (REQ-17.001–REQ-17.020) are in the traceability table with specific design section references — no gaps.
- **Fail-fast config validation:** Invalid regex → reject at load; missing endpoint/model → reject at load. Consistent with existing config validation patterns.
- **Provider reuse:** Model stage reuses `llm.OpenAICompatible` and `llm.Provider.Complete` — no new HTTP client, no new provider abstraction.
- **Cascade defaulting is safe:** Every failure path defaults to `full` tier with WARN logging — no user-visible degradation.
- **Testing strategy is well-scoped:** Maps clearly to ACs.

### Issues and recommendations

#### Blocker

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

#### Major

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

#### Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| M1 | **Model-stage token logging mechanism unspecified** | REQ-17.018 requires model-stage tokens logged separately and excluded from footer. `ModelClassifier.Classify` returns `(Tier, error)` — `CompletionResult.Usage` is not surfaced. | Add to `ModelClassifier` section: after `provider.Complete`, log prompt/completion tokens at INFO via `m.logger`; note that classification runs before `usageTurnAcc` is initialised, so its tokens never reach `footerLine()`. |
| M2 | **Model-stage timeout not designed** | REQ-17.011 lists timeout as a failure scenario. Existing `OpenAICompatible` uses 60 s default — generous for a ~100-token call. | Specify timeout strategy: add optional `timeout` field to `ClassificationModelConfig` or document that 60 s default is acceptable. |
| M3 | **Environment variable overrides unaddressed (REQ-17.016)** | REQ-17.016 says "config.yaml or environment variables". Design shows only config.json. Project uses env vars only for path resolution. | Clarify that `api_key_path` resolves via `PA_SECRETS_DIR`; note REQ-17.016 config.yaml → config.json erratum. |

#### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| m1 | **Simple-tier completion path not shown** | Pseudocode elides the `else` case. `opts` remains nil for simple tier. | Add one-line: "For simple tier, opts is nil; Provider.Complete treats nil opts as defaults (no tools)." |
| m2 | **Config format discrepancy vs. requirements** | REQ-17.016 says `config.yaml`; project uses `config.json`. | Note the erratum. |
| m3 | **ResolvePaths update not mentioned** | `ClassificationModelConfig.api_key_path` must be resolved by `ResolvePaths`. | Add line: "`ResolvePaths` extended to resolve `intent_classifier.model_stage.api_key_path`." |

### Project rules compliance

| Rule | Compliance |
|------|------------|
| KISS | ✅ |
| Fail fast | ✅ |
| Security | ✅ |
| Testability | ✅ |

### Traceability (this iteration)

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- **Scope:** [ep-scope.md](ep-scope.md)

---

## Review iteration 2

**Review date:** 2026-04-15
**Stage 7 iteration:** 2 of max 5
**Document reviewed:** [ep-system-design.md](ep-system-design.md)
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0
**Gate:** Pass

### Overall assessment

All six findings from iteration 1 have been resolved with clear, specific additions to the design document. The model-stage token logging mechanism, timeout strategy, environment-variable/path-resolution scope, simple-tier completion path, config-format erratum, and `ResolvePaths` extension are now explicitly documented at the appropriate component level. No new issues were found — the design fully traces all 20 requirements and all 18 acceptance criteria, maintains clean module boundaries, and follows project principles.

**Verdict:** Pass gate

### Iteration 1 findings — resolution status

| # | Status | Notes |
|---|--------|-------|
| M1 | Resolved | Token logging paragraph added to `ModelClassifier` section: logs prompt/completion tokens at INFO with `"component"="intent_classifier_model"`; notes structural separation from `usageTurnAcc`/`footerLine()`. Fully addresses REQ-17.018. |
| M2 | Resolved | `timeout` field added to `ClassificationModelConfig` (Go duration string, default `"5s"`). `ModelClassifier` section now specifies context-deadline application before `provider.Complete`, with error propagation to `CascadeClassifier` → WARN + `full`. Validation at load parses the duration. |
| M3 | Resolved | New "Path resolution" subsection documents `ResolvePaths` extension for `intent_classifier.model_stage.api_key_path` against `PA_SECRETS_DIR`. Config note clarifies env-var support is limited to path resolution (consistent with project). Config.yaml → config.json erratum also noted. |
| m1 | Resolved | Pseudocode comment added: simple tier leaves `opts` nil; `Provider.Complete` treats nil opts as provider defaults (no tools). Session history inclusion for both tiers is explicitly noted. |
| m2 | Resolved | Inline note after config JSON schema: "REQ-17.016 mentions `config.yaml` — the project uses `config.json`; the requirement text should be treated as referring to the project's actual config file format." |
| m3 | Resolved | "Path resolution" subsection states `ResolvePaths` extended for `intent_classifier.model_stage.api_key_path`, following the same pattern as `LLMProvider.APIKeyPath` and `EmbeddingProvider.APIKeyPath`. |

### New issues (if any)

#### Blocker

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

#### Major

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

#### Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

#### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

### Project rules compliance

| Rule | Compliance |
|------|------------|
| KISS | ✅ — Minimal new surface: one package, one interface, three focused structs. No unnecessary abstractions; reuses existing `llm.Provider`. |
| Fail fast | ✅ — Invalid regex and missing required config fields rejected at load time. |
| Security | ✅ — API key resolved via `PA_SECRETS_DIR`/`ResolvePaths`; no new HTTP clients; classification prompt sends only user message text and tier labels. |
| Testability | ✅ — `Classifier` interface enables dependency injection; nil-classifier path testable without mocks; testing strategy maps all ACs. |

### Traceability (this iteration)

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- **Scope:** [ep-scope.md](ep-scope.md)
