# 2026-09-22 — feat/cli-door-close

ADR-0184 slice 3. `POST /api/clis/{cli}/terminals` is gone (its 17 test
fixtures now create a launch agent: `launchFixture`). The one launch terminal
without an agent is a credential sign-in: `terminals.kind = 'signin'`
(migration 068), created only by `createSigninTerminal`, absent from
`/api/terminals`, peer owners and the app's lists (the feed reducer drops
`kind: signin`). The server closes it when the credential is imported, when
its session is gone, 15 min after creation, and at boot (`StartSigninReaper`,
1 min tick). The Providers card is its only door: **Open terminal** shows it in
a dialog on the card, **Cancel** deletes it, the strip reads `terminal.deleted`
("closed… sign in again"), and `GET /api/credentials/signin?cli=` restores the
strip after the pane was left.

Also fixed: a browser OAuth (Anthropic/Codex) had no timeout — an abandoned one
held 127.0.0.1:53692 and answered "already in progress" to every later login
until restart. Found because production held the port and failed the Omp
OAuth test; with the owner's OK, `POST /api/oauth/cancel` freed it. It now
gives up after 15 min.

visual-review: PASS (scratch clidoor3, desktop + 390px: strip, dialog with a
live terminal, Close/Cancel/closed-from-outside, lists empty during sign-in,
strip restored after a tab switch; mobile sheet width fixed).

## Next up

- `feat/cli-door-adopt`: Make agent from a PiCode shell running a catalog CLI (`tui.cli` already detected; bind endpoint must accept free shells).
