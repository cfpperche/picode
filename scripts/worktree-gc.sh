#!/usr/bin/env bash
# make worktree-gc — remove the worktrees nobody is using (ADR-0086).
#
# The contract says "remove the worktree after the merge"; in practice eight
# of ten trees on disk belonged to merged branches (1.4 GB each). A tree is
# removed only when ALL of these hold: its branch is merged into main, its
# working tree is clean, and nothing under it (outside node_modules) was
# written in the last hour — a session that just created a tree from main
# has a merged, clean, but freshly written tree. FORCE=1 drops the idle
# check. Dirty or unmerged trees are listed, never touched.
set -uo pipefail
cd "$(dirname "$0")/.."
root=$(pwd)

git worktree prune
git worktree list --porcelain | awk '/^worktree /{p=$2} /^branch /{print p, $2}' | while read -r path ref; do
  [ "$path" = "$root" ] && continue
  branch=${ref#refs/heads/}
  if ! git merge-base --is-ancestor "$branch" main 2>/dev/null; then
    echo "keep   $path  ($branch: not merged)"
    continue
  fi
  if [ -n "$(git -C "$path" status --porcelain 2>/dev/null)" ]; then
    echo "keep   $path  ($branch: dirty)"
    continue
  fi
  if [ -z "${FORCE:-}" ] && [ -n "$(find "$path" -path '*/node_modules' -prune -o -type f -newermt '-60 minutes' -print 2>/dev/null | head -1)" ]; then
    echo "keep   $path  ($branch: written in the last hour; FORCE=1 to remove)"
    continue
  fi
  if git worktree remove "$path" 2>/dev/null; then
    git branch -d "$branch" >/dev/null 2>&1 && echo "removed $path  ($branch merged, branch deleted)" || echo "removed $path  ($branch merged; branch kept)"
  else
    echo "keep   $path  (git refused to remove it)"
  fi
done
