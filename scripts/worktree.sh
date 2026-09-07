#!/usr/bin/env bash
# make worktree NAME=<name> [BRANCH=<branch>] — an isolated tree that is
# ready to build in seconds (ADR-0086).
#
# `git worktree add` alone left every session paying `npm ci` twice (web and
# www, ~526 MB, about a minute) before its first `make web`. The root's
# node_modules are hardlinked instead (same filesystem, one second): npm
# unlinks before it writes, so a later install in either tree never reaches
# the other. Never symlink them — `git worktree remove` once followed the
# link and emptied the root's install.
set -euo pipefail
cd "$(dirname "$0")/.."

name=${1:?usage: scripts/worktree.sh <name> [branch]}
branch=${2:-feat/$name}
path=".worktrees/$name"

if [ -e "$path" ]; then
  echo "worktree: $path already exists" >&2
  exit 1
fi
git worktree add "$path" -b "$branch" main
for d in web www; do
  if [ -d "$d/node_modules" ] && [ -f "$d/node_modules/.package-lock.json" ]; then
    cp -al "$d/node_modules" "$path/$d/node_modules"
    # The stamp must be newer than the freshly checked-out lockfile, or the
    # next `make web` reinstalls anyway.
    touch "$path/$d/node_modules/.package-lock.json"
  fi
done
echo "worktree ready: $path ($branch)"
echo "  cd $path && make web    # builds in seconds; no npm ci"
