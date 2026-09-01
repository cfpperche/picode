#!/usr/bin/env bash
# Points this clone's git at .githooks (AGENTS.md §5) and refuses to be
# silent when someone redirected hooks elsewhere.
#
# The comparison resolves paths on purpose: `core.hooksPath` is a path-type
# config, so git hands back an absolute value when read from a linked
# worktree even though it was written as `.githooks`. Comparing the literal
# string failed inside every worktree — a green repo reported as broken.
set -euo pipefail

# The main worktree's root is the parent of the common git dir; hooks live
# there, and a relative value is resolved against it by git.
common=$(git rev-parse --path-format=absolute --git-common-dir)
root=$(dirname "$common")
expected="$root/.githooks"

resolve() {
  case "$1" in
    /*) printf '%s\n' "$1" ;;
    ~*) printf '%s\n' "${1/#\~/$HOME}" ;;
    *)  printf '%s\n' "$root/$1" ;;
  esac
}

current=$(git config --get core.hooksPath || true)

if [ -z "$current" ]; then
  git config core.hooksPath .githooks
  echo "git hooks enabled (.githooks): the repository root stays on main"
  exit 0
fi

# realpath is not everywhere; fall back to the textual path.
norm() { realpath -m "$1" 2>/dev/null || printf '%s\n' "$1"; }

if [ "$(norm "$(resolve "$current")")" = "$(norm "$expected")" ]; then
  exit 0
fi

cat >&2 <<EOF
core.hooksPath points at '$current', not the repo's .githooks — this clone's
worktree guards are disabled (AGENTS.md §5). Fix it deliberately:
    git config core.hooksPath .githooks
EOF
exit 1
