#!/usr/bin/env bash
# make close — the mechanical end of a worktree session (ADR-0086).
#
# What used to be twenty-odd agent turns at peak context: run the gates the
# diff can break (or reuse the green run this exact tree already had),
# regenerate the committed artifacts the diff invalidated (OpenAPI, llms.txt),
# confirm main can fast-forward, and print the summary the closing docs are
# written from. It never merges, never deploys, never writes prose. Public
# captures refresh at `make deploy` (ADR-0105), not here: 85 capture commits
# in three days came from this step, a third of them for unchanged pixels.
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

# Reuse the green run this worktree already paid for, when nothing the gates
# read has changed since (ADR-0105 for the identical tree, ADR-0124 for the
# catch-up merge that brought unrelated work). The verdict and its reason come
# from one place; this script prints them and runs the gates otherwise.
if verdict=$(node scripts/ci-scope-reuse.mjs --check); then
  echo "close: $verdict"
else
  echo "close: $verdict"
  ./scripts/ci-scoped.sh || exit 1
fi

base=$(git merge-base main HEAD)
changed=$(git diff --name-only "$base" HEAD)

# Generated docs artifacts travel with the code that changed them.
if printf '%s\n' "$changed" | grep -qE '^(internal/|cmd/|docs-site/)'; then
  make --no-print-directory openapi llms >/dev/null || exit 1
  if [ -n "$(git status --porcelain -- docs-site/public/api/openapi.json docs-site/public/llms.txt)" ]; then
    git add docs-site/public/api/openapi.json docs-site/public/llms.txt
    git commit -q -m "docs: regenerate OpenAPI and llms.txt" && echo "close: committed regenerated OpenAPI/llms.txt"
  fi
fi

if [ -n "$(git status --porcelain)" ]; then
  echo "close: the gates left changes behind — inspect before merging:" >&2
  git status --short >&2
  exit 1
fi

echo
./scripts/close-summary.sh
