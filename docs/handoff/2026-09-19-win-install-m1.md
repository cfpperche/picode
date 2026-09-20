# 2026-09-19 — the stalled Windows install branch, adopted (feat/win-install-m1)

The only stuck work on the board: 1 commit ahead, 18 dirty files, no commit
for four days, gates green 4 days ago, and **no session note** — a session
stopped mid-flight. The owner asked what was next; this was the answer.

## What I found and did
- The WIP was not debris: it closes M1's remaining items, and it compiles and
  passes its own tests. Two facts were VM-run findings written down properly —
  a failure that hid behind `exit 0` because the launcher returned before the
  elevated child finished, and shell scripts whose `$` never survived the
  `wsl.exe` argument boundary (`for g in sudo wheel`, `conf=/etc/wsl.conf`).
- Committed as two commits (the M1 close, then the plan document that the
  branch had been executing but never committed), merged `main` (417 commits
  behind, no conflicts), `make ci-scoped` green.
- M1's own gate — the VM scenarios on `picode-test` — is the owner's, and is
  recorded in the plan and in `docs/handoff/open/windows-wsl.md`.

## Next up
- M2: `install.ps1` published on the site, the Windows guide rewritten around
  the one-liner, and the release-train pin bump (engineer decision 2 in the
  plan: pin per release).

## Debts
- The VM gate is unrun: the code is landed and host-tested, but no clean
  machine has executed it since the WIP was written.
