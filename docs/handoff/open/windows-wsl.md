# Windows and WSL

## Next

- [ ] Shell update notice (ADR-0216, 2026-09-24): the shell now announces `window.__PICODE_SHELL__ = {version, protocol}` and the UI has `shellSupports(n)`, but no "update PiCode Desktop" notice ships — `picode-desktop install` reuses the shell exe already on disk (`stageShellExe`), so there is no update path for the notice to point at. Build the path (install replaces an older shell, or the shell self-updates from the release), then show the notice when the page needs a protocol the shell lacks.
- [ ] Desktop shell waiting screen (2026-09-24, `feat/shell-waiting-screen`): host-tested stages and a headless render of every stage only — the live half (eval push, `navigate`, local-only `capabilities/waiting.json`, `certutil -user` prompt) has not run in the real shell. Owner, after `make desktop-restart`: (a) `wsl --shutdown`, open PiCode → "Starting Linux…" then PiCode loads on its own; (b) `systemctl --user stop picode` with the window hidden, reopen → "isn't answering" after ~25 s, Start PiCode → loads; (c) drag, minimize and close the waiting screen; (d) optional: remove the mkcert root from `CurrentUser\Root` and `LocalMachine\Root` → "doesn't trust" and Trust certificate.
- Desktop shell, daemon off port 8445 (2026-09-23, owner deferred): the runtime IPC grant (`desktop-shell/src/daemon_acl.rs`) is deployed (`make desktop-restart` 13:28) but untested live. Set the port to e.g. 8446 in Settings, open Management and check the scan runs and the main window's buttons answer, then set 8445 back. The port change restarts the daemon and ends turns in progress.
- Windows clean-machine install (ADR-0098): phase 1 stages in `picode-desktop.exe`, phase 2 `install.ps1` + winget, no paid signing (`docs/plans/windows-clean-install.md`).
- WSL control (P0/P1 shipped): P2 is the Storage app plus the Windows-facts route (own ADR); P3 — compact, `--set-sparse`, `--move` — costs the distro's sessions.

## Windows installer M1 — code landed, VM gate open (2026-09-19)

The branch that had been idle for four days (`feat/win-install-m1`) is
adopted and its WIP committed: the parent waits for the elevated child and
reports its exit code, the last error line lands in
`%ProgramData%\PiCode Desktop\install.log`, the elevated window pauses for
Enter on failure, `--user <name>` aims both the binary and pi at one account
(and `--user root` keeps the system-wide install), and the stage scripts no
longer carry a `$` that wsl.exe would eat. Plan: `docs/plans/windows-installer.md`.

**Open (owner, on the VM)**: M1's gate is the scenario run — clean-machine
path and adopted-Ubuntu `--user` path, idempotent re-runs, on `picode-test`
(Hyper-V, checkpoint `clean`), plus `startup-check`. M2 (`install.ps1` on the
site, the guide around the one-liner, the release-train pin bump) is the next
buildable milestone.

## Debts

- WSL disk: the tray warns in words only (no alert icon asset); docker's storage is the Docker app's, not measured here.
- Windows `Close` kills only its direct child; `internal/server` tests swap package-level probes, so `t.Parallel` would race (`scripts/go-test.sh` shards by process).
- [x] Desktop shell: `open_management_window` runs `discover_server()` (one hidden `wsl.exe` per distro, which can boot a stopped one) on the tray event thread; a slow WSL freezes the tray with nothing on screen. Move it off the event thread or reuse the board's known distro. Paid: the window opens from the health loop's known address; only an unknown one is looked up, off the event thread.
- [x] Cache table (`internal/hostfs/hostfs.go` Consumers): the pnpm row measures `~/.cache/pnpm` but `pnpm store prune` works on the store (`~/.local/share/pnpm/store` here), so a clean never shrinks the measured row; likewise a `go env -w GOCACHE=` elsewhere makes `go clean -cache` report 0 freed. Measure where the tool itself says (`pnpm store path`, `go env GOCACHE`). Paid (wsl-debts): each tool-owned cache is measured where its tool says (`hostfs.LocateConsumers`).
- [x] Desktop shell `.wslconfig` editor (`desktop-shell/src/lib.rs` edit, `wslconfig.rs:68`): a key is matched by name without its section — an edit can replace a same-named key in another owned section (the unused `fs` warning points at it). The Go wsl.conf editor matches section and key. Paid (wsl-debts): keys outside WSL sections are not read, and a key in both WSL sections ends as one line.
- [x] `disk-compact` stops the distro while the `PiCodeDistro` task restarts on failure within a minute; it now holds the distro (wsl-mgmt-history), but should also disable the task like move/backup do. Paid (wsl-debts): compact turns the task off and back on like move/backup.
