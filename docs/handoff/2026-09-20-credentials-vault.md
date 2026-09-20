# 2026-09-20 — feat/credentials-vault: one encrypted vault for every agent CLI

Shipped: `internal/credentials` — `<DataDir>/credentials.json` as an AES-256-GCM envelope with `credentials.key` beside it (both 0600, atomic replace), provider-keyed rows (label, vendor identity, paused, masked hint, health with age, origin); `ErrLocked`/`ErrCorrupt` are never overwritten, and ADR-0013's `accounts.json` is absorbed once and left in place. `internal/catalog` now delegates to it (pi's `auth.json` stays its own slot, its only writer). `internal/clicreds` declares the nine CLIs and reads each vendor's own login (pi, claude-code, codex, grok, hermes, opencode, muse, agy; omp has none — its store is a live SQLite PiCode will not read). `internal/server/credentials.go` exposes `GET /api/credentials?cli=`, add key, import, rename, pause, delete, verify (one listing call per provider, explicit click, outcome cached on the row). The pane ships for the eight guest CLIs in both apps (`CliCredentials.jsx`, shared stylesheet, shared pure helpers) with import/adopt rows and empty/blocked/locked/error states. Backup carries the vault in a secrets snapshot and never its key. Docs: ADR-0165, `docs/architecture/credentials.md`, the providers guide, a changelog fragment, and the plan updated with what shipped and the four deliberate deviations.
Verified: `make ci-scoped` PASS (fmt, vet, hooks, go[10 packages], test-js, build, docs); the vault package's tests cover the envelope, a flipped byte, a missing key, migration-once, pause/remove promotion, token merge and import-vs-activate. Visual review on a scratch instance passed, screenshots in `var/screenshots/` (`cred-populated`, `cred-row-menu`, `cred-add-dialog`, `cred-empty-codex`, `cred-locked`, `cred-mobile`, `cred-mobile-sheet`): `window.__picodeOverlayAudit()` returned ok:true in every state, rows aligned at 36px, and the desktop dialog and mobile bottom sheet both sat fully inside the viewport. Blind spot: the review covers those captured scratch-instance states only.
visual-review: 1 overlay inside the screenshot? yes. 2 every item readable? yes. 3 trigger still usable? yes. 4 clip/double-scroll/dead hover? no. 5 terminal-averse next click obvious? yes.
Merge: fast-forward ready.

## Next up

- Step 2 (launch injection, per-account dirs, guided vendor sign-in, harvest) is not started and needs its own ADR — `docs/plans/agent-cli-credentials.md`.

## Debts

- Credential mutations publish no change-feed event, and Verify answers API-key rows only (a subscription row's answer is Usage today, step 2 later) — `docs/architecture/credentials.md`.
- Import-only rows (omp; the subscription rows of Muse/Antigravity/Hermes/OpenCode) and the two `unverified:` parser facts carry their reason on the row — `internal/clicreds/specs.go`, `internal/clicreds/parse.go`.
