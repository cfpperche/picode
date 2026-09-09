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

last=$(ls docs/decisions .worktrees/*/docs/decisions 2>/dev/null | grep -oE '^[0-9]{4}' | sort -u | tail -1)
next=$(printf '%04d' $((10#${last:-0} + 1)))
file="docs/decisions/$next-$slug.md"
[ -e "$file" ] && { echo "adr: $file exists" >&2; exit 1; }

awk 'f { print } /^---$/ { f = 1 }' docs/decisions/template.md \
  | sed -e "s/ADR-NNNN: Title/ADR-$next: $title/" -e "s/YYYY-MM-DD/$(date +%F)/" \
  > "$file"
printf '| [%s](%s-%s.md) | %s | proposed |\n' "$next" "$next" "$slug" "$title" >> docs/decisions/README.md
echo "adr: $file (index row appended; status proposed)"
echo "     fill Boundary first — no boundary means it is not an ADR (AGENTS.md)"
