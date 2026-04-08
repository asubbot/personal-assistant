# EP-011 Native web search and HTTPS content fetch — Acceptance criteria

This document defines testable acceptance criteria for [EP-011](ep-scope.md) in Gherkin form. Each criterion traces to [ep-requirements.md](ep-requirements.md). Use this document as input for system design (stage 6) and implementation planning (stage 7).

**Contents**

- [Introduction](#introduction)
- [Acceptance criteria index](#acceptance-criteria-index)
- [Acceptance criteria](#acceptance-criteria)

---

## Introduction

EP-011 adds native **web_search** (Brave Search or DuckDuckGo, with optional **fallback search provider**), **web_fetch** (HTTPS-only, SSRF-safe), and an in-process search cache with TTL. This document states **when the epic is done** from a testing perspective: **16** acceptance criteria are verifiable and map to [ep-requirements.md](ep-requirements.md).

---

## Acceptance criteria index

| AC ID | REQ (trace) | Summary |
|-------|-------------|---------|
| [AC-11.001](#ac-11-001) | [REQ-11.001](ep-requirements.md#tools--providers), [REQ-11.002](ep-requirements.md#tools--providers), [REQ-11.019](ep-requirements.md#config--limits), [REQ-11.020](ep-requirements.md#config--limits) | When `web_tools.enabled`, **web_search** and **web_fetch** register as native tools; invalid `web_tools` fails startup |
| [AC-11.002](#ac-11-002) | [REQ-11.003](ep-requirements.md#tools--providers) | **web_search** rejects empty or whitespace-only query |
| [AC-11.003](#ac-11-003) | [REQ-11.004](ep-requirements.md#tools--providers), [REQ-11.006](ep-requirements.md#tools--providers), [REQ-11.008](ep-requirements.md#tools--providers) | Brave provider returns ranked title, URL, snippet from API JSON |
| [AC-11.004](#ac-11-004) | [REQ-11.005](ep-requirements.md#tools--providers), [REQ-11.008](ep-requirements.md#tools--providers) | DuckDuckGo provider returns ranked title, URL, snippet from HTML response |
| [AC-11.005](#ac-11-005) | [REQ-11.007](ep-requirements.md#tools--providers) | Brave provider with missing API key file returns structured error |
| [AC-11.006](#ac-11-006) | [REQ-11.009](ep-requirements.md#search-cache)–[REQ-11.012](ep-requirements.md#search-cache) | Cache hit returns same results without second upstream call; TTL expiry refetches; max entries evicts |
| [AC-11.007](#ac-11-007) | [REQ-11.010](ep-requirements.md#search-cache) | Different normalized queries or **search provider** chains produce distinct cache keys |
| [AC-11.008](#ac-11-008) | [REQ-11.013](ep-requirements.md#web_fetch--ssrf) | **web_fetch** rejects `http://` and non-https schemes |
| [AC-11.009](#ac-11-009) | [REQ-11.014](ep-requirements.md#web_fetch--ssrf) | **web_fetch** rejects loopback and private-range targets |
| [AC-11.010](#ac-11-010) | [REQ-11.015](ep-requirements.md#web_fetch--ssrf) | **web_fetch** rejects redirect to `http` or over max hops |
| [AC-11.011](#ac-11-011) | [REQ-11.016](ep-requirements.md#web_fetch--ssrf), [REQ-11.017](ep-requirements.md#config--limits) | **web_fetch** truncates or stops at max body bytes and honors timeout |
| [AC-11.012](#ac-11-012) | [REQ-11.018](ep-requirements.md#config--limits) | **web_search** upstream call respects configured timeout |
| [AC-11.013](#ac-11-013) | [REQ-11.021](ep-requirements.md#security--observability)–[REQ-11.023](ep-requirements.md#security--observability) | Brave API key bytes do not appear in tool result strings built for errors; logs use bounded fields |
| [AC-11.014](#ac-11-014) | [REQ-11.022](ep-requirements.md#security--observability) | Validation failures return deterministic error messages without stack traces in tool output |
| [AC-11.015](#ac-11-015) | [REQ-11.024](ep-requirements.md#testing) | Automated tests cover SSRF helpers, scheme checks, cache, **web_fetch** over local HTTPS test server |
| [AC-11.016](#ac-11-016) | [REQ-11.025](ep-requirements.md#tools--providers), [REQ-11.026](ep-requirements.md#tools--providers), [REQ-11.024](ep-requirements.md#testing) | **Fallback search provider** runs after **primary search provider** failure and returns structured results |

---

## Acceptance criteria

### AC-11.001

**AC-11.001** (Trace: [REQ-11.001](ep-requirements.md#tools--providers), [REQ-11.002](ep-requirements.md#tools--providers), [REQ-11.019](ep-requirements.md#config--limits), [REQ-11.020](ep-requirements.md#config--limits))

Given `web_tools.enabled` is true and all required `web_tools` fields are valid  
When PersonalAssistant starts  
Then **web_search** and **web_fetch** SHALL be registered on the native tool registry  

Given `web_tools.enabled` is true and a required numeric bound is zero or negative  
When PersonalAssistant loads configuration  
Then startup SHALL fail with a configuration error  

---

### AC-11.002

**AC-11.002** (Trace: [REQ-11.003](ep-requirements.md#tools--providers))

Given **web_search** is invoked  
When the query parameter is empty or contains only whitespace  
Then the tool SHALL return a structured error and SHALL NOT call an upstream search endpoint  

---

### AC-11.003

**AC-11.003** (Trace: [REQ-11.004](ep-requirements.md#tools--providers), [REQ-11.006](ep-requirements.md#tools--providers), [REQ-11.008](ep-requirements.md#tools--providers))

Given the active search provider is Brave Search and the API key file contains a non-empty key  
When **web_search** runs against a test HTTP server that returns a valid Brave-style JSON payload  
Then the tool result SHALL include at least one item with non-empty title, URL, and snippet fields  

---

### AC-11.004

**AC-11.004** (Trace: [REQ-11.005](ep-requirements.md#tools--providers), [REQ-11.008](ep-requirements.md#tools--providers))

Given the active search provider is DuckDuckGo  
When **web_search** runs against a test HTTP server that returns a minimal DuckDuckGo HTML results page  
Then the tool result SHALL include at least one item with non-empty title, URL, and snippet fields  

---

### AC-11.005

**AC-11.005** (Trace: [REQ-11.007](ep-requirements.md#tools--providers))

Given the active search provider is Brave Search and the API key file is missing  
When **web_search** is invoked  
Then the tool SHALL return a structured error that names the failure class and SHALL NOT include the API key contents  

---

### AC-11.006

**AC-11.006** (Trace: [REQ-11.009](ep-requirements.md#search-cache)–[REQ-11.012](ep-requirements.md#search-cache))

Given identical normalized query and provider and TTL not expired  
When **web_search** is invoked twice  
Then the second invocation SHALL NOT perform a second upstream HTTP request  

Given TTL elapsed between invocations  
When **web_search** is invoked again  
Then a new upstream HTTP request SHALL occur  

Given the cache holds the maximum configured entries and a new distinct key is inserted  
When eviction runs  
Then the cache size SHALL remain at or below the maximum entry count  

---

### AC-11.007

**AC-11.007** (Trace: [REQ-11.010](ep-requirements.md#search-cache))

Given two queries that differ after trim and case-fold normalization  
When **web_search** runs for each  
Then upstream requests SHALL occur for each distinct cache key  

Given the same normalized query and **primary search provider**  
When **web_search** runs once with a **fallback search provider** configured and once without  
Then the two invocations SHALL use distinct cache keys  

---

### AC-11.008

**AC-11.008** (Trace: [REQ-11.013](ep-requirements.md#web_fetch--ssrf))

Given a **web_fetch** URL with scheme `http`  
When **web_fetch** validates the URL  
Then the tool SHALL return a structured error  

---

### AC-11.009

**AC-11.009** (Trace: [REQ-11.014](ep-requirements.md#web_fetch--ssrf))

Given a **web_fetch** URL whose host resolves to `127.0.0.1` or a private IPv4 address used in tests  
When **web_fetch** validates the target  
Then the tool SHALL return a structured error without opening a TCP connection to that target  

---

### AC-11.010

**AC-11.010** (Trace: [REQ-11.015](ep-requirements.md#web_fetch--ssrf))

Given an HTTPS response with `Location: http://example.com/`  
When **web_fetch** follows redirects  
Then the tool SHALL reject the fetch with a structured error  

Given more redirects than the configured maximum  
When **web_fetch** processes the chain  
Then the tool SHALL reject the fetch with a structured error  

---

### AC-11.011

**AC-11.011** (Trace: [REQ-11.016](ep-requirements.md#web_fetch--ssrf), [REQ-11.017](ep-requirements.md#config--limits))

Given a response body larger than the configured maximum byte count  
When **web_fetch** reads the body  
Then the stored result SHALL not exceed the configured maximum  

Given a server that delays beyond the configured **web_fetch** timeout  
When **web_fetch** runs  
Then the tool SHALL fail with a timeout class error  

---

### AC-11.012

**AC-11.012** (Trace: [REQ-11.018](ep-requirements.md#config--limits))

Given a search upstream that sleeps longer than the configured **web_search** timeout  
When **web_search** runs  
Then the tool SHALL fail with a timeout class error  

---

### AC-11.013

**AC-11.013** (Trace: [REQ-11.021](ep-requirements.md#security--observability)–[REQ-11.023](ep-requirements.md#security--observability))

Given Brave Search API key file content is a known test secret string  
When **web_search** returns an error or success path that surfaces text to the agent  
Then the returned tool string SHALL NOT contain the secret substring  

---

### AC-11.014

**AC-11.014** (Trace: [REQ-11.022](ep-requirements.md#security--observability))

Given a **web_fetch** URL that fails SSRF validation  
When the tool returns  
Then the error message SHALL be a short deterministic classification without a full Go stack trace in the tool output string  

---

### AC-11.015

**AC-11.015** (Trace: [REQ-11.024](ep-requirements.md#testing))

Given the automated test suite for EP-011  
When tests run in default CI configuration without live Brave or DuckDuckGo endpoints  
Then tests SHALL still validate scheme rules, SSRF classification, cache behaviour, and **web_fetch** against a local HTTPS test server  

---

### AC-11.016

**AC-11.016** (Trace: [REQ-11.025](ep-requirements.md#tools--providers), [REQ-11.026](ep-requirements.md#tools--providers), [REQ-11.024](ep-requirements.md#testing))

Given **web_tools.search** names a **primary search provider** and a distinct **fallback search provider**  
When the **primary search provider** **upstream search request** fails with an outcome listed in the **system design artefact** and the **fallback search provider** **upstream search request** succeeds  
Then **web_search** SHALL return structured ranked items from the **fallback search provider** and SHALL have attempted the **primary search provider** first  

Given **web_tools.search** names a **fallback search provider** equal to the **primary search provider**  
When PersonalAssistant loads configuration  
Then startup SHALL fail with a configuration error  

---
