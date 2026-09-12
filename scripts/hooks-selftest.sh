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

# 2b. Content integrity for the living docs (parallel-session clobbering).
#     A CHANGELOG.md whose first line is the handoff header (or the reverse)
#     means another session wrote into this worktree — the commit must die.
if (cd "$repo/wt" && printf '# Handoff — living project state\n\nbody\n' > CHANGELOG.md && git add CHANGELOG.md && git commit -q -m clobber 2>/dev/null); then bad "clobbered CHANGELOG refused" "a handoff copy was committed as the changelog"; else ok "clobbered CHANGELOG refused"; fi
(cd "$repo/wt" && git reset -q && git checkout -q -- CHANGELOG.md 2>/dev/null; rm -f CHANGELOG.md)
if (cd "$repo/wt" && mkdir -p docs && printf '# Handoff — living project state\n\nbody\n' > docs/handoff.md && git add docs/handoff.md && git commit -q -m "handoff committed" 2>/dev/null); then bad "committed board refused" "a worktree committed the generated board"; else ok "committed board refused"; fi
# The refusal leaves the path staged; unstage it, or every later row inherits it.
(cd "$repo/wt" && git reset -q 2>/dev/null; rm -f docs/handoff.md)

# 2c. The board is generated, not written (ADR-0123): even a forced `git add
#     -f` of docs/handoff.md dies, so a stale copy can never reach main, and a
#     topic file is the way to add an item.
if (cd "$repo/wt" && mkdir -p docs/handoff/open && printf '# Terminal\n\n## Debts\n\n- a debt\n' > docs/handoff/open/terminal.md && git add docs/handoff/open/terminal.md && git commit -q -m "topic file" 2>/dev/null); then ok "an open-topic file is allowed"; else bad "an open-topic file is allowed" "the documented place for a debt is blocked"; fi
if (cd "$repo/wt" && git add -f docs/handoff.md && git commit -q -m "forced board" 2>/dev/null); then bad "forced board commit refused" "git add -f put the generated board in a commit"; else ok "forced board commit refused"; fi
(cd "$repo/wt" && git reset -q 2>/dev/null; rm -f docs/handoff.md)

# 2e. Whitespace errors stop at the commit (ADR-0105), not in an unread CI run.
if (cd "$repo/wt" && printf 'trailing blank \n' > ws.txt && git add ws.txt && git commit -q -m ws 2>/dev/null); then bad "trailing whitespace refused" "a whitespace error was committed"; else ok "trailing whitespace refused"; fi
if (cd "$repo/wt" && printf 'clean\n' > ws.txt && git add ws.txt && git commit -q -m ws 2>/dev/null); then ok "clean file allowed"; else bad "clean file allowed" "the whitespace check refuses a clean file"; fi

# 2f. The changelog is assembled from fragments (ADR-0105). The assembler is
# copied in so the hook also parses staged fragments here, as it does in a
# real checkout.
for d in "$repo" "$repo/wt"; do mkdir -p "$d/scripts" && cp "$HOOKS/../scripts/changelog-assemble.mjs" "$d/scripts/"; done
if (cd "$repo/wt" && printf '# Changelog\n\n## [Unreleased]\n\n- edited on a branch\n' > CHANGELOG.md && git add CHANGELOG.md && git commit -q -m "changelog on a branch" 2>/dev/null); then bad "CHANGELOG edit on a branch refused" "a branch edited CHANGELOG.md directly"; else ok "CHANGELOG edit on a branch refused"; fi
(cd "$repo/wt" && git reset -q && rm -f CHANGELOG.md)
if (cd "$repo/wt" && mkdir -p docs/changelog.d && printf '### Added\n- a thing\n' > docs/changelog.d/wt.md && git add docs/changelog.d/wt.md && git commit -q -m "fragment" 2>/dev/null); then ok "changelog fragment on a branch allowed"; else bad "changelog fragment on a branch allowed" "the fragment flow is blocked"; fi
if (cd "$repo/wt" && printf 'no section heading\n' > docs/changelog.d/bad.md && git add docs/changelog.d/bad.md && git commit -q -m "bad fragment" 2>/dev/null); then bad "malformed fragment refused" "a fragment that make changelog cannot parse was committed"; else ok "malformed fragment refused"; fi
(cd "$repo/wt" && git reset -q && rm -f docs/changelog.d/bad.md)
if (mkdir -p docs/changelog.d && printf '### Added\n- on main\n' > docs/changelog.d/main.md && git add docs/changelog.d/main.md && git commit -q -m "fragment on main" 2>/dev/null); then bad "fragment written on main refused" "a fragment was committed on main in the root"; else ok "fragment written on main refused"; fi
git reset -q; rm -rf docs/changelog.d
PICODE_ALLOW_SWITCH=1 git merge -q --ff-only feat/wt2 >/dev/null 2>&1 || PICODE_ALLOW_SWITCH=1 git merge -q feat/wt2 -m "bring the fragment" >/dev/null 2>&1
if [ -f docs/changelog.d/wt.md ] && printf '# Changelog\n\n## [Unreleased]\n\n### Added\n\n- a thing\n' > CHANGELOG.md && git rm -q docs/changelog.d/wt.md && git add CHANGELOG.md && git commit -q -m "assemble" 2>/dev/null; then ok "assembly on main (fragment deleted) allowed"; else bad "assembly on main (fragment deleted) allowed" "make changelog cannot commit its result"; fi
if (printf '# Changelog\n\n## [Unreleased]\n\n- typed on main\n' > CHANGELOG.md && git add CHANGELOG.md && git commit -q -m "changelog on main" 2>/dev/null); then bad "bare CHANGELOG edit on main refused" "main took a direct changelog edit"; else ok "bare CHANGELOG edit on main refused"; fi
git reset -q; git checkout -q -- CHANGELOG.md 2>/dev/null
if (printf '# Changelog\n\n## [Unreleased]\n\n## [0.2.0] - 2026-09-10\n\n### Added\n\n- a thing\n' > CHANGELOG.md && git add CHANGELOG.md && git commit -q -m "release 0.2.0" 2>/dev/null); then ok "release cut on main allowed"; else bad "release cut on main allowed" "the hook blocks the version heading the release process writes"; fi

