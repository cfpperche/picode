# 2026-09-25 — feat/legacy-sweep-2: second legacy sweep
Owner: "faz tudo". Commits: 576f14360, f9781e2ad, 0fbbb8086, 3f1d74482, 287a8b9d2, 6de4e8cb0, 8f8ed12f0, 212b17e53.
QA scripts: six moved off the links retired earlier today (qa-cli-settings, -recovery, qa-cli-packages,
qa-cli-providers, qa-llama, qa-mobile-settings) — missed by the earlier branches. qa-cli-providers'
"unsupported" row used Codex (supported since ADR-0166); it now uses an unknown CLI id.
Removed: cmd/uicheck (stale, unreferenced); picode-dash-scope localStorage cleanup. Go tray migration
(ADR-0142 amended): --tray parse + runRetiredTray, startup-repair --retarget-shell, RetargetTask/
retargetTask/CanRetarget, TrayArgs, task.ps1 'retarget' op and '--tray' (repair accepts only --hidden);
desktop-swap.sh refuses a task that does not start picode-shell.exe (owner's PiCodeDesktop task measured:
picode-shell.exe --hidden). recoverPiReceiverRuntime /proc environ fallback (a hello without runtimePid
recovers nothing). Rust shell: disk_report --stream retry fallback; btab_cdp_call domains now a required
Vec<String>; preview.rs comment fixed. store.CreateTerminal (39 test sites use CreateTerminalIn with
FreeWorkspaceID). Three stale comments fixed (pkgs/legacy.go, tmux/server.go, automationsPi.js).
Verified: `make ci-scoped` PASS (full scope); all six scripts pass `node --check`; Go tests pass on Linux
and compile for GOOS=windows; make desktop-test passes. Live QA on picode-docs-fixture (:18861):
qa-cli-providers fails at its first ready() (waits for `#providers-view .prov-bar`, removed 2026-09-21 by
7aed3a1cc, ADR-0169 — broken before today); qa-cli-settings failed a keyboard density assertion at ~158,
unrelated to links, so its edited sections never ran; the other four were not run. Blind spots: Rust not
compiled here (cargo in WSL unsafe); task.ps1 not executed (Windows only).
visual-review: n/a (no UI-visible change). Merge: fast-forward ready.

## Debts

- Stale QA scripts and the owner's first shell/startup-repair run: docs/handoff/open/legacy-compat.md
