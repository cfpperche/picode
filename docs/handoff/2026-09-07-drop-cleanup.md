# 2026-09-07 — feat/drop-cleanup: the CLI attach door stops touching the project's .gitignore

Owner spotted `.picode/drop/` in a repo's file tree and asked about
centralizing it in PiCode's global data dir. Refused: the door pastes a
path — the CLI reads it, confined to its own cwd — so the folder stays in
the project on purpose (ADR-0089 amended with the reasoning). Chasing the
question found two real bugs in `touchDropGitignore`: no root
`.gitignore` left attachments fully untracked forever; an existing one
got a silent, uncommitted `.picode/drop/` line appended, dirtying `git
status` on every attach (this repo's own root `.gitignore` carried
exactly that line all session). Fix: a nested, uncommitted
`.picode/.gitignore` (`drop/`), written once, never opening the
project's file. Second bug: nothing ever deleted a staged attachment (11
files, 3 MB, in this repo's own `.picode/drop/`); `sweepDropDir` removes
anything older than 7 days on the next drop into the same project — no
daemon.

Verified: `make ci-scoped` PASS (4 new/changed Go tests). Live on the
docs fixture against a real `claude-code` terminal: no-gitignore project
got the nested file with no root write; existing `.gitignore` left
byte-for-byte unchanged; a 30-day-old staged file was swept on the next
attach, a fresh sibling survived.

Not done: this repo's own leftover drop files (<24h old) were left in
place — the fix is opportunistic, not a one-time purge. The root
checkout's obsolete `.gitignore` diff was reverted by hand.

Merge: `git merge --ff-only feat/drop-cleanup && make ci` from main.
