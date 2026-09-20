#!/usr/bin/env bash
# make adr NAME=<short-title> [TITLE="Words"] — seed the next decision record
# (ADR-0105). Allocates the number across the root and every worktree (the
# collision that renumbered 0080/0081/0082 by hand), writes the file from the
# template and appends the index row, so nobody reads the 19 KB index to add
# one line.
set -euo pipefail
cd "$(dirname "$0")/.."

name=${1:?usage: scripts/adr-new.sh <short-title> ["Title words"]}
title=${2:-$name}
slug=$(printf '%s' "$name" | tr 'A-Z' 'a-z' | tr -c 'a-z0-9\n' '-' | sed -E 's/-+/-/g; s/^-//; s/-$//')
[ -n "$slug" ] || { echo "adr: empty slug from '$name'" >&2; exit 1; }

# Every checkout counts: the root and each linked worktree. From inside a
# worktree .worktrees/ is not visible, so the list comes from git itself.
# A listed worktree whose directory is gone (a pruned /tmp scratchpad) must
# not fail the allocation — its names simply contribute nothing.
last=$(git worktree list --porcelain | awk '/^worktree /{ sub(/^worktree /, ""); print $0 "/docs/decisions" }' \
  | while IFS= read -r d; do ls "$d" 2>/dev/null || true; done \
  | grep -oE '^[0-9]{4}' | sort -u | tail -1)
next=$(printf '%04d' $((10#${last:-0} + 1)))
file="docs/decisions/$next-$slug.md"
[ -e "$file" ] && { echo "adr: $file exists" >&2; exit 1; }

awk 'f { print } /^---$/ { f = 1 }' docs/decisions/template.md \
  | sed -e '1{/^$/d;}' -e "s/ADR-NNNN: Title/ADR-$next: $title/" -e "s/YYYY-MM-DD/$(date +%F)/" \
  > "$file"
printf '| [%s](%s-%s.md) | %s | proposed |\n' "$next" "$next" "$slug" "$title" >> docs/decisions/README.md
echo "adr: $file (index row appended; status proposed)"
echo "     fill Boundary first — no boundary means it is not an ADR (AGENTS.md)"
