#!/usr/bin/env bash
# make close — the mechanical end of a worktree session (ADR-0086).
#
# What used to be twenty-odd agent turns at peak context: run the gates the
# diff can break, regenerate the committed artifacts the diff invalidated
# (OpenAPI, llms.txt, public captures), confirm main can fast-forward, and
# print the summary the closing docs are written from. It never merges,
# never deploys, never writes prose.
set -uo pipefail
cd "$(dirname "$0")/.."

branch=$(git branch --show-current)
if [ "$branch" = "main" ] || [ -z "$branch" ]; then
  echo "close: run from a worktree branch, not main" >&2
  exit 1
fi
if [ -n "$(git status --porcelain)" ]; then
  echo "close: commit or discard the working tree first:" >&2
  git status --short >&2
  exit 1
fi

./scripts/ci-scoped.sh || exit 1

base=$(git merge-base main HEAD)
changed=$(git diff --name-only "$base" HEAD)

# Generated docs artifacts travel with the code that changed them.
if printf '%s\n' "$changed" | grep -qE '^(internal/|cmd/|www/)'; then
  make --no-print-directory openapi llms >/dev/null || exit 1
  if [ -n "$(git status --porcelain -- www/public/api/openapi.json www/public/llms.txt)" ]; then
    git add www/public/api/openapi.json www/public/llms.txt
    git commit -q -m "docs: regenerate OpenAPI and llms.txt" && echo "close: committed regenerated OpenAPI/llms.txt"
  fi
fi

# Public captures: only when the UI changed, only when the fingerprint says so.
if printf '%s\n' "$changed" | grep -q '^web/'; then
  if ! DOCS_STRICT=1 node scripts/docs-check.mjs >/dev/null 2>&1; then
    echo "close: public captures are stale for this diff — recapturing (make docs-shots)"
    make --no-print-directory docs-shots || { echo "close: docs-shots failed; rerun `make docs-shots` (the fixture is flaky on a busy machine)" >&2; exit 1; }
    if [ -n "$(git status --porcelain -- www/img)" ]; then
      git add www/img
      git commit -q -m "docs: refresh public captures" && echo "close: committed refreshed captures"
    fi
  fi
fi

if [ -n "$(git status --porcelain)" ]; then
  echo "close: the gates left changes behind — inspect before merging:" >&2
  git status --short >&2
  exit 1
fi

echo
./scripts/close-summary.sh
