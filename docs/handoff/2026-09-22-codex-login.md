# 2026-09-22 — feat/codex-login: Codex signs in through the GUI

Shipped: ADR-0191 (owner direction: GUI sign-in per CLI, "pode seguir pro codex"). Codex's Add opens `CodexLoginDialog.jsx` (browser and mobile), which mirrors Codex 0.156.0's `/login`:
- ChatGPT in the browser: it opens the `authUrl`, the wait is animated, and a link reopens the page if the tab was lost;
- a device code, with a Copy button, for use from a phone or another machine;
- an API key;
- Amazon Bedrock, by Bedrock API key or by access keys;
- "Sign in from a terminal" as the fallback.

The server drives `codex app-server`'s `account/login/*` over stdio (`internal/server/codex_login.go`; `POST`/`GET`/`DELETE /api/codex/login`, `DELETE /api/codex/platform`), with one sign-in at a time (409 otherwise). Codex writes its own `auth.json` and `config.toml`. A ChatGPT or key login is imported into the vault. A login that is not Bedrock, including Use, removes the `model_provider = "amazon-bedrock"` that Codex leaves behind, using the format-preserving TOML doc and the new `clisettings.Doc.Scalar`. Bedrock shows as "Codex uses Amazon Bedrock · region" with Edit and Stop using (Codex's own logout). The platform line and Stop using are now shared by Claude Code and Codex.

Verified: `make ci-scoped` PASS. `TestCodexGUILogin` runs a fake app-server: the test binary re-execs itself as `codex` and writes the files the way the real Codex was measured to. It covers API key, Bedrock plus roster, refusal, ChatGPT completion with the `model_provider` fix, device code, 409, Stop using and the input refusals. `TestGuestRosterCarriesThePaneFieldsOnly` now expects each CLI's own dialog kind. The protocol was measured against the real Codex 0.156.0 in a scratch `CODEX_HOME` with fake values. Scratch QA ran the real Codex with an isolated HOME. The first pass was PASS. The recheck of the additions was FAIL (footer jitter, and Copy flush against the link); both were fixed and a second recheck PASSED with constant geometry.
visual-review: PASS (cxl-1-method.png, cxl-5-bedrock-saved.png, cxl3-device-code.png, cxl3-mobile-device.png; card 5/5)

Not verified: a real ChatGPT, API-key or Bedrock sign-in end to end. The browser door needs the browser to reach `localhost:1455` on the PiCode machine; the device code covers the rest. Nits: the ellipsis reveal steps unevenly (a 1.25em box around one glyph); Copy code is 36px on mobile.

## Next up

- Next CLI onto its own GUI sign-in dialog: Grok, Hermes, Muse, Antigravity; OpenCode last (owner's order)

## Debts

- Codex GUI sign-in (ChatGPT, device code, API key, Bedrock) not yet exercised live with real accounts (ADR-0191)

Merge: fast-forward ready.
