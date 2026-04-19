# EP-030 — Audit report (implementation close-out)

| Field | Content |
|-------|---------|
| **Epic** | EP-030 Remove Hermes text-based tool path |
| **Date** | 2026-04-19 |

## Summary

Implementation removes the Hermes text-tool path, `internal/tooltext`, `ForceJSONOutput` / `supports_json_mode` behaviour, and rejects removed config keys at load. Native `tool_calls` is the only conversation tool execution path. Operator documentation and examples were aligned with EP-030.

## Verification

- `make check` — pass (race-enabled tests).
- `./bin/validate EP-030` — pass (all in-scope ACs traced).

## Notes

- Stage 10 (delegated code review) was not re-run in this session; prior design review artefacts remain under this epic directory. Follow-up review is optional if the owner wants a second pass on the final diff.
