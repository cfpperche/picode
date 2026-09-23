# 2026-09-23 — feat/grok-login: Grok signs in through the GUI

Shipped: ADR-0192 (owner direction). Grok's Add opens `GrokLoginDialog.jsx` (browser and mobile) with these options:
- **Sign in with Grok** (browser), with an animated wait and a link to reopen the page;
- **Device code**, with a copy button;
- **xAI API key**;
- "Sign in from a terminal" as the fallback.

`internal/server/grok_login.go` runs Grok's own `grok login --oauth|--device-auth` (measured on 1.0.41: both run without a terminal) and parses the page and code it prints. It serves `POST`/`GET`/`DELETE /api/grok/login`, one sign-in at a time. A session is imported into the vault after exit 0.

Fixed: a Grok key saved in PiCode never reached Grok. Saving and using a key now:
1. refuses while Grok terminals run;
2. files Grok's session in the vault, under Grok's own email;
3. runs `grok logout` and verifies it (502 if the session stays);
4. only then copies `auth.json` aside to `credfiles/grok-session.json`;
5. records the key, so launches get `XAI_API_KEY`.

Use on the session writes it back from that copy, which also makes it Activatable while the live file is gone. The "a session outranks this key" note hides while the key is in use, and the Use confirm text now matches what a key does. The key-in-use setting is generic per CLI (`keyInUse`/`setKeyInUse`); Claude Code's setting keeps its name.

Verified: `make ci-scoped` PASS. `TestGrokGUILogin` runs a fake `grok` (the test binary re-execs itself) and covers device and browser flows, 409, a refusal, a sticky logout refused with no copy kept, key Use (session filed, labelled, Activatable), env precedence, and Use back to the session. Scratch QA with the real Grok and an isolated HOME: the first pass was FAIL (no Use on the filed session, stale note, confirm text, unchecked logout, generic label). All were fixed, and the recheck PASSED with a realistic fake session that Grok's own logout recognises. A later side effect (a refused switch overwrote the good copy) was fixed server-side and covered by the test.
visual-review: PASS (gkl-2-waiting.png, gkl2-1-menu.png, gkl2-2-session-used.png, gkl2-3-keyconfirm.png, gkl2-4-sticky.png; card 5/5)

Not verified: a real Grok account or key end to end.

## Next up

- Next CLI onto its own GUI sign-in: Hermes, Muse, Antigravity; OpenCode last (owner's order)

## Debts

- `scripts/qa-scratch.sh` copies the owner's `~/.pi/agent/auth.json` (a live xAI session included) into every scratch: a Grok key switch there can sign that copy out and Use can write it into the scratch Grok home (ADR-0192)
- Grok GUI sign-in and key switch not yet exercised live with a real account (ADR-0192)

Merge: fast-forward ready.
