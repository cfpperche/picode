# 2026-09-23 — feat/muse-login: Muse signs in through the GUI

Shipped: ADR-0195. Muse's Add opens `MuseLoginDialog.jsx` (browser and mobile) with these options:
- **Meta account:** PiCode runs `muse login` (measured on 1.3.0: a device code with no terminal needed) through the new generic `deviceLogin` runner (`internal/server/device_login.go`, `POST`/`GET`/`DELETE /api/muse/login`). The dialog shows the code, a Copy button, the link on its own line, and an animated Waiting. On exit 0 the login is filed in the vault, named by Muse's `user_email`.
- **Meta API key:** "Save and use" saves it to the vault and makes it live with Use.
- **Sign in from a terminal**, as the fallback.

Fixed: Use on Muse overwrote Muse's single credential slot. Now it first files the login that slot holds (`fileCLILogin`), so a key never destroys an account login made in Muse's own terminal.

Verified: `make ci-scoped` PASS. `TestMuseGUILogin` (fake `muse`) covers the code flow, 409, filing, Use key → Use account round trip and the add kind. `TestMuseUseFilesTheLoginItReplaces` covers a terminal-made login surviving a key and being labelled by its email. Scratch QA with the real Muse and an isolated HOME PASSED: the code flow and cancel, geometry constant, seeded account → key → Use back, mobile, and Grok/Hermes unchanged. The label fix came after the QA and is covered by the test.
visual-review: PASS (msl-2-code.png, msl-3-after-key.png, msl-3-after-use.png, msl-4-m-code.png; card 5/5)

Not verified: a real Meta account or key end to end. UX notes from QA: the pane keeps its own "Sign in" button beside Add provider (two doors to the same sign-in), and the disabled Waiting button is low-contrast.

## Next up

- Antigravity onto its own GUI sign-in; OpenCode last (owner's order); Grok and Hermes could move onto the generic `deviceLogin` runner

## Debts

- Muse GUI sign-in not yet exercised live with a real Meta account or key (ADR-0195)

Merge: fast-forward ready.
