#!/usr/bin/env bash
# verify-ai-sdlc-pin.sh — Fail if nested ai-sdlc/ HEAD does not match ai-sdlc.version pin.
# Called from make check. Exit 0 when pin matches; non-zero with remediation hints otherwise.

set -e

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION_FILE="ai-sdlc.version"
SDLCDIR="ai-sdlc"

if [ ! -f "$VERSION_FILE" ]; then
  echo "error: missing $VERSION_FILE" >&2
  exit 1
fi

PIN="$(grep -v '^#' "$VERSION_FILE" | tail -1 | tr -d '[:space:]')"
if [ -z "$PIN" ]; then
  echo "error: no pin in $VERSION_FILE (expected tag or full commit SHA on last non-comment line)" >&2
  exit 1
fi

if [ ! -d "$SDLCDIR/.git" ]; then
  echo "error: missing $SDLCDIR/ git clone" >&2
  echo "  git clone https://github.com/asubbot/ai-sdlc.git $SDLCDIR" >&2
  echo "  git -C $SDLCDIR checkout \"\$(grep -v '^#' $VERSION_FILE | tail -1 | tr -d '[:space:]')\"" >&2
  echo "See docs/installation.md#agentic-sdlc-process-clone-contributors-only" >&2
  exit 1
fi

if ! EXPECTED="$(git -C "$SDLCDIR" rev-parse "${PIN}^{commit}" 2>/dev/null)"; then
  echo "error: pin ${PIN} not found in local $SDLCDIR/ clone" >&2
  echo "  git -C $SDLCDIR fetch" >&2
  echo "  git -C $SDLCDIR checkout \"${PIN}\"" >&2
  exit 1
fi

ACTUAL="$(git -C "$SDLCDIR" rev-parse HEAD)"
if [ "$EXPECTED" != "$ACTUAL" ]; then
  EXPECTED_SHORT="$(git -C "$SDLCDIR" rev-parse --short "$EXPECTED")"
  ACTUAL_SHORT="$(git -C "$SDLCDIR" rev-parse --short "$ACTUAL")"
  ACTUAL_DESC="$(git -C "$SDLCDIR" describe --tags --always 2>/dev/null || true)"
  echo "error: ai-sdlc pin mismatch" >&2
  echo "  pin in $VERSION_FILE: ${PIN}" >&2
  echo "  expected commit: ${EXPECTED_SHORT}" >&2
  if [ -n "$ACTUAL_DESC" ]; then
    echo "  actual HEAD: ${ACTUAL_SHORT} (${ACTUAL_DESC})" >&2
  else
    echo "  actual HEAD: ${ACTUAL_SHORT}" >&2
  fi
  echo "  git -C $SDLCDIR fetch && git -C $SDLCDIR checkout \"${PIN}\"" >&2
  exit 1
fi

echo "ai-sdlc pin OK: ${PIN} ($(git -C "$SDLCDIR" rev-parse --short HEAD))"
