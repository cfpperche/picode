# 2026-09-22 — feat/omp-add-dialog: omp gets pi's Add provider flow

Shipped: the owner asked for every CLI to get pi's Add provider flow, starting with omp. The omp roster now carries `add.kind: "provider"` and a `signin` per row (`browser` when `oauth.Supports`, `terminal` when the CLI has its own login). `AddProviderDialog.jsx` (browser and mobile) takes `cli`, `roster` and `onTerminalSignin`, and for a guest it offers only what that CLI has natively:
- a key, where the row names `env.api_key` (`POST /api/credentials`);
- an account, via `POST /api/credentials/signin`: the browser flow is polled the same way as pi's, and the terminal flow goes to the pane's sign-in strip;
- Custom provider, where `custom.available` is set.
The labels are PiCode's vocabulary, otherwise Omp's own name, sorted alphabetically. Omp's " · Sign in" name tag is dropped. Pi's own flow is unchanged. The other guests keep the old key form.

Verified: `make ci-scoped` PASS. The new server test covers `add.kind`, `signin` browser/key-only/terminal, and OpenCode staying on the key form. A scratch instance with the real omp: 82-item picker; key, method, browser and terminal steps; the Custom page; Pi and OpenCode unchanged; 390px sheet; overlay audit ok on every step. Nothing was clicked through to a real vendor sign-in.
visual-review: PASS (ompdlg-1-picker.png … ompdlg-12-mobile-terminal.png; card 5/5)

Not done: the rest of the guests (OpenCode, Hermes, Codex, Claude Code, Grok, Muse, Antigravity) are still on the key form. Each needs its native list and doors measured first. The pinned Custom provider row stays highlighted after a search, so Enter opens it (same in pi's dialog). Some catalog rows show a letter icon instead of the vendor's.

## Next up

- Move the next guest CLI onto pi's Add flow: measure its native provider list and sign-in, then set `add.kind` and `signin` (omp is the reference, `docs/architecture/cli-providers.md`)

Merge: fast-forward ready.
