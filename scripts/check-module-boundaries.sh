#!/usr/bin/env bash
# check-module-boundaries.sh — Verify no circular deps and that module boundary rules hold.
# See docs/EP-104/04-system-design.md §2.1 and implementation plan §10.1.
# Exit 0 if all checks pass; non-zero and message otherwise.

set -e

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

MODULE="pa"

# Collect all packages under cmd/pa and internal/
packages=$(go list ./cmd/pa ./internal/... 2>/dev/null | sort -u)
if [ -z "$packages" ]; then
  echo "error: no packages found (go list ./cmd/pa ./internal/...)" >&2
  exit 1
fi

# Build direct internal dependency edges: from -> to (only pa/internal/* or pa/cmd/*)
declare -A edges
cycle_fail=0

for pkg in $packages; do
  case "$pkg" in
    ${MODULE}/internal/*|${MODULE}/cmd/*) ;;
    *) continue ;;
  esac
  imports=$(go list -f '{{range .Imports}}{{.}} {{end}}' "$pkg" 2>/dev/null)
  for imp in $imports; do
    case "$imp" in
      ${MODULE}/internal/*|${MODULE}/cmd/*)
        if [ "$pkg" != "$imp" ]; then
          edges["$pkg $imp"]=1
        fi
        ;;
    esac
  done
done

# Cycle detection (DFS)
has_cycle() {
  local start=$1
  local path_str=$2
  local node
  for node in $packages; do
    if [ "${edges[$start $node]+x}" ]; then
      if [ "$node" = "$start" ]; then
        echo "cycle: $start -> $start"
        return 0
      fi
      if echo " $path_str " | grep -q " $node "; then
        echo "cycle: $path_str -> $node"
        return 0
      fi
      if has_cycle "$node" "$path_str $node"; then
        return 0
      fi
    fi
  done
  return 1
}

for pkg in $packages; do
  case "$pkg" in
    ${MODULE}/internal/*|${MODULE}/cmd/*) ;;
    *) continue ;;
  esac
  if cycle=$(has_cycle "$pkg" "$pkg"); then
    echo "error: circular dependency: $cycle" >&2
    cycle_fail=1
  fi
done

if [ "$cycle_fail" -ne 0 ]; then
  exit 1
fi

# Forbidden edges: adapter (telegram) may only depend on config, core.
# Core may not depend on concrete impls: vector/sqlite, llm/openai (only interfaces).
telegram_allowed=" pa/internal/config pa/internal/core "
core_forbidden=" pa/internal/vector/sqlite pa/internal/llm/openai "

forbidden_fail=0

for pkg in $packages; do
  case "$pkg" in
    ${MODULE}/internal/*|${MODULE}/cmd/*) ;;
    *) continue ;;
  esac
  imports=$(go list -f '{{range .Imports}}{{.}} {{end}}' "$pkg" 2>/dev/null)
  for imp in $imports; do
    case "$imp" in
      ${MODULE}/internal/*|${MODULE}/cmd/*) ;;
      *) continue ;;
    esac
    if [ "$pkg" = "${MODULE}/internal/telegram" ]; then
      if ! echo "$telegram_allowed" | grep -q " $imp "; then
        echo "error: adapter (telegram) must not import $imp (allowed: config, core)" >&2
        forbidden_fail=1
      fi
    fi
    if [ "$pkg" = "${MODULE}/internal/core" ]; then
      if echo "$core_forbidden" | grep -q " $imp "; then
        echo "error: core must not import concrete impl $imp (wiring in cmd/pa)" >&2
        forbidden_fail=1
      fi
    fi
  done
done

if [ "$forbidden_fail" -ne 0 ]; then
  exit 1
fi

echo "module boundaries OK (no cycles, no forbidden edges)"
