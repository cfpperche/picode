#!/usr/bin/env bash
# make ci-scoped — the gates this worktree's diff can break (ADR-0086).
#
# `make ci` is the whole matrix and stays the gate for the merge on main.
# Iterating in a worktree paid the same four minutes for a CSS fix, and
# with several sessions doing it at once main moved during every run
# (44 catch-up merges in nine days). The diff against main decides:
#   go        → go test on the changed packages and everything that imports them
#   web       → frontend tests + the embedded build
#   packages  → frontend/package tests
#   docs      → docs-check + public site + Vale
#   metadata  → the fast gates only (fmt, vet, hooks)
# Anything that shapes the gates themselves (go.mod, Makefile, hooks,
# scripts/ci-scope.mjs) runs `make ci`. CI_SCOPE_BASE overrides the base ref.
set -euo pipefail
cd "$(dirname "$0")/.."

base_ref=${CI_SCOPE_BASE:-main}
base=$(git merge-base "$base_ref" HEAD 2>/dev/null || true)
paths=$(
  { [ -n "$base" ] && git diff --name-only "$base" HEAD; git diff --name-only; git diff --name-only --cached; git ls-files --others --exclude-standard; } 2>/dev/null | sort -u
)
eval "$(printf '%s\n' "$paths" | node scripts/ci-scope.mjs --local)"

# A green run is remembered per tree (ADR-0105): `make close` reuses it when
# the tree has not changed since, instead of paying the gates twice.
stamp() {
  [ -z "$(git status --porcelain)" ] || return 0
  git rev-parse 'HEAD^{tree}' > "$(git rev-parse --git-dir)/picode-ci-scoped.ok" 2>/dev/null || true
}

ran=()
if [ -n "$SCOPE_FULL" ]; then
  echo "ci-scoped: gate-shaping or unknown paths changed — running the full matrix"
  make --no-print-directory ci
  stamp
  echo "ci-scoped: PASS (full)"
  exit 0
fi

make --no-print-directory hooks-check fmt-check vet
ran+=("fmt" "vet" "hooks")

if [ -n "$SCOPE_GO" ]; then
  # Changed directories → packages (walking up for embedded files such as
  # SQL migrations) → every package whose dependency list names one.
  changed=""
  for p in $SCOPE_GO_PATHS; do
    d=$(dirname "$p")
    while [ "$d" != "." ]; do
      if pkg=$(go list "./$d" 2>/dev/null); then changed="$changed $pkg"; break; fi
      d=$(dirname "$d")
    done
  done
  changed=$(printf '%s\n' $changed | sort -u | tr '\n' ' ')
  pkgs=$(go list -f '{{.ImportPath}} {{join .Deps " "}}' ./... | awk -v changed="$changed" '
    BEGIN { n = split(changed, c, " "); for (i = 1; i <= n; i++) if (c[i] != "") want[c[i]] = 1 }
    { for (i = 1; i <= NF; i++) if ($i in want) { print $1; next } }')
  if [ -z "$pkgs" ]; then
    echo "ci-scoped: go paths changed but no package resolved — running go test ./..."
    pkgs="./..."
  fi
  echo "ci-scoped: go test $(printf '%s\n' $pkgs | wc -l | tr -d ' ') package(s)"
  ./scripts/go-test.sh $pkgs
  ran+=("go[$(printf '%s\n' $pkgs | wc -l | tr -d ' ')]")
fi

if [ -n "$SCOPE_WEB" ] || [ -n "$SCOPE_PACKAGES" ]; then
  make --no-print-directory test-js
  ran+=("test-js")
fi
if [ -n "$SCOPE_WEB" ]; then
  make --no-print-directory build
  ran+=("build")
fi
if [ -n "$SCOPE_DOCS" ]; then
  make --no-print-directory ci-docs vale
  ran+=("docs")
fi
if [ -n "$SCOPE_METADATA" ] && [ ${#ran[@]} -eq 3 ]; then
  ran+=("metadata")
fi

stamp
echo "ci-scoped: PASS ($(IFS=, ; echo "${ran[*]}"); $SCOPE_COUNT path(s) vs $base_ref)"
