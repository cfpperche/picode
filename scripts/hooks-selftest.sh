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

# 2e-2. Conflict markers in the staged diff (2026-09-18: a conflicted merge
#       resolved with git add -A shipped the markers and broke main's build —
#       twice in one day). Decision table: an added start marker refused; an
#       added end marker refused; the bare `=======` separator refused by
#       the pre-existing whitespace check; a marker quoted mid-line allowed;
#       a marker outside the added lines (context) allowed.
if (cd "$repo/wt" && printf 'x\n<<<<<<< HEAD\ny\n' > cm.txt && git add cm.txt && git commit -q -m cm 2>/dev/null); then bad "added start marker refused" "a conflict start marker was committed"; else ok "added start marker refused"; fi
if (cd "$repo/wt" && printf 'x\n>>>>>>> main\ny\n' > cm.txt && git add cm.txt && git commit -q -m cm 2>/dev/null); then bad "added end marker refused" "a conflict end marker was committed"; else ok "added end marker refused"; fi
if (cd "$repo/wt" && printf 'Title\n=======\nbody\n' > cm.txt && git add cm.txt && git commit -q -m cm 2>/dev/null); then bad "bare separator refused (whitespace check)" "the pre-existing git --check refusal of ======= regressed"; else ok "bare separator refused (whitespace check)"; fi
if (cd "$repo/wt" && printf 'x\necho "<<<<<<< quoted"\ny\n' > cm.txt && git add cm.txt && git commit -q -m cm 2>/dev/null); then ok "marker quoted mid-line allowed" "a string starting with a quote was refused"; else bad "marker quoted mid-line allowed" "a mid-line marker was refused"; fi
(cd "$repo/wt" && printf 'x\n"<<<<<<< quoted in a string"\ny\n' > cm.txt && git add cm.txt && git commit -q -m cm 2>/dev/null) # seed a committed quote for the context row
if (cd "$repo/wt" && printf 'x\n"<<<<<<< quoted in a string"\nz\n' > cm.txt && git add cm.txt && git commit -q -m cm 2>/dev/null); then ok "marker outside the added lines allowed" "an unchanged quoted marker blocked an unrelated edit"; else bad "marker outside the added lines allowed" "the check scans context, not just added lines"; fi
(cd "$repo/wt" && git reset -q 2>/dev/null; rm -f cm.txt)
# …and the incident replay: markers entering through a conflict RESOLUTION
# commit (MERGE_HEAD present), which skips every later check. This row is
# why the marker check sits before the merge early-exit.
(cd "$repo/wt" \
  && git switch -q -c cm-a && printf 'a\n' > cmm.txt && git add cmm.txt && git commit -q -m "cm side a" \
  && git switch -q -c cm-b HEAD~1 && printf 'b\n' > cmm.txt && git add cmm.txt && git commit -q -m "cm side b" \
  && git switch -q cm-a && git merge -q cm-b 2>/dev/null; true)
if (cd "$repo/wt" && printf '<<<<<<< HEAD\na\n=======\nb\n>>>>>>> cm-b\n' > cmm.txt && git add cmm.txt && git commit -q -m "bad resolution" 2>/dev/null); then bad "marker resolution refused (merge)" "the incident path still commits markers through a merge"; else ok "marker resolution refused (merge)"; fi
if (cd "$repo/wt" && printf 'a\nb\n' > cmm.txt && git add cmm.txt && git commit -q -m "clean resolution" 2>/dev/null); then ok "clean resolution (merge) allowed" "the guard blocks the documented flow"; else bad "clean resolution (merge) allowed" "a correct merge resolution was refused"; fi
(cd "$repo/wt" && git switch -q feat/wt2 && git branch -q -D cm-a cm-b 2>/dev/null; rm -f cmm.txt)

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
git reset -q; git checkout -q -- CHANGELOG.md 2>/dev/null
# The preamble above the first "## [" heading is prose about the file, not
# an entry: correcting it is neither an assembly nor a release cut, and it
# is how the "add an entry to [Unreleased]" contradiction got fixed. The
# entries below the heading must survive untouched, or this is an edit.
released='## [Unreleased]\n\n## [0.2.0] - 2026-09-10\n\n### Added\n\n- a thing\n'
if (printf "# Changelog\n\nAssembled from fragments; entries live in docs/changelog.d/.\n\n$released" > CHANGELOG.md && git add CHANGELOG.md && git commit -q -m "changelog preamble" 2>/dev/null); then ok "CHANGELOG preamble edit allowed"; else bad "CHANGELOG preamble edit allowed" "the hook blocks a correction to the file's own prose"; fi
git reset -q; git checkout -q -- CHANGELOG.md 2>/dev/null
# ...and the exception must not become a door for entries: a change that
# also reaches past the first heading is still refused.
if (printf "# Changelog\n\nPreamble line.\n\n$released\n- typed past the heading\n" > CHANGELOG.md && git add CHANGELOG.md && git commit -q -m "entry via preamble" 2>/dev/null); then bad "entry below the heading still refused" "the preamble exception let an entry through"; else ok "entry below the heading still refused"; fi
git reset -q; git checkout -q -- CHANGELOG.md 2>/dev/null

# 2g. Handoff state reaches main by fast-forward — and, since ADR-0149, a
#     correction to what already landed reaches it directly, next to the
#     claim it corrects. A *new* note still belongs to its branch.
if (mkdir -p docs && printf '# Handoff — living project state\n\nrecorded the merge\n' > docs/handoff.md && git add -f docs/handoff.md && git commit -q -m "handoff on main" 2>/dev/null); then bad "board on main refused" "main took a generated board"; else ok "board on main refused"; fi
git reset -q; rm -rf docs
if (mkdir -p docs/handoff && printf 'note\n' > docs/handoff/2026-01-01-x.md && git add docs/handoff && git commit -q -m "note on main" 2>/dev/null); then bad "new session note on main refused" "main took a session note directly"; else ok "new session note on main refused"; fi
git reset -q; rm -rf docs
# A note that a branch already landed: seeded through the documented one-off
# escape hatch, because main may not create one.
(mkdir -p docs/handoff && printf 'note\n' > docs/handoff/2026-01-01-y.md && git add docs/handoff && PICODE_ALLOW_SWITCH=1 git commit -q -m "seed a landed note (one-off)")
if (printf 'note\ncorrected\n' > docs/handoff/2026-01-01-y.md && git add docs/handoff/2026-01-01-y.md && git commit -q -m "correct a landed note" 2>/dev/null); then ok "amending a landed note on main allowed"; else bad "amending a landed note on main allowed" "the post-merge correction has no door (ADR-0149)"; fi
if (mkdir -p docs/handoff/open && printf '# Topic\n\n## Debts\n\n- [x] paid in the field\n' > docs/handoff/open/topic.md && git add docs/handoff/open/topic.md && git commit -q -m "a durable item on main" 2>/dev/null); then ok "new topic file on main allowed"; else bad "new topic file on main allowed" "a fact learned after the merge has nowhere to go"; fi
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
