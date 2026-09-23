# ADR-0195: Muse signs in through PiCode's GUI

- **Status**: accepted (owner direction 2026-09-23: GUI sign-in per CLI, "pode seguir")
- **Date**: 2026-09-23
- **Boundary**: process. PiCode runs `muse login` on the person's behalf. Persistence: Use on Muse now files the login Muse holds in the vault before replacing it. Protocol: `POST`/`GET`/`DELETE /api/muse/login`, `add.kind: "muse"`.

## Context

Measured on Muse 1.3.0:
- `muse login` is Meta's device-code flow. It runs without a terminal: it prints the page and the code, waits, and writes `~/.config/muse/auth.json`.
- `muse auth set --api-key-stdin` stores a key in the same file.
- Muse keeps **one** credential (logout removes "the API key or the Meta-account login"), and `META_API_KEY` outranks both.
- PiCode already read and wrote both kinds (ADR-0166), but signing in needed a terminal. Use on a key would overwrite a login made in Muse's own terminal that PiCode never imported.

## Decision

The Muse dialog offers two doors, with the terminal as the fallback:
- **Meta account:** PiCode runs `muse login` (the generic `deviceLogin` runner), shows its page and code, and files the login in the vault on exit 0.
- **Meta API key:** saved to the vault and made live with Use.

Use on Muse files the login its one slot holds before writing over it, so no credential PiCode did not keep is lost. Use on the account row brings it back.

## Consequences

Muse signs in without a terminal, and a key never silently destroys an account login. The dialog depends on the text `muse login` prints; if that changes, the dialog shows Muse's last line and the terminal fallback still works. `META_API_KEY` in the person's environment still outranks whatever PiCode writes; that is Muse's rule.

## Alternatives considered

- **`muse auth set --api-key-stdin` for the key.** Equivalent for the file, but Use already writes both kinds faithfully, and one writer keeps Use and Add consistent.
