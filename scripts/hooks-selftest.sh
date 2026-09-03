#!/usr/bin/env bash
# Proves the git guards actually guard (AGENTS.md §5). Runs the whole policy
# matrix against a throwaway repository, so it never touches this clone.
#
# Why a functional test and not just "is core.hooksPath set": the guard is
# subtle. `git worktree add -b` writes the new worktree's HEAD through the
# same reference transaction, from the same directory, with the same payload
# as a plain switch — the only difference is the invoking command. An edit
# that loses that distinction either blocks every worktree (nobody can work)
# or blocks nothing (the guard is decorative), and both fail silently.
set -uo pipefail

HOOKS=$(cd "$(dirname "$0")/.." && pwd)/.githooks
fails=0
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

say() { printf '  %-46s %s\n' "$1" "$2"; }
ok()  { say "$1" "ok"; }
bad() { say "$1" "FAIL — $2"; fails=$((fails + 1)); }

for h in reference-transaction pre-commit; do
  [ -f "$HOOKS/$h" ] || { bad "$h present" "missing $HOOKS/$h"; continue; }
  [ -x "$HOOKS/$h" ] || bad "$h executable" "chmod +x .githooks/$h"
done

# reference-transaction landed in git 2.28; without it only pre-commit guards.
gitver=$(git --version | awk '{print $3}')
if [ "$(printf '%s\n2.28.0\n' "$gitver" | sort -V | head -1)" != "2.28.0" ]; then
  bad "git supports reference-transaction" "git $gitver < 2.28 — switch cannot be refused"
fi

repo="$tmp/repo"
# -b main: CI runners have no init.defaultBranch, so a fresh repo starts on
# master — and the pre-commit guard rightly refuses the init commit there,
# which left the repo unborn and broke the two assertions below in cascade
# (seen on ubuntu-24.04 CI 2026-09-03). The script already requires git 2.28.
git init -q -b main "$repo"
cd "$repo" || exit 1
git config user.email selftest@picode.local
git config user.name "hooks selftest"
git config core.hooksPath "$HOOKS"
git commit -q --allow-empty -m init
git branch -q -M main
git branch -q feat/existing

# 1. The root checkout cannot leave main.
if git switch -q -c feat/new 2>/dev/null; then bad "switch -c refused in root" "the checkout moved to $(git branch --show-current)"; else ok "switch -c refused in root"; fi
if git switch -q feat/existing 2>/dev/null; then bad "switch refused in root" "the checkout moved"; else ok "switch refused in root"; fi
if git checkout -q -b feat/new2 2>/dev/null; then bad "checkout -b refused in root" "the checkout moved"; else ok "checkout -b refused in root"; fi
[ "$(git branch --show-current)" = "main" ] && ok "root still on main" || bad "root still on main" "on $(git branch --show-current)"

# 2. Everything the policy must NOT break.
if git worktree add -q "$repo/wt" -b feat/wt 2>/dev/null; then ok "worktree add allowed"; else bad "worktree add allowed" "the guard blocks the flow it demands"; fi
if (cd "$repo/wt" && git switch -q -c feat/wt2 2>/dev/null); then ok "switch inside a worktree allowed"; else bad "switch inside a worktree allowed" "linked worktrees must be free"; fi
if (cd "$repo/wt" && date > f.txt && git add f.txt && git commit -q -m "feature commit" 2>/dev/null); then ok "feature commit in a worktree allowed"; else bad "feature commit in a worktree allowed" "pre-commit is too broad"; fi
if (date > root.txt && git add root.txt && git commit -q -m "main commit" 2>/dev/null); then ok "commit on main in root allowed"; else bad "commit on main in root allowed" "pre-commit blocks the documented flow"; fi
if git checkout -q -- . 2>/dev/null; then ok "checkout -- <path> allowed"; else bad "checkout -- <path> allowed" "file checkout must not be refused"; fi

# 3. Escape hatch, return home, and the pre-commit belt when off main.
if PICODE_ALLOW_SWITCH=1 git switch -q feat/existing 2>/dev/null; then ok "PICODE_ALLOW_SWITCH override works"; else bad "PICODE_ALLOW_SWITCH override works" "override refused"; fi
date > off.txt && git add off.txt
if git commit -q -m "feature commit in root" 2>/dev/null; then bad "feature commit in root refused" "committed on $(git branch --show-current)"; else ok "feature commit in root refused"; fi
git reset -q HEAD off.txt
if git switch -q main 2>/dev/null; then ok "returning to main always allowed"; else bad "returning to main always allowed" "cannot get home"; fi

echo
if [ "$fails" -eq 0 ]; then
  echo "  git guards verified: the root stays on main, worktrees stay free."
  exit 0
fi
echo "  $fails guard check(s) failed — .githooks is not protecting this repo." >&2
exit 1
