# 2026-09-20 — creds-signin: guided sign-in per CLI + row identity (ADR-0168)

Shipped: the Providers pane starts a CLI's own login (`POST
/api/credentials/signin`, `clicreds.Spec.Login`, hint strip + Check now in both
apps); vault rows are named by the store or by the person (`Login.Identity`,
`Store.Adopt`), the vendor's profile endpoint fills display only, the roster
matches a live file offline (key, else the saved token); Muse reads/writes both
file shapes (`expires_at` = unix seconds, refresh token optional); the
env-shaped api_key fingerprint is a constant. ADR-0168, plan updated.

Verified: `make close` green twice (after main's merge too) — go test full
server/clicreds/credentials/usage suites, 54 JS tests, `make web`, openapi
regenerated, docs-check ok. New server tests pin: signin argv reaches the
launch, two codex logins stay two rows, a named login re-keys instead of
copying, roster in-use matches a profile-named row by token. Scratch QA
(qa-scratch, desktop + 414×896 mobile) drove Sign in → Check now → alert;
screenshots read by a subagent, overlay audit ok:true, rows 36px aligned.
Blind spot: no real vendor OAuth was completed end-to-end in QA — the flows
verified are the CLI's own login running in the terminal and the file/import
path around it; hover/scroll affordances not provable from statics.

visual-review: PASS (strip/notyet/mobile re-read after fixes; card 5/5; first
round found orphaned Dismiss, unstyled alert, and a mislabeled mobile capture —
all fixed and re-verified in pixels)
Not done / debts: harvest (read a renewed token back) and launch env injection
remain open in docs/plans/agent-cli-credentials.md; Claude/Muse/pi keep one
subscription row until named (the pane says so and offers the name).
Merge: fast-forward ready.

## Next up

- Env-var injection at launch (the vault's step 2 remainder) — declarations
  carry the names; wire them to the launcher without touching HOME.
- Harvest: import a CLI-renewed token back into its vault row (today the copy
  goes stale and re-import is manual).
