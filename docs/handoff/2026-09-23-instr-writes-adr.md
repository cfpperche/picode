# 2026-09-23 — feat/instr-writes-adr: ADR-0204 and the first instruction-file fixes

The owner accepted ADR-0204 the same day: the Instructions tab may propose small, exact fixes and write them only on confirmation. The fixes never run git, never overwrite a personal file, stay inside the workspace, and a write is refused when any file changed since the diff was shown (compared by content hash).

**Built.** `internal/cliinstructions/fix.go` covers two fixes:
- `bridge`: adds `@AGENTS.md` to a `CLAUDE.md` that only points to it in words.
- `personal`: creates `CLAUDE.local.md` or `AGENTS.override.md` and adds it to the `.gitignore` in the same folder.

Both are covered by a decision-table test (`fix_test.go`): drift, never overwriting, links refused, a folder leading out refused. `GET/POST /api/workspaces/{id}/instructions/fix`: 409 carries the fresh change, 404 means the fix no longer applies. In the UI, **Review change** on the findings and **Add personal file** open a diff dialog; there is a new finding for a personal file not in `.gitignore`.

**Caught in QA.** The first write succeeded on disk, but the dialog stayed open and the list kept the old finding: the code called `toast.success`, which does not exist here (`toast.ok`), and the exception was swallowed. Now the reload runs before the toast.

Verified: `make ci-scoped` and `make close`; the full flow on scratch `instrfix` (bridge, personal from the finding, personal from the menu, each checked with `git status` in the fixture: unstaged, nothing committed). Visual review in a subagent. Nothing deployed.
