# ADR-0197: Antigravity signs in through PiCode's GUI, with a pasted Google code

- **Status**: accepted (owner, 2026-09-23: "sigo sua recomendação" — option (a))
- **Date**: 2026-09-23
- **Boundary**: process. PiCode runs `agy -p . --print-timeout 1s` under a pseudo terminal (`script -qfec`), passes a code on its stdin, and kills its process group. Persistence: a login already in Antigravity's token file is filed in the vault and set aside during the sign-in, then removed on success or put back on failure. Protocol: `POST`/`GET`/`DELETE /api/agy/login`, `POST /api/agy/login/code`, `add.kind: "agy"`.

## Context

Measured on Antigravity 1.2.9:
- There is no login command, and `GEMINI_API_KEY` in the environment is ignored.
- With no login, print mode (`agy -p`) prints Google's authorization page. That page's callback, `antigravity.google/oauth-callback`, shows a code, and Antigravity asks for it on stdin, but **only with a terminal on stdin** (a pipe gets "Run 'agy' to log in"). It allows **60 seconds**.
- Print mode then goes on to run its prompt. `agy models` does not offer the sign-in.

The owner chose (a) over keeping Antigravity terminal-only: a GUI sign-in whose cost is a race against print mode's prompt.

## Decision

- **Pseudo terminal.** PiCode runs `agy -p . --print-timeout 1s` under `script -qfec` in its own process group. It shows the page (and opens it), passes the pasted code in, and **kills the group the moment the token file holds a login**, before the prompt runs. The prompt is the smallest one, with a one-second cap.
- **A login already present** would skip the sign-in and run the prompt. So it is filed in the vault and set aside, then removed on success or put back on failure or cancel.
- **The dialog** opens on its one door: Google's page, a field for the code, a 60-second countdown, and "Get a new link" when the window closes. There is no terminal door, since Antigravity has no sign-in command.
- **Refusals:** the sign-in is refused while Antigravity terminals run. Without `script` it says so.

## Consequences

Antigravity signs in without a terminal. The costs are named rather than hidden:
- the minute-long window;
- the dependency on `script` (util-linux, present on Linux and WSL);
- in the worst case, one tiny request if the kill loses the race (in the test, agy is stopped long before its prompt would finish);
- a login without an `id_token` collapses into the vault's single unnamed row.

If Antigravity's print-mode text changes, the dialog shows its last line.

## Alternatives considered

- **(b) Keep Antigravity terminal-only.** Rejected by the owner.
- **Drive Google OAuth with Antigravity's client id from PiCode's own engine.** Rejected: that reverse-engineers a vendor's OAuth client; the pseudo terminal uses Antigravity's own flow.
