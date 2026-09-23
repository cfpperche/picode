# 2026-09-23 — feat/hermes-login: Hermes credentials through the GUI

Shipped: ADR-0193 (owner: "(a)", a new credential goes last in Hermes's pool).
- **Roster.** Hermes's roster appends its own `PROVIDER_REGISTRY`, which gives 45 providers in total. It is read through the Python behind the `hermes` on PATH (`hermesInterpreter`, which follows the installer wrapper's exec line), cached per source, with `PICODE_HERMES_CATALOG` for tests. Declared ids, aliases, `aws_sdk`/`vertex`/`external_process` and Qwen OAuth are left out.
- **Dialog.** The Add dialog is pi's picker. The key door runs `hermes auth add <id> --type api-key` with the key on **stdin**. The account door runs `--type oauth --no-browser`, a device code, only where Hermes runs one (`HermesDeviceSignin`: not Anthropic or Copilot). The page and code are shown, there is an animated wait, and Back or close cancels on the server. It uses `POST`/`GET`/`DELETE /api/hermes/credential`, one sign-in at a time.
- **Vault.** What Hermes stored is filed in the vault.
- **Ids.** Vault ids map to Hermes ids (`HermesAddID`).

Fixed: `parseHermes` now reads a pooled key from `access_token`, which is where Hermes 0.21 keeps it (measured). Before, PiCode never detected pooled keys.

Verified: `make ci-scoped` PASS. New tests: the catalog parser, `For`/`EnvVar` with the catalog, `HermesAddID`, the pool key field, and the wrapper → interpreter lookup. `TestHermesGUICredentials` runs a fake `hermes` that fails if the key ever reaches argv, and covers roster, device-only account doors, key and pool order, refusal, OAuth page, code and filing, 409, and the refusals.

Scratch QA with the real Hermes. The first run was PARTIAL: the isolated HOME hid the catalog, which exposed that the reader depended on HOME. After the fix to the PATH lookup, the rerun PASSED: 45 providers, DeepSeek keys appended p0 and p1 in the scratch pool (the real `~/.hermes` untouched), and the Nous code flow with its cancel.
visual-review: PASS (hml2-1-picker.png, hml2-2-table2.png, hml2-3-nous-code.png, hml2-4-mobile-picker.png; card 5/5)

Not verified: a real Hermes key or sign-in end to end. "Use first" (moving a credential to the front of the pool) is not built.

## Next up

- Next CLI onto its own GUI sign-in: Muse, Antigravity; OpenCode last (owner's order)

## Debts

- Hermes GUI credentials not yet exercised live with a real key or account (ADR-0193)

Merge: fast-forward ready.
