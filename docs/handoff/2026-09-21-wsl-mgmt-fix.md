# 2026-09-21 — wsl-mgmt-fix

Management window ACL fixed; the tray's disk items are gone.

## What changed

- **Management window was dead on arrival.** Its 7 commands sat on the ACL
  guard's exception list under a wrong assumption ("never invoked from a
  webview"), so every tab answered `not allowed by ACL`. They now live in
  `capabilities/management.json`; the exception list is down to
  `computerlab_open`; the guard test enforces the management commands.
- **`clipboard_files` gap.** Registered and granted but missing from
  `build.rs` — the guard test's other check caught it.
- **Tray simplified.** The WSL disk line and **Give back ≈N GB…** are gone;
  the facts and the give-back flow live in the Management window's Disk tab.
  Removed with them: the disk poller, the Rust compact flow, `diskline.rs`.

## Verification

- `make ci-scoped` green (fmt, vet, hooks, desktop-test, xwin, docs).
- `cargo xwin check`: no new warnings; ACL-guard replication passes (57 cmds).

## Next up

- Visual check on Windows (owner): Management tabs render without ACL errors;
  tray shows no disk line / Give back. Needs `make desktop-restart` — the
  Linux host cannot run the shell.
