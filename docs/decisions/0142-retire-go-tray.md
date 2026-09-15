# ADR-0142: The shell is the only Windows resident — the Go tray retires

- **Status**: accepted (owner approved 2026-09-15; reverses the 2026-09-11 coexistence correction in `docs/plans/desktop-v2.md`; amends ADR-0120's "supervisor" wording back and ADR-0071's task target)
- **Date**: 2026-09-15
- **Boundary**: process — which Windows process owns logon autostart, the WSL keepalive and the tray, and what the close/quit contract is

## Context

Two PiCode residents run on the owner's Windows machine: the Go systray
(`picode-desktop.exe --tray`, ADR-0020, via the `PiCodeDesktop` logon task)
and the Tauri shell (`picode-shell.exe`, ADR-0120). Both show a Pi tray icon;
only the Go tray owns the keepalive that holds the WSL VM up, so the shell
cannot keep the service alive on its own and the tray cannot show a window.

On 2026-09-11 the owner rejected porting the keepalive to Rust and retiring
the Go tray — "the two coexist". Four days of duplicated duty (two icons,
two tooltips, two menus) reversed that call: the old tray dies and the
desktop shell becomes the single resident. ADR-0120 anticipated exactly
this migration ("the keepalive/watchdog must migrate before the Go tray
retires, or the two coexist with duplicated duty").

The systemd user unit inside WSL is not in question: it is the mechanism
both residents share. This decision is about the Windows resident only.

## Decision

`picode-shell.exe` becomes the only Windows resident: it owns logon
autostart (the retargeted `PiCodeDesktop` task, ADR-0071 policy unchanged),
the WSL keepalive, health/disk polling, the single tray icon, and the
window. Closing the window hides it — the tray keeps the service running;
reopening the shell resumes work. Only the tray menu's Quit exits the
process. The Go systray loop retires: `cmd/picode-desktop` keeps its
headless CLIs (`install doctor disk disk-compact clean startup-check
startup-repair update`), which the shell drives as subprocess tools (the
established `run_cli` pattern), and loses its `--tray` path, keepalive
supervision and systray dependency, so a second tray cannot regress.

## Consequences

- **Easier**: one icon, one tooltip, one menu; window close/quit semantics a
  desktop user already understands; the Management window and the tray act
  through the same shell instead of two processes.
- **Harder**: the keepalive (child + Job-Object supervision + post-compact
  re-arm) is reimplemented in Rust; `startup-check`/`startup-repair` must
  accept the shell exe as the task target; `update` and `release.yml` cover
  two exes; the handover on the live machine must never drop the keepalive
  (runbook in `docs/plans/retire-go-tray.md`).
- **Cost accepted**: a second implementation of supervision duty, and a live
  migration on a machine with running agents/terminals (owner-coordinated,
  no `wsl --shutdown`).
- **If we're wrong**: the Go tray code is deleted, so rollback is the
  previous release's `picode-desktop.exe` plus the backed-up task XML —
  both kept by the runbook, not by the repo.

## Alternatives considered

| Alternative | Why not |
|---|---|
| Keep the Go tray, drop the shell | Trashes the 2026-09-11/15 flagship direction (work browser, CDP bridge, Management); the shell already owns the window and the browser policy |
| Keep both residents permanently | The duplicated duty this ADR ends: two icons, two menus, and a shell that cannot hold the VM it renders |
| Shell supervises a headless Go `keepalive` helper | The heartbeat would depend on a second exe the shell must also supervise — one more failure hop for the duty that matters most |
| Tauri autostart plugin instead of the scheduled task | A second autostart mechanism; the task keeps the ADR-0071 resident policy, the UAC story and the existing inspect/repair surface |
| Merge the Go CLIs into the shell binary | The Go tools are tested, release-published and drive the provision JSON contract; a port buys nothing |
