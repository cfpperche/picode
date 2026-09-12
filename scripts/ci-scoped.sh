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

# Say which tree and branch this is (AGENTS.md §5): a session that edits the
# wrong checkout wastes its own work and confuses everyone else's status.
printf 'ci-scoped: %s on %s\n' "$(pwd)" "$(git branch --show-current 2>/dev/null || echo '(no git)')"

base_ref=${CI_SCOPE_BASE:-main}
base=$(git merge-base "$base_ref" HEAD 2>/dev/null || true)
paths=$(
  { [ -n "$base" ] && git diff --name-only "$base" HEAD; git diff --name-only; git diff --name-only --cached; git ls-files --others --exclude-standard; } 2>/dev/null | sort -u
)
eval "$(printf '%s\n' "$paths" | node scripts/ci-scope.mjs --local)"

# A green run is remembered by what it covered (ADR-0105, ADR-0124): `make
# close` reuses it when neither the tree nor the content the gates read
# changed — a catch-up merge that brought unrelated work does not re-test.
stamp() {
  node scripts/ci-scope-reuse.mjs --write --base "$base" \
    ${stamped_pkgs:+--packages "$stamped_pkgs"} ${SCOPE_FULL:+--full} || true
}

ran=()
stamped_pkgs=""
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
  # What the run read, for `make close`'s reuse decision (ADR-0124): the tested
  # packages plus their transitive dependencies, and nothing else.
  stamped_pkgs=$(go list -deps -f '{{.ImportPath}} {{.Dir}}' $pkgs 2>/dev/null | awk -v mod="$(head -1 go.mod | awk '{print $2}')" '$1 ~ "^"mod { print $1 }' | sort -u | tr '\n' ' ')
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
