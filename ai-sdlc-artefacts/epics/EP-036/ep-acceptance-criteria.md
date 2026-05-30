---
artefact: ep-acceptance-criteria
epic_id: EP-036
status: draft
source_of_truth: true
updated_at: 2026-05-30
---

# EP-036 — Simplify intent classification — Acceptance criteria

## Introduction

Testable acceptance criteria for **EP-036**: remove the intent-classifier **model stage** and the **`full_lite`** tier so classification is heuristic-only with two outcomes (`simple`, `full`), ambiguous heuristics default to **`full`** without a classification LLM call, and removed config keys fail at load. Criteria trace to [ep-requirements.md](ep-requirements.md) and [ep-scope.md](ep-scope.md). Test levels follow [strategy.md](../../strategy.md) §2 (Unit / Integration / E2E / Manual).

---

## Acceptance criteria index

| AC ID | REQ (trace) | Test level | Summary |
|-------|-------------|------------|---------|
| [AC-36.001](#ac-36-001) | [REQ-36.001](ep-requirements.md#req-36-001--two-complexity-tiers) | Unit | Exactly two tiers: `simple`, `full` |
| [AC-36.002](#ac-36-002) | [REQ-36.002](ep-requirements.md#req-36-002--remove-full_lite-tier) | Unit | `full_lite` / `TierFullLite` absent from `intent` |
| [AC-36.003](#ac-36-003) | [REQ-36.003](ep-requirements.md#req-36-003--one-tier-per-turn-when-enabled) | Integration | Enabled classifier assigns one tier per turn |
| [AC-36.004](#ac-36-004) | [REQ-36.004](ep-requirements.md#req-36-004--heuristic-evaluation-order) | Unit | Heuristic order: length → simple → full → ambiguous |
| [AC-36.005](#ac-36-005) | [REQ-36.005](ep-requirements.md#req-36-005--no-full_lite-patterns-in-heuristic) | Unit | Heuristic does not evaluate `full_lite_patterns` |
| [AC-36.006](#ac-36-006) | [REQ-36.006](ep-requirements.md#req-36-006--ambiguous-defaults-to-full), [REQ-36.023](ep-requirements.md#req-36-023--classification-and-config-load-tests) | Unit | Ambiguous → `full` / `default`; no classification LLM call |
| [AC-36.007](#ac-36-007) | [REQ-36.007](ep-requirements.md#req-36-007--confident-heuristic-stage-label), [REQ-36.011](ep-requirements.md#req-36-011--stage-values-heuristic-or-default) | Unit | Confident → `heuristic`; stages only `heuristic` \| `default` |
| [AC-36.008](#ac-36-008) | [REQ-36.008](ep-requirements.md#req-36-008--delete-model-stage-code), [REQ-36.009](ep-requirements.md#req-36-009--remove-modelclassifier-type) | Manual | `model.go` / `model_test.go` removed; no `ModelClassifier` |
| [AC-36.009](#ac-36-009) | [REQ-36.010](ep-requirements.md#req-36-010--no-classification-llm-wiring) | Manual (build/grep) | `cmd/pa` does not wire classification LLM |
| [AC-36.010](#ac-36-010) | [REQ-36.012](ep-requirements.md#req-36-012--dispatch-simple-and-full-only), [REQ-36.013](ep-requirements.md#req-36-013--remove-full_lite-prompt-builder) | Unit | Core tier dispatch and builders: `simple` and `full` only |
| [AC-36.011](#ac-36-011) | [REQ-36.014](ep-requirements.md#req-36-014--parity-for-simple-and-full-assembly) | Integration | `simple` / `full` main-prompt assembly unchanged vs pre-epic |
| [AC-36.012](#ac-36-012) | [REQ-36.015](ep-requirements.md#req-36-015--former-full_lite-uses-full-path) | Integration | Pre-epic `full_lite` fixtures run `full` assembly |
| [AC-36.013](#ac-36-013) | [REQ-36.016](ep-requirements.md#req-36-016--reject-model_stage-config-key), [REQ-36.024](ep-requirements.md#req-36-024--reject-removed-keys-in-tests) | Unit | Config load rejects `intent_classifier.model_stage` |
| [AC-36.014](#ac-36-014) | [REQ-36.017](ep-requirements.md#req-36-017--reject-full_lite_patterns-config-key), [REQ-36.024](ep-requirements.md#req-36-024--reject-removed-keys-in-tests) | Unit | Config load rejects `heuristic.full_lite_patterns` |
| [AC-36.015](#ac-36-015) | [REQ-36.018](ep-requirements.md#req-36-018--enabled-heuristic-schema), [REQ-36.019](ep-requirements.md#req-36-019--validate-heuristic-at-load) | Unit | Enabled `heuristic` schema and regex / length validation at load |
| [AC-36.016](#ac-36-016) | [REQ-36.020](ep-requirements.md#req-36-020--keep-intent_classifier-root-key), [REQ-36.021](ep-requirements.md#req-36-021--null-intent_classifier-disables-classification) | Unit | Root `intent_classifier` key; JSON `null` accepted |
| [AC-36.017](#ac-36-017) | [REQ-36.022](ep-requirements.md#req-36-022--update-configs-and-operator-docs) | Manual | Operator docs describe two-tier heuristic-only cascade |
| [AC-36.018](#ac-36-018) | [REQ-36.022](ep-requirements.md#req-36-022--update-configs-and-operator-docs) | Manual | Live operator `.config/config.json` loads under new schema |
| [AC-36.019](#ac-36-019) | [REQ-36.025](ep-requirements.md#req-36-025--retire-obsolete-tier-tests) | Manual (test inventory) | Obsolete model / three-tier / `full_lite` tests removed or rewritten |
| [AC-36.020](#ac-36-020) | [REQ-36.026](ep-requirements.md#req-36-026--make-check-passes) | Manual (make check) | `make check` exits zero |
| [AC-36.021](#ac-36-021) | [REQ-36.027](ep-requirements.md#req-36-027--epic-validation-passes) | Manual (validate) | `./bin/validate ears EP-036` passes |
| [AC-36.022](#ac-36-022) | [REQ-36.022](ep-requirements.md#req-36-022--update-configs-and-operator-docs) | Unit | Example and testdata configs load without removed keys |

---

## Scenarios

### AC-36.006 Ambiguous default without classification LLM (Trace: REQ-36.006, REQ-36.023)

Given `intent_classifier` is enabled and a user message is ambiguous under heuristic rules  
When `CascadeClassifier.Classify` runs with a spy or mock classification LLM provider wired in `cmd/pa` tests  
Then the result SHALL be tier `full` and stage `default`  
And the classification LLM `Complete` SHALL NOT be invoked for that turn.

### AC-36.007 Confident heuristic stage (Trace: REQ-36.007, REQ-36.011)

Given `intent_classifier` is enabled and a user message matches `simple_patterns` or `full_patterns` confidently  
When `CascadeClassifier.Classify` runs  
Then `Result.Stage` SHALL be `heuristic`  
And `Result.Stage` SHALL NOT be any value other than `heuristic` or `default` in production outcomes.

### AC-36.013 Reject model_stage (Trace: REQ-36.016, REQ-36.024)

Given a `config.json` fixture containing `intent_classifier.model_stage`  
When config load runs in `internal/config` tests  
Then load SHALL fail with an explicit validation error.

### AC-36.014 Reject full_lite_patterns (Trace: REQ-36.017, REQ-36.024)

Given a `config.json` fixture containing `intent_classifier.heuristic.full_lite_patterns`  
When config load runs in `internal/config` tests  
Then load SHALL fail with an explicit validation error.

### AC-36.022 Representative configs load (Trace: REQ-36.022)

Given `config.examples/config.example.json` and updated `internal/config/testdata/` fixtures on the epic branch  
When config load runs in automated tests  
Then each file SHALL load successfully without removed nested keys.

---

## Acceptance criteria

<a id="ac-36-001"></a>

### AC-36.001

**Trace:** [REQ-36.001](ep-requirements.md#req-36-001--two-complexity-tiers)  
**Test level:** Unit

Given the `intent` package after EP-036  
When inspecting exported tier values used for main-LLM prompt assembly  
Then exactly two tiers SHALL exist: `simple` and `full`.

---

<a id="ac-36-002"></a>

### AC-36.002

**Trace:** [REQ-36.002](ep-requirements.md#req-36-002--remove-full_lite-tier)  
**Test level:** Unit

Given the `intent` package and production code under `cmd/` and `internal/` after EP-036  
When searching for the identifier `TierFullLite` or the tier string `full_lite` in tier dispatch  
Then no production symbol or tier branch SHALL remain for `full_lite`.

---

<a id="ac-36-003"></a>

### AC-36.003

**Trace:** [REQ-36.003](ep-requirements.md#req-36-003--one-tier-per-turn-when-enabled)  
**Test level:** Integration

Given `intent_classifier` is enabled in configuration  
When the handler classifies a user message before main-model prompt assembly  
Then exactly one tier in {`simple`, `full`} SHALL be assigned for that turn.

---

<a id="ac-36-004"></a>

### AC-36.004

**Trace:** [REQ-36.004](ep-requirements.md#req-36-004--heuristic-evaluation-order)  
**Test level:** Unit

Given heuristic configuration with distinct `simple_patterns`, `full_patterns`, and `max_simple_len`  
When classifying messages that exercise each decision point  
Then evaluation order SHALL be: length guard (`max_simple_len`), then `simple_patterns`, then `full_patterns`, then ambiguous.

---

<a id="ac-36-005"></a>

### AC-36.005

**Trace:** [REQ-36.005](ep-requirements.md#req-36-005--no-full_lite-patterns-in-heuristic)  
**Test level:** Unit

Given the heuristic classifier implementation after EP-036  
When inspecting `internal/intent/heuristic.go` and its tests  
Then the heuristic stage SHALL NOT read or evaluate a `full_lite_patterns` field.

---

<a id="ac-36-006"></a>

### AC-36.006

**Trace:** [REQ-36.006](ep-requirements.md#req-36-006--ambiguous-defaults-to-full), [REQ-36.023](ep-requirements.md#req-36-023--classification-and-config-load-tests)  
**Test level:** Unit

Given `intent_classifier` is enabled and a message is ambiguous after heuristics  
When `CascadeClassifier.Classify` runs (unit or handler test with a classification LLM spy)  
Then the result SHALL be tier `full` and `Result.Stage` `default`  
And no classification LLM `Complete` call SHALL occur for that turn.

---

<a id="ac-36-007"></a>

### AC-36.007

**Trace:** [REQ-36.007](ep-requirements.md#req-36-007--confident-heuristic-stage-label), [REQ-36.011](ep-requirements.md#req-36-011--stage-values-heuristic-or-default)  
**Test level:** Unit

Given `intent_classifier` is enabled and heuristics return a confident tier  
When `CascadeClassifier.Classify` runs  
Then `Result.Stage` SHALL be `heuristic` and `Result.Tier` SHALL match the confident heuristic tier  
And production classification outcomes SHALL use only `Result.Stage` values `heuristic` or `default`.

---

<a id="ac-36-008"></a>

### AC-36.008

**Trace:** [REQ-36.008](ep-requirements.md#req-36-008--delete-model-stage-code), [REQ-36.009](ep-requirements.md#req-36-009--remove-modelclassifier-type)  
**Test level:** Manual  
**Status:** AC-36.008 MANUAL ONLY — verified by repository tree inspection (`internal/intent/model.go` and `model_test.go` absent) and grep for `ModelClassifier` (zero matches in product code); no unit test applies.

Given the EP-036 change set on the epic branch  
When inspecting `internal/intent/`  
Then `model.go` and `model_test.go` SHALL be absent  
And the type name `ModelClassifier` SHALL not appear in product packages.

---

<a id="ac-36-009"></a>

### AC-36.009

**Trace:** [REQ-36.010](ep-requirements.md#req-36-010--no-classification-llm-wiring)  
**Test level:** Manual (build/grep)  
**Status:** AC-36.009 MANUAL ONLY — verified by grep of `cmd/pa` intent wiring for classification LLM / `ModelClassifier` construction (zero matches) and successful `make check` build.  
**Related coverage:** the behaviour of no extra LLM call is exercised by the ambiguous-default cascade unit tests.

Given the EP-036 change set  
When inspecting `cmd/pa` intent-classifier construction (`buildIntentClassifier` and related wiring)  
Then the product SHALL NOT construct a classification LLM provider or `ModelClassifier` for intent classification.

---

<a id="ac-36-010"></a>

### AC-36.010

**Trace:** [REQ-36.012](ep-requirements.md#req-36-012--dispatch-simple-and-full-only), [REQ-36.013](ep-requirements.md#req-36-013--remove-full_lite-prompt-builder)  
**Test level:** Unit

Given tier main-prompt assembly in `internal/core` after EP-036  
When inspecting `assembleTierMainLLMParams` and tier builder functions  
Then dispatch SHALL handle only `intent.TierSimple` and `intent.TierFull`  
And `buildTierFullLiteMainPrompt` and `TierFullLite` branches SHALL be absent.

---

<a id="ac-36-011"></a>

### AC-36.011

**Trace:** [REQ-36.014](ep-requirements.md#req-36-014--parity-for-simple-and-full-assembly)  
**Test level:** Integration

Given existing or updated tier main-prompt tests for `simple` and `full` from before EP-036  
When those tests run on the epic branch  
Then expected prompt structure, included components, and token/rune assertions for each tier SHALL match the pre-EP-036 baseline for that tier.

---

<a id="ac-36-012"></a>

### AC-36.012

**Trace:** [REQ-36.015](ep-requirements.md#req-36-015--former-full_lite-uses-full-path)  
**Test level:** Integration

Given user messages that were classified `full_lite` under pre-EP-036 heuristic or model rules (fixture table in tests)  
When classified and assembled on the epic branch  
Then main-LLM parameter assembly SHALL use the existing `full` tier path for that turn.

---

<a id="ac-36-013"></a>

### AC-36.013

**Trace:** [REQ-36.016](ep-requirements.md#req-36-016--reject-model_stage-config-key), [REQ-36.024](ep-requirements.md#req-36-024--reject-removed-keys-in-tests)  
**Test level:** Unit

Given a `config.json` containing `intent_classifier.model_stage` at any depth under `intent_classifier`  
When config load runs  
Then load SHALL fail with an explicit validation error naming the removed key.

---

<a id="ac-36-014"></a>

### AC-36.014

**Trace:** [REQ-36.017](ep-requirements.md#req-36-017--reject-full_lite_patterns-config-key), [REQ-36.024](ep-requirements.md#req-36-024--reject-removed-keys-in-tests)  
**Test level:** Unit

Given a `config.json` containing `intent_classifier.heuristic.full_lite_patterns`  
When config load runs  
Then load SHALL fail with an explicit validation error naming the removed key.

---

<a id="ac-36-015"></a>

### AC-36.015

**Trace:** [REQ-36.018](ep-requirements.md#req-36-018--enabled-heuristic-schema), [REQ-36.019](ep-requirements.md#req-36-019--validate-heuristic-at-load)  
**Test level:** Unit

Given `intent_classifier` is enabled with a valid `heuristic` object  
When config load runs  
Then the loaded config SHALL require `simple_patterns`, `full_patterns`, and `max_simple_len`  
And invalid regex patterns or `max_simple_len` &lt; 1 SHALL fail load at validation time.

---

<a id="ac-36-016"></a>

### AC-36.016

**Trace:** [REQ-36.020](ep-requirements.md#req-36-020--keep-intent_classifier-root-key), [REQ-36.021](ep-requirements.md#req-36-021--null-intent_classifier-disables-classification)  
**Test level:** Unit

Given root-key validation for `config.json`  
When `intent_classifier` is JSON `null` or an enabled heuristic-only object  
Then load SHALL accept the configuration  
And the documented top-level key list SHALL still include `intent_classifier`.

---

<a id="ac-36-017"></a>

### AC-36.017

**Trace:** [REQ-36.022](ep-requirements.md#req-36-022--update-configs-and-operator-docs)  
**Test level:** Manual  
**Status:** AC-36.017 MANUAL ONLY — verified by reading `docs/configuration.md` and `docs/llm-provider-roles-and-logging.md` for two-tier heuristic-only prose and absence of model-stage / `full_lite` setup instructions.

Given the EP-036 documentation update  
When an operator reads `docs/configuration.md` and `docs/llm-provider-roles-and-logging.md`  
Then the docs SHALL describe heuristic-only classification with tiers `simple` and `full` only  
And SHALL NOT document `model_stage`, `full_lite_patterns`, or a three-tier / model-stage cascade.

---

<a id="ac-36-018"></a>

### AC-36.018

**Trace:** [REQ-36.022](ep-requirements.md#req-36-022--update-configs-and-operator-docs)  
**Test level:** Manual  
**Status:** AC-36.018 MANUAL ONLY — operator `.config/config.json` is verified manually and by app startup validation.  
**Related coverage:** automated schema, positive-load, and removed-key rejection coverage for the new classifier schema is provided by the example config, a testdata fixture, and the dedicated rejection tests, which remain Unit.

Given the operator `.config/config.json` updated for the shrunk classifier schema on the epic branch  
When the application loads and validates it at startup (or an operator runs the binary against it)  
Then the live config SHALL load successfully without `model_stage` or `full_lite_patterns`.

---

<a id="ac-36-019"></a>

### AC-36.019

**Trace:** [REQ-36.025](ep-requirements.md#req-36-025--retire-obsolete-tier-tests)  
**Test level:** Manual (test inventory)  
**Status:** AC-36.019 MANUAL ONLY — verified by searching the test tree for tests whose sole purpose is model-stage parsing, three-way tier classification, or `full_lite` prompt token deltas (none remain, or they are rewritten); `make check` provides secondary confirmation.

Given the EP-036 test refresh  
When reviewing tests under `internal/intent/`, `internal/core/`, `cmd/pa/`, and related packages  
Then tests dedicated only to the removed model stage, three-way tier routing, or `full_lite` token deltas SHALL be removed or rewritten to the two-tier heuristic cascade.

---

<a id="ac-36-020"></a>

### AC-36.020

**Trace:** [REQ-36.026](ep-requirements.md#req-36-026--make-check-passes)  
**Test level:** Manual (make check)  
**Status:** AC-36.020 MANUAL ONLY — verified by running `make check` from the repository root (exit 0); this is a process gate, not a unit test.

Given EP-036 implementation is complete on the epic branch  
When `make check` runs from the repository root  
Then it SHALL exit with status zero.

---

<a id="ac-36-021"></a>

### AC-36.021

**Trace:** [REQ-36.027](ep-requirements.md#req-36-027--epic-validation-passes)  
**Test level:** Manual (validate)  
**Status:** AC-36.021 MANUAL ONLY — verified by running `./bin/validate ears EP-036` from the repository root after `make build`; this is an artefact gate, not a product unit test.

Given `ep-requirements.md` for EP-036 on the epic branch  
When `./bin/validate ears EP-036` runs from the repository root  
Then validation SHALL report no EARS format errors for the requirements artefact.

---

<a id="ac-36-022"></a>

### AC-36.022

**Trace:** [REQ-36.022](ep-requirements.md#req-36-022--update-configs-and-operator-docs)  
**Test level:** Unit

Given `config.examples/config.example.json` and updated files under `internal/config/testdata/` on the epic branch  
When config load runs in automated tests  
Then each file SHALL load successfully without `model_stage` or `full_lite_patterns`.
