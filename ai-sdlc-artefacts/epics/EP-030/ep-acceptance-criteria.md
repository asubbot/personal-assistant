# EP-030 — Remove Hermes text-based tool path — Acceptance criteria

This document defines acceptance criteria for [ep-scope.md](ep-scope.md), traced to [ep-requirements.md](ep-requirements.md).

---

## Acceptance criteria index

| AC | REQ (trace) | Summary |
|----|----------------|---------|
| [AC-30.001](#ac-30-001) | REQ-30.001, REQ-30.002 | Main system prompt contains no Hermes markers or tooltext format block |
| [AC-30.002](#ac-30-002) | REQ-30.002, REQ-30.016 | Handler does not parse `<tool_call>` from assistant text for tool execution |
| [AC-30.003](#ac-30-003) | REQ-30.003 | Package `internal/tooltext` is absent |
| [AC-30.004](#ac-30-004) | REQ-30.004, REQ-30.013 | Load rejects `tools.text_based_enabled` in config JSON |
| [AC-30.005](#ac-30-005) | REQ-30.005, REQ-30.013 | Load rejects `supports_json_mode` on an `llm_providers` entry |
| [AC-30.006](#ac-30-006) | REQ-30.006 | Load rejects `default_response_format` other than `text` |
| [AC-30.007](#ac-30-007) | REQ-30.007 | OpenAI-compatible completion request omits `response_format` json_object when options are default text path |
| [AC-30.008](#ac-30-008) | REQ-30.008 | `full_lite` with dynamic selection applies cap without legacy text-based gate |
| [AC-30.009](#ac-30-009) | REQ-30.009 | Startup emits WARN when baseline `supports_tools` false and tools exist |
| [AC-30.010](#ac-30-010) | REQ-30.010 | No `hermes_parse` or `invoked_via=hermes` in product logging paths for tool outcomes |
| [AC-30.011](#ac-30-011) | REQ-30.011 | Native tool descriptions use `index_text` only; no removed catalog field influences prompts |
| [AC-30.012](#ac-30-012) | REQ-30.012, REQ-30.016 | `docs/configuration.md` documents native-only tools and removed keys |
| [AC-30.013](#ac-30-013) | REQ-30.014, REQ-30.015 | `make check` and `./bin/validate EP-030` pass |

---

## Acceptance criteria

### AC-30.001

**AC-30.001** (Trace: [REQ-30.001](ep-requirements.md#hermes-removal), [REQ-30.002](ep-requirements.md#hermes-removal))

Given a valid configuration with tools enabled for the main assistant and at least one catalog tool selected for the main LLM turn  
When the core assembles the main conversation system message for tier `full` or `full_lite`  
Then the system message content SHALL NOT contain substring `<<<PA_BEGIN_HERMES_TOOL_FORMAT>>>` and SHALL NOT contain substring `<tool_call>` from static Hermes instructions.

---

### AC-30.002

**AC-30.002** (Trace: [REQ-30.002](ep-requirements.md#hermes-removal), [REQ-30.016](ep-requirements.md#native-tool-contract))

Given a handler test or integration test that records assistant output containing a fake `<tool_call>` JSON block without native `tool_calls`  
When the conversation handler processes the completion result  
Then the core SHALL NOT execute a tool solely based on that text markup.

---

### AC-30.003

**AC-30.003** (Trace: [REQ-30.003](ep-requirements.md#hermes-removal))

Given the repository tree under `internal/`  
When a reviewer searches for import path `pa/internal/tooltext`  
Then no non-vendored Go file outside `ai-sdlc-artefacts/` SHALL import that package and directory `internal/tooltext/` SHALL NOT exist.

---

### AC-30.004

**AC-30.004** (Trace: [REQ-30.004](ep-requirements.md#configuration), [REQ-30.013](ep-requirements.md#configuration))

Given a `config.json` fragment that includes `"text_based_enabled": true` inside `"tools"`  
When `config.Load` runs  
Then loading SHALL fail with an error substring naming `text_based_enabled`.

---

### AC-30.005

**AC-30.005** (Trace: [REQ-30.005](ep-requirements.md#configuration), [REQ-30.013](ep-requirements.md#configuration))

Given a `config.json` where the first `llm_providers` object includes `"supports_json_mode": true`  
When `config.Load` runs  
Then loading SHALL fail with an error substring naming `supports_json_mode`.

---

### AC-30.006

**AC-30.006** (Trace: [REQ-30.006](ep-requirements.md#llm-defaults-and-request-shape))

Given a `config.json` where an `llm_providers` entry sets `"default_response_format": "json_object"`  
When `config.Load` runs  
Then loading SHALL fail.

---

### AC-30.007

**AC-30.007** (Trace: [REQ-30.007](ep-requirements.md#llm-defaults-and-request-shape))

Given an OpenAI-compatible provider constructed from a valid config fixture with `default_response_format` `text`  
When the core issues a standard chat completion with default completion options and no explicit response format  
Then the outbound HTTP JSON body SHALL omit `response_format` with type `json_object` (either omit `response_format` or keep text-only semantics per implementation).

---

### AC-30.008

**AC-30.008** (Trace: [REQ-30.008](ep-requirements.md#dynamic-tool-capping))

Given dynamic tool selection enabled and a `full_lite` tier turn with more eligible tools than the configured cap  
When the main LLM request is built  
Then the attached tool id list length SHALL be at most the configured maximum (dynamic cap applied) without requiring any former `text_based_enabled` flag in configuration.

---

### AC-30.009

**AC-30.009** (Trace: [REQ-30.009](ep-requirements.md#operator-warning))

Given configuration where the baseline LLM provider has `supports_tools` false and the tool catalog exposes at least one tool id used for conversation  
When the application finishes wiring the tool registry and starts serving  
Then the startup logs SHALL contain exactly one `WARN` record whose message states that native tool calling is disabled and conversation tools will not run.

---

### AC-30.010

**AC-30.010** (Trace: [REQ-30.010](ep-requirements.md#observability))

Given a successful native tool invocation on the main path  
When structured logs are written for that invocation  
Then log attributes SHALL NOT use value `hermes` for tool invocation source and SHALL NOT emit failure class `hermes_parse`.

---

### AC-30.011

**AC-30.011** (Trace: [REQ-30.011](ep-requirements.md#tool-catalog))

Given catalog tools with defined `index_text`  
When native tool definitions are built for the LLM request  
Then each tool’s description SHALL equal that tool’s `index_text` and SHALL NOT be taken from any removed catalog field.

---

### AC-30.012

**AC-30.012** (Trace: [REQ-30.012](ep-requirements.md#documentation), [REQ-30.016](ep-requirements.md#native-tool-contract))

Given the file `docs/configuration.md` in the repository  
When an operator reads the LLM provider and tools sections  
Then the documentation SHALL state that conversation tools require native tool support on the baseline provider and SHALL list removed configuration keys for this epic.

---

### AC-30.013

**AC-30.013** (Trace: [REQ-30.014](ep-requirements.md#verification), [REQ-30.015](ep-requirements.md#verification))

Given the repository HEAD with EP-030 changes applied  
When the maintainer runs `make check` then `./bin/validate EP-030`  
Then both commands SHALL exit with status zero.

---

## Notes

- AC-30.002 is satisfied by automated tests that assert no tool execution from Hermes-only bodies; trace comments SHALL bind tests to AC ids.
- AC-30.007 may be covered by existing OpenAI client unit tests after `ForceJSONOutput` removal; update trace comments to AC-30.007 / REQ-30.007.
