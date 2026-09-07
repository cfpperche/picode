# 2026-09-07 — mobile-files-git: Files/editor and Git

Shipped: owner-approved mobile scope extension (ADR-0095), independent
full-screen Files and Git from Work, agent tools and terminal actions.
Files has browse/search, editor/previews, explicit dirty navigation and
mtime/root recovery; Git has ancestry, branch filters, commit/worktree
details, PRs and existing Prepare/Run/Ask channels. Workspace history and
commit reads now work without an agent; all owners check supplied roots.
Navigation serials prevent stale folder reuse across duplicate browser history.

Verified: Go server tests, 1,215 JS tests, production builds, docs and Vale.
Files: 8 browser groups / 22 captures; Git: 56 captures / 124 guarded reads;
integration: 7 journeys / 22 captures, including actual terminal preparation
without Enter and a short viewport with the commit input focused.
visual-review: PASS — screenshots read; every overlay audit ok; card
contained/readable/usable/no-clip/clear-next-action = yes/yes/yes/yes/yes.
Evidence: `var/screenshots/mobile-files-git/`; repeatable scripts in `scripts/`.
Public captures and OpenAPI regenerated; decisions in `docs/plans/mobile-v2.md`.

Debts: physical iOS/Android IME/PWA/push/resume acceptance remains open.
File writes retain the existing symlink and non-atomic mtime limitations.
Merge: fast-forward ready; deployment remains the normal owner/timer batch.
