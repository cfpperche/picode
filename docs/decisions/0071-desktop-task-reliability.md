# ADR-0071: Resident Windows task policy and explicit startup repair

- **Status**: accepted (extends ADR-0020; startup mechanism unchanged)
- **Date**: 2026-09-05

## Context

The owner's tray stopped while the existing `PiCodeDesktop` task remained
registered. Its Windows defaults included a 72-hour execution limit and battery
restrictions. Those settings are unsuitable for a resident application, but the
earlier exit cause is unconfirmed. Doctor checked only task existence.

The tray also owns a WSL keepalive. It must run as the interactive user, and
the existing restart flow launches it through Windows rather than attaching it
to a WSL shell. Recovery must distinguish process failure from intentional Quit.
The command dispatcher previously let a successful tray return fall through to
the unknown-command exit, turning a normal Quit into a nonzero result.

The [t3code/paseo/Cursor benchmark](../benchmarks/2026-08-24-adopt-t3code-paseo-cursor.md)
motivates explicit runtime states: configured startup is not a running process,
and an inspection error is not a missing task.

## Decision

Keep ADR-0020's user-logon task and current tray/WSL boundary. The installer
registers an enabled task for the installing Windows user's interactive token,
at limited privilege, with an explicit resident policy. Registration uses the
built-in Task Scheduler COM API through embedded PowerShell and one definition
write, followed by verification. No Go dependency or Windows service is added.

| Property | Value |
|---|---|
| ExecutionTimeLimit | `PT0S` (unlimited) |
| Battery, idle and network gates | off |
| MultipleInstances | IgnoreNew |
| RestartOnFailure | 3 launch retries, 1 minute apart; not a general process supervisor |
| Normal Quit | successful exit; no failure retry |

`startup-check` and doctor inspect actual settings, identity, executable,
runtime state and last task result. The check never resolves or starts WSL.
`startup-repair` updates only an existing owned task's policy after backing up
its XML. Action, principal, triggers and the enabled/disabled choice survive.
No-op repair does not write. Unexpected identity/shape, a missing executable,
or a trigger's own time limit is refused before mutation. Only native access
denied permits a standard UAC handoff; launching the elevated copy is not proof
that repair completed. The original tray executable must be upgraded first.

## Consequences

- Installation no longer inherits process-limiting scheduler defaults.
  Native tests validate registration, repair, preservation, missing/foreign
  actions, duplicate launches, launch retry and successful exit.
- Microsoft's [protocol definition](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/2ff4aa5a-7bc4-449f-bbb1-27475645867f)
  scopes restart retries to unmet start conditions and failure to start an
  action. Native acceptance observed no restart within 90 seconds after an
  already-started helper exited 1. Do not promise arbitrary crash recovery;
  a separate supervisor/lifetime proposal needs owner approval.
- A limited-user task may still require elevation to change an existing
  administrator-created definition. Repair does not weaken its ACL or change
  its account to avoid that boundary.
- [Windows requires at least one minute between retries](https://learn.microsoft.com/en-us/windows/win32/taskschd/tasksettings-restartinterval).
  The WSL VM may stop during that gap: recovery of the tray is not a promise
  that agents or terminals survive a crash. The tray's keepalive coupling is
  unchanged; a separate runtime-lifetime design remains future work.
- If the policy is wrong, repair retains the previous XML in the user's
  PiCode task-backup directory. Repair does not restart processes; rollback
  remains an explicit owner operation. Physical battery/sleep/logon tests on
  a live workstation require owner coordination.

## Alternatives considered

| Alternative | Trade-off |
|---|---|
| Only change this workstation | Future installs keep the same defaults and diagnostics remain incomplete. |
| Startup through `HKCU\Run` | Valid simpler autostart, but requires a replacement for the existing detached restart flow; does not decouple keepalive from the tray. |
| Windows service plus separate tray | Possible future topology; requires identity, IPC and runtime-lifetime design beyond the approved repair. |
| Unlimited immediate failure retries | Repeated failures become a restart loop; bounded retries leave an inspectable outcome. |
| Re-run full installation to repair policy | Unnecessarily reaches WSL provisioning and certificate trust. |

See the [decision table and acceptance plan](../plans/desktop-task-reliability.md).
