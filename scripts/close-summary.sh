#!/usr/bin/env bash
# make close-summary — everything the closing docs need, and nothing else
# (ADR-0086). The handoff and changelog for a branch are written from THIS
# output by a fresh, small session (or by the working session after
# compaction), not from a context that has read the whole repository.
set -euo pipefail
cd "$(dirname "$0")/.."

branch=$(git branch --show-current)
base=$(git merge-base main HEAD 2>/dev/null || git rev-parse HEAD~1)
today=$(date +%F)
slug=$(printf '%s' "${branch#*/}" | tr -c 'a-zA-Z0-9\n' '-' | sed 's/-*$//')

echo "# close-summary — $branch"
echo
echo "date: $today"
echo "base: $(git rev-parse --short "$base") (main)   head: $(git rev-parse --short HEAD)"
echo
echo "## Commits (oldest first)"
git log --reverse --format='- %h %s' "$base"..HEAD
echo
echo "## Diff"
git diff --stat "$base"..HEAD | tail -n 25
echo
echo "## Scope"
git diff --name-only "$base"..HEAD | node scripts/ci-scope.mjs --local | grep -E '^SCOPE_(FULL|GO|WEB|PACKAGES|DOCS|METADATA)=' | sed 's/^SCOPE_//' | tr '\n' ' '
echo
echo
echo "## Docs still owed by this branch"
owed=0
if ! git diff --name-only "$base"..HEAD | grep -q '^docs/changelog\.d/'; then
  if git diff --name-only "$base"..HEAD | grep -qE '^(web/|internal/|cmd/|packages/)'; then
    echo "- docs/changelog.d/$slug.md: no fragment yet — user-visible change? '### Added' / '### Fixed' + one line (never edit CHANGELOG.md on a branch)"
    owed=1
  fi
fi
if ! git diff --name-only "$base"..HEAD | grep -q '^docs/handoff/'; then
  echo "- docs/handoff/$today-$slug.md: not written — one file, ≤ 25 lines (see .pi/skills/handoff-update/SKILL.md)"
  owed=1
fi
if git diff --name-only "$base"..HEAD | grep -qE '^docs/decisions/[0-9]{4}-'; then
  git diff --name-only "$base"..HEAD | grep -qx 'docs/decisions/README.md' || { echo "- docs/decisions/README.md: new ADR is not in the index"; owed=1; }
fi
[ "$owed" -eq 0 ] && echo "- nothing: changelog/handoff already travel with the code"
echo
echo "## Merge"
if git merge-base --is-ancestor main HEAD; then
  echo "- main can fast-forward to $branch:  cd /home/goat/picode && git merge --ff-only $branch && make ci"
else
  echo "- main moved: git merge main (in this worktree), rerun make close, then fast-forward from the root"
fi
echo "- after the merge: git worktree remove .worktrees/<name>; git branch -d $branch   (or: make worktree-gc)"
echo "- deploy is NOT part of closing: the owner runs 'make deploy' from the root when they want it (ADR-0105)"
echo "- this session ends here; the next branch starts in a new one (ADR-0105)"
