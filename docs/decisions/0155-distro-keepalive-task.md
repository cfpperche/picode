# ADR-0155: The distro keepalive is a scheduled task, not the shell's child

- **Status**: accepted (amends ADR-0142's keepalive-lifetime clause: the
  keepalive now deliberately outlives the shell, and the job object guards
  only the fallback child)
- **Date**: 2026-09-18
- **Boundary**: process — who owns the WSL distro's lifetime on Windows and
  what a shell death (crash, upgrade, taskkill) may take down with it.

## Context

Every terminal, agent and tmux session lives inside the WSL distro, and WSL
reclaims a distro with no Windows-side client after its idle timeout
(~60s). ADR-0142 answered that with a keepalive child of the resident:
`wsl.exe -d <distro> -- /bin/sleep infinity`, tied to the shell by a job
object so it "cannot outlive the shell however the shell dies".

That guarantee is the bug. On 2026-09-18, four `wsl --terminate`-grade
instance terminations (journal: `InitTerminateInstanceInternal … calling
reboot(RB_POWER_OFF)`) killed every session at once. The 18:05:05 one is
fully forensically reconstructed: `make desktop-restart` swapped the shell
exes at 18:04:32, the task's taskkill ended the old shell — and its
keepalive with it — and 31 seconds after the new shell started, WSL
reclaimed the distro. Every deploy that day was clean ("all N session(s)
still alive", ~15 times); the kills all trace to the resident's death, not
the daemon restart. `scripts/desktop-swap.sh` itself documents the shape
(the 2026-09-02 incident) — but treated it as a rule for *callers*, while
the coupling that makes the rule necessary stayed in the design.

## Decision

The distro's Windows-side owner is a **scheduled task, `PiCodeDistro`**,
whose action is `wsl.exe [-d <distro>] --exec /bin/sleep infinity` —
logon trigger, no execution-time limit, IgnoreNew, restart-on-failure
(three tries, one minute apart). Its lifetime is independent of any
process: the shell crash it was built to survive is now survived. The
shell's duty shrinks to *ensuring* the task (register when the action
differs, `schtasks /run` otherwise — idempotent), with ADR-0142's job-
object child kept as the fallback for machines where the task machinery
fails. `scripts/desktop-swap.sh` ensures the task **before** any taskkill,
so the swap window never leaves the distro unowned. Deliberate release is
unchanged: the Give back / compact flows terminate the instance, the sleep
dies with it (the task returns to Ready), and the next ensure re-arms.

The registration payload goes through PowerShell cmdlets
(`Register-ScheduledTask`), not the `schtasks /create` CLI: ONLOGON needs
admin there, and `/sd` is locale-brittle. Distro names are validated
(alphanumeric, `-_.`, space) before they enter the payload; anything else
degrades to WSL's default distro.

## Consequences

- **Fixes the class, not the instance**: shell crash, Windows update,
  future scripts — no path strands the distro anymore, because nothing
  Windows-side dies with the shell.
- **Inverts ADR-0142's guarantee on purpose**: a stray keepalive now
  outlives the shell. That is the point; the release valve is the existing
  Give back flow, which ends the sleep by ending the instance.
- **Cost**: the shell spawns `schtasks /run` once per health poll (~5s)
  — a hidden ~30ms process, the same order the shell already pays for
  health and disk reads.
- **Untested rows, named**: the Rust `TaskEnsure` register/run paths do
  real Windows IO and are compile-checked (`cargo xwin check`), not unit-
  tested here; the live machine carries the task since 2026-09-18
  (registered, Running) and that deployment is the test.
- **Open suspect, not closed**: the two terminations that do not coincide
  with a desktop-restart (16:44, 17:06) may be WSL's own `sparseVhd=true`
  reclaim (`.wslconfig`). A keepalive cannot cure that; enabling WSL's
  Operational log is the next measurement (debt in
  `docs/handoff/open/wsl-keepalive.md`).

## Alternatives considered

| Alternative | Why not |
|---|---|
| Overlap in `desktop-swap.sh` (start the new resident before the kill) | Patches one caller; a shell crash or any other taskkill still strands the distro |
| `vmIdleTimeout` raised in `.wslconfig` | The marra: works for every cause but defeats idle RAM reclamation machine-wide and stays one config edit from silently reverting |
| A Windows service holding the distro | Stronger lifecycle than a task, but needs admin install and a wrapper exe for what a logon task already does |
| Keepalive as a systemd unit inside the distro | Internal processes do not count as WSL clients — the reclaim ignores them, which is why the keepalive must be a Windows-side client at all |
