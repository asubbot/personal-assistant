# Project-level audit (stage 11)

**Date and time of creation:** 2026-04-21 (UTC)

**Purpose:** Project-wide audit summary per pipeline stage 11 (project-level §2a–§3a in `ai-sdlc/specification/pipeline.spec.md` and stage-11 skill): epic inventory and last recorded stage-11 artefacts. This file does not substitute a fresh `make check` on the current branch.

**Links (under ai-sdlc-artefacts/ only):** [scope.md](scope.md) · [strategy.md](strategy.md)

---

## Epic summary table


| EP                                 | Name                                                                              | Status                                          | Test coverage | ep_audit-report                                                 |
| ---------------------------------- | --------------------------------------------------------------------------------- | ----------------------------------------------- | ------------- | --------------------------------------------------------------- |
| [EP-001](epics/EP-001/ep-scope.md) | PersonalAssistant MVP                                                             | DONE                                            | 76.1%         | [ep-audit-report (2026-03-16)](epics/EP-001/ep-audit-report.md) |
| [EP-002](epics/EP-002/ep-scope.md) | Automatic memory summarization                                                    | DONE                                            | 73.3%         | [ep-audit-report (2026-04-11)](epics/EP-002/ep-audit-report.md) |
| [EP-003](epics/EP-003/ep-scope.md) | Agent security hardening                                                          | NEW                                             | —             | —                                                               |
| [EP-004](epics/EP-004/ep-scope.md) | Structured tools and Tool-calling API                                             | DONE                                            | 78.1%         | [ep-audit-report (2026-03-18)](epics/EP-004/ep-audit-report.md) |
| [EP-005](epics/EP-005/ep-scope.md) | SSH subsystem execution channel (pa-runner)                                       | NEW                                             | —             | —                                                               |
| [EP-006](epics/EP-006/ep-scope.md) | Tool-call reliability and model escalation                                        | DONE                                            | 78.6%         | [ep-audit-report (2026-03-20)](epics/EP-006/ep-audit-report.md) |
| [EP-007](epics/EP-007/ep-scope.md) | Observability: correlation, local analytics, and metrics                          | NEW                                             | —             | —                                                               |
| [EP-008](epics/EP-008/ep-scope.md) | LLM Parameters Enhancement                                                        | DONE                                            | 79.5%         | [ep-audit-report (2026-03-22)](epics/EP-008/ep-audit-report.md) |
| [EP-009](epics/EP-009/ep-scope.md) | Dynamic Tool Creation with Docker Sandbox                                         | DONE                                            | 77.4%         | [ep-audit-report (2026-03-23)](epics/EP-009/ep-audit-report.md) |
| [EP-010](epics/EP-010/ep-scope.md) | Distributed remote Go tool pipeline                                               | CANCELED (UX is not good for the product)       | —             | —                                                               |
| [EP-011](epics/EP-011/ep-scope.md) | Native web search and HTTPS content fetch (tools)                                 | DONE                                            | 72.9%         | [ep-audit-report (2026-04-09)](epics/EP-011/ep-audit-report.md) |
| [EP-012](epics/EP-012/ep-scope.md) | Telegram HTML formatting and typing indicator                                     | DONE                                            | 73.4%         | [ep-audit-report (2026-04-09)](epics/EP-012/ep-audit-report.md) |
| [EP-013](epics/EP-013/ep-scope.md) | Runtime skills and consolidated system prompt                                     | DONE                                            | 73.8%         | [ep-audit-report (2026-04-10)](epics/EP-013/ep-audit-report.md) |
| [EP-014](epics/EP-014/ep-scope.md) | Sliding session memory window                                                     | DONE                                            | 74.1%         | [ep-audit-report (2026-04-10)](epics/EP-014/ep-audit-report.md) |
| [EP-015](epics/EP-015/ep-scope.md) | Telegram token usage footer                                                       | DONE                                            | 73.4%         | [ep-audit-report (2026-04-14)](epics/EP-015/ep-audit-report.md) |
| [EP-016](epics/EP-016/ep-scope.md) | Manual day notes, write_memory, and vector memory refinement                      | DONE                                            | 73.8%         | [ep-audit-report (2026-04-15)](epics/EP-016/ep-audit-report.md) |
| [EP-017](epics/EP-017/ep-scope.md) | Intent Classifier for Prompt Optimization                                         | DONE                                            | 73.8%         | [ep-audit-report (2026-04-15)](epics/EP-017/ep-audit-report.md) |
| [EP-018](epics/EP-018/ep-scope.md) | Tiered Prompt Cost Reduction                                                      | DONE                                            | 73.8%         | [ep-audit-report (2026-04-15)](epics/EP-018/ep-audit-report.md) |
| [EP-019](epics/EP-019/ep-scope.md) | Scheduled Agent Jobs and Legacy Scheduler Replacement                             | DONE                                            | 73.8%         | [ep-audit-report (2026-04-16)](epics/EP-019/ep-audit-report.md) |
| [EP-020](epics/EP-020/ep-scope.md) | Natural-Language Scheduled Job Creation from Telegram                             | DONE                                            | 73.8%         | [ep-audit-report (2026-04-16)](epics/EP-020/ep-audit-report.md) |
| [EP-021](epics/EP-021/ep-scope.md) | Scheduler routing without a separate gate (main handler + optional runtime skill) | DONE                                            | 73.8%         | [ep-audit-report (2026-04-16)](epics/EP-021/ep-audit-report.md) |
| [EP-022](epics/EP-022/ep-scope.md) | Reliability hardening for local SQLite stores and outbound HTTP timeouts          | DONE                                            | 73.7%         | [ep-audit-report (2026-04-17)](epics/EP-022/ep-audit-report.md) |
| [EP-023](epics/EP-023/ep-scope.md) | Atomic catalog writes for create_tool                                             | DONE                                            | 74.2%         | [ep-audit-report (2026-04-17)](epics/EP-023/ep-audit-report.md) |
| [EP-024](epics/EP-024/ep-scope.md) | Operator documentation for provider roles and safe logging defaults               | DONE                                            | 74.2%         | [ep-audit-report (2026-04-17)](epics/EP-024/ep-audit-report.md) |
| [EP-025](epics/EP-025/ep-scope.md) | Test layout cleanup: E2E separation                                               | DONE                                            | 74.2%         | [ep-audit-report (2026-04-17)](epics/EP-025/ep-audit-report.md) |
| [EP-026](epics/EP-026/ep-scope.md) | Core refactor: tier builders in conversation handler                              | DONE                                            | 74.2%         | [ep-audit-report (2026-04-17)](epics/EP-026/ep-audit-report.md) |
| [EP-027](epics/EP-027/ep-scope.md) | Composition root and application lifecycle                                        | DONE                                            | 73.7%         | [ep-audit-report (2026-04-17)](epics/EP-027/ep-audit-report.md) |
| [EP-028](epics/EP-028/ep-scope.md) | Per-user rate limiting and tier-aware tool round caps                             | CANCEL (Not necessary for one user using model) | —             | —                                                               |
| [EP-029](epics/EP-029/ep-scope.md) | Health, readiness and operator observability surface                              | DONE                                            | 72.8%         | [ep-audit-report (2026-04-18)](epics/EP-029/ep-audit-report.md) |
| [EP-030](epics/EP-030/ep-scope.md) | Remove Hermes text-based tool path                                                | DONE                                            | 72.8%         | [ep-audit-report (2026-04-19)](epics/EP-030/ep-audit-report.md) |
| [EP-031](epics/EP-031/ep-scope.md) | Vector Memory Search Tool                                                         | DONE                                            | 72.9%         | [ep-audit-report (2026-04-21)](epics/EP-031/ep-audit-report.md) |
| [EP-032](epics/EP-032/ep-scope.md) | Specialized Knowledge Search Tools                                                | DONE                                            | 73.0%         | [ep-audit-report (2026-04-21)](epics/EP-032/ep-audit-report.md) |


---

## Notes

- **Epics with `ep-scope.md`:** 32. **With `ep-audit-report.md`:** 27. **No stage-11 report yet:** EP-003, EP-005, EP-007, EP-010 (canceled), EP-028 (canceled).
- **Pipeline §2a:** No epic was **IN_PROGRESS** at audit time, so `**make check` was not run** solely for this project-level table (per skill: run when auditing IN_PROGRESS epics).
- **Test coverage column:** Values are taken from each epic’s `ep-audit-report.md` at the time that report was written (project-wide statement aggregate from that run), not a new measurement on the current commit. For EP-029 and EP-030, the latest terminal run (`make check`, 2026-04-20) reports `total: (statements) 72.8%`, so the table keeps `72.8%`.
- For a **current** quality gate on your branch, run `**make check`** (and `**./bin/validate EP-XXX**` as needed).

