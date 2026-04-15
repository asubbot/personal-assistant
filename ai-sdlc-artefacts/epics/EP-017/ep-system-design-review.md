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
