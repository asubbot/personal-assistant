# Epic scope — EP-011 Native web search and HTTPS content fetch (tools)

| Field | Content |
|-------|---------|
| **ID** | EP-011 |
| **Status** | DONE |
| **Title** | Native web search and HTTPS content fetch (tools) |
| **Description** | Add first-class **native** (non-browser) tools for **web search** and **HTTPS-only** fetching of public page content, with **two search providers** from the start—**Brave Search** and **DuckDuckGo**—selectable in configuration, a **simple in-process search result cache** with configurable **TTL** (no database), plus SSRF-safe URL handling, limits, and observability aligned with the existing tool contract. |
| **First version date** | 2026-04-08 |

## Glossary

Terms from the project [scope.md](../../scope.md) glossary apply. Epic-specific terms:

| Term | Definition |
|------|------------|
| **Native tool** | A tool implemented in the Go core using standard HTTP client(s) and parsing; **no** headless browser, **no** remote browser MCP for this epic’s search/fetch path. |
| **web_search (tool)** | A registered tool that returns ranked search results (title, URL, snippet or equivalent) for a query string, using a configured **search provider**. |
| **Search provider** | One of **Brave Search** (Brave Search API) or **DuckDuckGo** (implementation uses DuckDuckGo’s supported or documented programmatic interface chosen at system-design stage; not a second browser). |
| **web_fetch (tool)** | A registered tool that retrieves **public** document body (e.g. HTML or plain text) for a single URL over **HTTPS only**. |
| **HTTPS-only policy** | Only URLs with scheme `https` are accepted for **web_fetch**; `http`, `file`, `data`, IP-literals used to bypass host policy, and other non-HTTPS schemes are rejected at validation. |
| **SSRF mitigation** | Controls that block or restrict requests to private, loopback, link-local, metadata, and other disallowed targets; exact rules are specified in requirements/design. |
| **Search result cache** | An **in-memory** (per process) store of **web_search** responses keyed by a normalized query and **active provider** (and any other inputs defined in design), with a **time-to-live (TTL)** after which entries expire; **no** database or disk persistence for this epic. |

## Scope (features/capabilities)

- **web_search with two providers (day one):** Implement **Brave Search** and **DuckDuckGo** as **first-class** providers. Configuration selects a **primary search provider** and MAY select a distinct **fallback search provider**; **web_search** tries the **primary search provider** first and performs one **upstream search request** on the **fallback search provider** when the **primary search provider** attempt fails per requirements and design.
- **Brave provider:** Uses Brave Search API with API key from configuration or secret path (no key in LLM context or logs); respects Brave’s documented limits and terms.
- **DuckDuckGo provider:** Implements search via a **stable, documented** DuckDuckGo interface (e.g. official or explicitly allowed HTTP API/HTML contract—final choice in system design); **no** dependency on a full browser runtime.
- **In-process search cache (TTL):** **web_search** uses a **simple in-memory** cache in the core process: configurable **TTL**, bounded size or eviction policy to avoid unbounded growth (exact policy in design); cache key includes normalized query and the ordered **primary** / **fallback** provider chain (and design-defined facets). **No** database, **no** shared cache across processes or hosts.
- **web_fetch (HTTPS only):** Accept only `https://` URLs; reject all other schemes. Enforce **SSRF mitigations**, maximum response size, timeouts, and redirect policy (e.g. limit hops, disallow downgrade to HTTP).
- **Tool contract:** Both tools follow the existing PersonalAssistant tool model: name, description, validated input schema, registration in configuration alongside other tools, deterministic error reporting to the agent.
- **Security and privacy:** No secrets in prompts or operator-visible logs; redact or truncate fetch/search payloads in logs per project norms; document threat assumptions (SSRF, abuse, rate limits).
- **Configuration:** Documented keys for **primary** and optional **fallback** provider choice, Brave API key path (when either role uses Brave), timeouts, size limits, **search cache TTL** (and max entries or equivalent cap), and any allow/deny host patterns required by design—without weakening fail-fast validation.
- **Testing:** Unit tests for URL validation, scheme enforcement, and SSRF helpers; integration tests with **HTTP test servers** or recorded fixtures (no live network required in CI by default unless explicitly agreed).

## Success criteria

- **Two search providers:** With valid configuration, **web_search** returns structured results from the **primary search provider** when that attempt succeeds, and from the **fallback search provider** when configured and the **primary search provider** attempt fails per design.
- **Search cache:** Repeated **web_search** calls with the same normalized inputs within **TTL** are served from **in-memory** cache without a new upstream search request; after TTL expiry, the next call fetches fresh results. Tests cover at least one hit and one miss/expiry path.
- **HTTPS-only fetch:** **web_fetch** succeeds for a valid `https` URL to a public test endpoint in integration tests and **rejects** `http://` and other non-HTTPS inputs with a clear error.
- **SSRF:** Documented rules are implemented; automated tests demonstrate blocking of at least loopback and a private-range target (per design).
- **No browser:** Search and fetch paths do not launch or require a headless browser; code review and architecture notes state this explicitly.
- **Regression safety:** Existing tests pass; new behaviour covered at unit and/or integration level per [strategy.md](../../strategy.md).

## Out of scope / deferred

- **Headless browser** automation for search or rendering (e.g. Playwright, Chromium) for this epic.
- **Authenticated** or **cookie-session** browsing of sites behind login.
- **Non-HTTPS** fetch, including `http`→`https` automatic upgrade as a silent workaround (policy is **reject** non-HTTPS input; user may re-paste an `https` URL).
- **Distributed or external cache** (CDN, Redis, shared volume), **fetch** response caching, or **disk-persisted** search history; this epic allows **only** in-process **web_search** result cache with TTL.
- **Distributed rate-limit pool** or paid search tiers beyond what configuration requires for MVP.
- **MCP-based** web search/fetch as the **only** implementation path (MCP may remain optional elsewhere; this epic is **native** tools in core).

## Traceability

- **Scope:** Extends the **Tool** concept in [scope.md](../../scope.md): new tools for external knowledge retrieval while preserving **reliability and security** (validation, security model, logging subsystem awareness).
- **Strategy:** Aligns with [strategy.md](../../strategy.md): testable increments, security checks, unit/integration emphasis; no requirement to add browser-based E2E for the open web in CI.
