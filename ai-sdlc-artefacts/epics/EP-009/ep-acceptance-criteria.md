# EP-009 Dynamic Tool Creation with Docker Sandbox — Acceptance criteria

This document defines testable acceptance criteria for [EP-009](ep-scope.md) in Gherkin form. Each criterion traces to [ep-requirements.md](ep-requirements.md). Use this document as input for system design (stage 6) and implementation planning (stage 7).

**Contents**

- [Introduction](#introduction)
- [Acceptance criteria index](#acceptance-criteria-index)
- [Acceptance criteria](#acceptance-criteria)

---

## Introduction

EP-009 adds dynamic tool creation with Docker-based sandbox execution on configured nodes and persistence to the tool catalog. This document states **when the epic is done** from a testing perspective: each AC is verifiable and maps to one or more requirements.

---

## Acceptance criteria index

| AC ID | REQ (trace) | Summary |
|-------|-------------|---------|
| [AC-09.001](#ac-09-001) | [REQ-09.001](ep-requirements.md#docker-sandbox-execution) | Sandbox execution uses Docker network bridge |
| [AC-09.002](#ac-09-002) | [REQ-09.002](ep-requirements.md#docker-sandbox-execution) | Sandbox applies 256MB memory limit |
| [AC-09.003](#ac-09-003) | [REQ-09.003](ep-requirements.md#docker-sandbox-execution) | Sandbox applies 0.5 CPU limit |
| [AC-09.004](#ac-09-004) | [REQ-09.004](ep-requirements.md#docker-sandbox-execution) | Sandbox enforces 30s timeout |
| [AC-09.005](#ac-09-005) | [REQ-09.005](ep-requirements.md#docker-sandbox-execution) | Python 3.14 pa-sandbox image available |
| [AC-09.006](#ac-09-006) | [REQ-09.006](ep-requirements.md#docker-sandbox-execution) | Node.js 22 pa-sandbox image available |
| [AC-09.007](#ac-09-007) | [REQ-09.007](ep-requirements.md#docker-sandbox-execution) | Alpine base pa-sandbox image available |
| [AC-09.008](#ac-09-008) | [REQ-09.008](ep-requirements.md#tool-creation) | create_tool accepts required and optional parameters |
| [AC-09.009](#ac-09-009) | [REQ-09.009](ep-requirements.md#tool-creation) | Whitelist validation; invalid template rejected |
| [AC-09.010](#ac-09-010) | [REQ-09.010](ep-requirements.md#tool-creation) | Duplicate tool ID rejected |
| [AC-09.011](#ac-09-011) | [REQ-09.011](ep-requirements.md#tool-creation) | Valid tool appended to tools.yaml |
| [AC-09.012](#ac-09-012) | [REQ-09.012](ep-requirements.md#tool-creation) | New tool in runtime catalog without restart |
| [AC-09.013](#ac-09-013) | [REQ-09.013](ep-requirements.md#tool-creation) | Success response includes tool id |
| [AC-09.014](#ac-09-014) | [REQ-09.014](ep-requirements.md#non-functional-requirements) | Cached image: start within 5s |
| [AC-09.015](#ac-09-015) | [REQ-09.015](ep-requirements.md#non-functional-requirements) | create_tool completes within 1s |
| [AC-09.016](#ac-09-016) | [REQ-09.016](ep-requirements.md#non-functional-requirements) | Unit coverage ≥70% for create_tool and validation |
| [AC-09.017](#ac-09-017) | [REQ-09.017](ep-requirements.md#non-functional-requirements) | Secret-like content rejected on persist |

---

## Acceptance criteria

### Docker sandbox execution

**AC-09.001** (Trace: [REQ-09.001](ep-requirements.md#docker-sandbox-execution))

Given a catalog tool whose substituted command runs sandboxed code on a configured node  
When PersonalAssistant executes that tool through the node runner  
Then the remote command SHALL include Docker `--network bridge` for outbound network access  

---

**AC-09.002** (Trace: [REQ-09.002](ep-requirements.md#docker-sandbox-execution))

Given a sandbox execution on a node  
When PersonalAssistant builds the `docker run` invocation  
Then the command SHALL include `--memory="256m"`  

---

**AC-09.003** (Trace: [REQ-09.003](ep-requirements.md#docker-sandbox-execution))

Given a sandbox execution on a node  
When PersonalAssistant builds the `docker run` invocation  
Then the command SHALL include `--cpus="0.5"`  

---

**AC-09.004** (Trace: [REQ-09.004](ep-requirements.md#docker-sandbox-execution))

Given a sandbox execution on a node  
When the container runs longer than 30 seconds  
Then PersonalAssistant SHALL terminate the execution and surface a timeout outcome  

---

**AC-09.005** (Trace: [REQ-09.005](ep-requirements.md#docker-sandbox-execution))

Given the operator has built and tagged `pa-sandbox:python` on the node per epic documentation  
When a tool template references `pa-sandbox:python`  
Then execution SHALL use an image that provides Python 3.14 and the packages listed in REQ-09.005  

---

**AC-09.006** (Trace: [REQ-09.006](ep-requirements.md#docker-sandbox-execution))

Given the operator has built and tagged `pa-sandbox:node` on the node per epic documentation  
When a tool template references `pa-sandbox:node`  
Then execution SHALL use an image that provides Node.js 22 LTS and the packages listed in REQ-09.006  

---

**AC-09.007** (Trace: [REQ-09.007](ep-requirements.md#docker-sandbox-execution))

Given the operator has built and tagged `pa-sandbox:base` on the node per epic documentation  
When a tool template references `pa-sandbox:base`  
Then execution SHALL use an Alpine-based image with `curl` and `jq`  

---

### Tool creation

**AC-09.008** (Trace: [REQ-09.008](ep-requirements.md#tool-creation))

Given the LLM issues a `create_tool` call  
When the call includes id, index_text, template, and node_id  
Then PersonalAssistant SHALL accept the call and optional arguments and system_prompt when present  

---

**AC-09.009** (Trace: [REQ-09.009](ep-requirements.md#tool-creation))

Given a `create_tool` request with a template  
When the template does not start with `docker run --rm --network bridge` or `docker run --rm --network none`  
Then PersonalAssistant SHALL reject the request with a validation error  

---

**AC-09.010** (Trace: [REQ-09.010](ep-requirements.md#tool-creation))

Given an existing tool id in the catalog  
When `create_tool` uses the same id  
Then PersonalAssistant SHALL reject the request with a duplicate ID error  

---

**AC-09.011** (Trace: [REQ-09.011](ep-requirements.md#tool-creation))

Given a valid `create_tool` request  
When validation succeeds  
Then the new tool entry SHALL appear appended in tools.yaml in valid YAML list form  

---

**AC-09.012** (Trace: [REQ-09.012](ep-requirements.md#tool-creation))

Given a successful `create_tool` write to tools.yaml  
When the write completes  
Then the new tool SHALL be present in the in-memory catalog without restarting the process  

---

**AC-09.013** (Trace: [REQ-09.013](ep-requirements.md#tool-creation))

Given a successful `create_tool` operation  
When the operation completes  
Then the result returned to the LLM SHALL state the tool id and that the tool is available  

---

### Non-functional requirements

**AC-09.014** (Trace: [REQ-09.014](ep-requirements.md#non-functional-requirements))

Given the sandbox image is present in the node Docker cache  
When a sandbox tool run starts  
Then the time from run request to container start SHALL be at most 5 seconds in the integration test environment  

---

**AC-09.015** (Trace: [REQ-09.015](ep-requirements.md#non-functional-requirements))

Given a valid `create_tool` request  
When measured end-to-end in automated tests  
Then validation, file write, and runtime catalog update SHALL complete within 1 second  

---

**AC-09.016** (Trace: [REQ-09.016](ep-requirements.md#non-functional-requirements))

Given the `create_tool` and template validation packages  
When `make check` or the coverage report runs  
Then line coverage for those packages SHALL be at least 70 percent  

---

**AC-09.017** (Trace: [REQ-09.017](ep-requirements.md#non-functional-requirements))

Given operator-configured secret detection patterns  
When `create_tool` receives a template or field matching a pattern  
Then PersonalAssistant SHALL reject persistence with an error  

---

## Traceability

- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Scope:** [ep-scope.md](ep-scope.md)
