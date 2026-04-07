# validate — SDLC Validation Tool

Multi-purpose validation tool for the PersonalAssistant SDLC pipeline.

## Current Validators

### AC (Acceptance Criteria) Validation

Validates that all Acceptance Criteria from an epic's `ep-acceptance-criteria.md` are covered by tests.

**All Epics (Default):**
```bash
make build
./bin/validate
```

**Single Epic:**
```bash
./bin/validate EP-009
```

**With JSON Output:**
```bash
./bin/validate --json
./bin/validate --json EP-009
```

Output:
```
🔍 Validating AC coverage for all 9 epics...

📋 Epic Validation Summary

Epic       Coverage     Status
────────────────────────────────────
✓ EP-001        95%
✓ EP-004        88%
✗ EP-009        61%
────────────────────────────────────

❌ OVERALL: 84 covered, 2 deferred, 113 total (76.1%)

❌ AC not covered by tests (project-wide): 27

EP-009
  • AC-09.001
  ...

Tip: run `./bin/validate EP-XXX` for per-AC detail and test refs.
```

**Coverage Declaration:**

Mark tests with `// Covers AC-XX.YYY` comment:

```go
// Covers AC-09.008: create_tool accepts parameters
func TestCreateToolTool_Run(t *testing.T) {
    // ...
}
```

Supports:
- Single: `// Covers AC-09.001`
- Multiple: `// Covers AC-09.001, AC-09.002`
- Range: `// Covers AC-09.008–013`
- Mixed: `// Covers AC-09.001, AC-09.003–005, AC-09.010`

See [VALIDATION.md](./VALIDATION.md) for full documentation.

## Exit Codes

- **0** — All validations passed ✅
- **1** — Validation failed ❌

## Future Validators

- [ ] REQ (Requirements) traceability
- [ ] Design specification consistency
- [ ] Code coverage thresholds
- [ ] Dependency graph validation
