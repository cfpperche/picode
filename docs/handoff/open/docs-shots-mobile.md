# Docs harness: docs-shots mobile-inbox regression

## Debts

- `make docs-shots` fails deterministically at `app-mobile-inbox` since main's 2026-09-13 browser/workspace-picker batch: the mobile list stays on skeleton with store data present (badges render), while the same URL renders fine in a warm session — and it reproduces with **main's own web tree** built and shot from this worktree (differential run, 2026-09-13). Every session touching web inputs is blocked from the docs gate until fixed or the harness's reload/viewport flow is made race-free. Desktop surfaces pass; the failure is mobile-after-desktop in one session (`scripts/docs-shots.mjs`, fresh `shot-*` session each run).
- `pi auth check` reports `invalid_state` for a provider whose `models.json` fails schema validation — the error surfaces as an unrelated auth verdict (seen while debugging custom providers; worth an upstream report).
