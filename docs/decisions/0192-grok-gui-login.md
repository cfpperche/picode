# ADR-0192: Grok signs in through PiCode's GUI

- **Status**: accepted (owner direction 2026-09-22/23: GUI sign-in per CLI, "pode seguir")
- **Date**: 2026-09-23
- **Boundary**: process. PiCode runs `grok login --oauth|--device-auth` and `grok logout` on the person's behalf. Persistence: the setting `credentials.grok.key` (the key in use), and a copy of Grok's last `auth.json` at `<data>/credfiles/grok-session.json`. Protocol: `POST`/`GET`/`DELETE /api/grok/login`, `add.kind: "grok"`.

## Context

Grok 1.0.41's own sign-in is `grok login` (browser OAuth through auth.x.ai) or `grok login --device-auth`, plus the `XAI_API_KEY` variable. Measured in a scratch `GROK_HOME`:
- both login modes run without a terminal: they print the page to open (and, for the device flow, the code), wait, and write Grok's `auth.json`;
- `XAI_API_KEY` is read (`auth_type=ApiKey` in its debug log);
- a signed-in session outranks the key (the spec's existing note).

Before this ADR, PiCode offered the terminal login plus Import, and a key saved for Grok **never reached it**: nothing injected the variable, and Use writes only sessions.

## Decision

The Grok dialog offers three doors: the browser, a device code, and an xAI API key. The terminal stays as a fallback.

| Case | Action |
|---|---|
| browser / device | PiCode runs `grok login`, shows its page (and code), and waits for exit 0 (10-minute window, one at a time); the session is then imported into the vault; a chosen key is cleared |
| Use on a key row ("Save and use") | refused while Grok terminals run; otherwise Grok's session is filed in the vault, its `auth.json` copied aside, `grok logout` run, and the key recorded; new launches get `XAI_API_KEY` |
| Use on a session row | written into `auth.json`, from the kept copy when the file is gone (its `<issuer>::<client id>` key is not in the vault); the key is cleared |
| the person's env sets `XAI_API_KEY` | never overwritten |

## Consequences

Grok signs in without a terminal. A Grok key saved in PiCode now works, and switching back to the session is one click. The dialog depends on the text `grok login` prints; if that changes, the dialog shows Grok's last line and the terminal fallback still works. Writing a different account's session back into the kept copy leaves that copy's email and principal fields describing the older account until Grok refreshes them.

## Alternatives considered

- **Inject the key without signing out.** Rejected: the session outranks it, so the key would silently do nothing.
- **Write `auth.json` from the vault alone.** Rejected: the entry key (`<issuer>::<client id>`) is Grok's, and guessing it is the reverse engineering the kept copy avoids.
