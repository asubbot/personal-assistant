# Story scope — US-02 Docker deploy

**Story:** US-02  
**Title:** Docker deploy — run core on DS220+

---

## Formulation

As an operator, I want to run the PersonalAssistant core as a single Docker container (including on Synology DS220+), so that I can deploy with one command.

## Traceability to AC/REQ

| AC | REQ | Summary |
|----|-----|---------|
| [AC-003](../../ep-acceptance-criteria.md#ac-003) | [REQ-001](../../ep-requirements.md#interface-and-deployment), [REQ-002](../../ep-requirements.md#interface-and-deployment) | Container runs on x86_64; invalid wiring → error, no serve |
| [AC-004](../../ep-acceptance-criteria.md#ac-004) | [REQ-002](../../ep-requirements.md#interface-and-deployment) | Image builds and runs on DS220+ without code change |