# 2g. Handoff state never lands on main directly (ADR-0105).
if (mkdir -p docs && printf '# Handoff — living project state\n\nrecorded the merge\n' > docs/handoff.md && git add -f docs/handoff.md && git commit -q -m "handoff on main" 2>/dev/null); then bad "board on main refused" "main took a generated board"; else ok "board on main refused"; fi
git reset -q; rm -rf docs
if (mkdir -p docs/handoff && printf 'note\n' > docs/handoff/2026-01-01-x.md && git add docs/handoff && git commit -q -m "note on main" 2>/dev/null); then bad "session note on main refused" "main took a session note directly"; else ok "session note on main refused"; fi
git reset -q; rm -rf docs
if (date > root2.txt && git add root2.txt && git commit -q -m "ordinary main commit" 2>/dev/null); then ok "ordinary commit on main still allowed"; else bad "ordinary commit on main still allowed" "the living-docs guard is too broad"; fi

# 3. Escape hatch, return home, and the pre-commit belt when off main.
if PICODE_ALLOW_SWITCH=1 git switch -q feat/existing 2>/dev/null; then ok "PICODE_ALLOW_SWITCH override works"; else bad "PICODE_ALLOW_SWITCH override works" "override refused"; fi
date > off.txt && git add off.txt
if git commit -q -m "feature commit in root" 2>/dev/null; then bad "feature commit in root refused" "committed on $(git branch --show-current)"; else ok "feature commit in root refused"; fi
git reset -q HEAD off.txt
if git switch -q main 2>/dev/null; then ok "returning to main always allowed"; else bad "returning to main always allowed" "cannot get home"; fi
# 3b. Main never rewinds (2026-09-12: a stale-ref fast-forward erased merged
#     work). Forward is the only direction; a rollback needs its own flag.
old_tip=$(git rev-parse main)
older=$(git rev-parse main~1)
# reset --hard fires the reference transaction; update-ref does not (a git
# limitation, not ours — agents drive the repo with reset/merge/switch).
if git reset -q --hard "$older" 2>/dev/null; then bad "rewind via reset --hard refused" "main moved to $older"; else ok "rewind via reset --hard refused"; fi
[ "$(git rev-parse main)" = "$old_tip" ] && ok "main did not move on the refused rewind" || bad "main did not move on the refused rewind" "on $(git rev-parse main)"
if PICODE_ALLOW_MAIN_REWIND=1 git reset -q --hard "$older" 2>/dev/null; then ok "PICODE_ALLOW_MAIN_REWIND override works"; else bad "PICODE_ALLOW_MAIN_REWIND override works" "the deliberate rollback was blocked"; fi
if git merge -q --ff-only "$old_tip" 2>/dev/null; then ok "forward fast-forward still allowed"; else bad "forward fast-forward still allowed" "getting back to the tip broke"; fi
[ "$(git rev-parse main)" = "$old_tip" ] && ok "back at the tip after the override round-trip" || bad "back at the tip" "on $(git rev-parse main)"

# 3c. The incident replay: a fast-forward from a stale ref to a DIVERGENT
#     tip looks legitimate to the process that runs it and still erases
#     main's tip (2026-09-12).
(cd "$repo/wt" && git switch -q -c stale "$older" && date > divergent.txt && git add divergent.txt && git commit -q -m divergent)
if git merge -q --ff-only stale 2>/dev/null; then bad "divergent stale ff refused" "main moved to the divergent tip"; else ok "divergent stale ff refused"; fi
[ "$(git rev-parse main)" = "$old_tip" ] && ok "main survived the divergent ff" || bad "main survived the divergent ff" "on $(git rev-parse main)"
(cd "$repo/wt" && git switch -q --detach && git branch -q -D stale)


echo
if [ "$fails" -eq 0 ]; then
  echo "  git guards verified: the root stays on main, worktrees stay free, living docs travel by fast-forward."
  exit 0
fi
echo "  $fails guard check(s) failed — .githooks is not protecting this repo." >&2
exit 1
