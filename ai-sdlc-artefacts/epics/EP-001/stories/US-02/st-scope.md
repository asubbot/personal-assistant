# Story scope — US-02 Docker deploy

**Story:** US-02  
**Title:** Docker deploy — run core on DS220+

---

## Formulation

As an operator, I want to run the PersonalAssistant core as a single Docker container (including on Synology DS220+), so that I can deploy with one command.

## Traceability to AC/REQ

| AC | REQ | Summary |
|----|-----|---------|
| [AC-01.003](../../ep-acceptance-criteria.md#ac-01-003) | [REQ-01.001](../../ep-requirements.md#interface-and-deployment), [REQ-01.002](../../ep-requirements.md#interface-and-deployment) | Container runs on x86_64; invalid wiring → error, no serve |
| [AC-01.004](../../ep-acceptance-criteria.md#ac-01-004) | [REQ-01.002](../../ep-requirements.md#interface-and-deployment) | Image builds and runs on DS220+ without code change |
