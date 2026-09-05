# Desktop task reliability

Scope: keep ADR-0020's user-logon task and tray-owned WSL keepalive. Do not
introduce a Windows service, migrate startup to Run, reprovision WSL during
repair, or promise that a crash/restart preserves the WSL VM.

The installer registers the complete task policy in one Task Scheduler write:
interactive limited user, user-scoped logon, no duration/battery/idle/network
gates, IgnoreNew, three launch retries one minute apart. One minute is the
Windows minimum; recovery does not guarantee that WSL survives that interval.
Normal Quit must exit zero, including when initial distro resolution failed.

`startup-check` is read-only and works without resolving/starting WSL.
`startup-repair` backs up and updates only an existing owned task's policy;
it preserves its action, principal, triggers and enabled/disabled choice,
does not start/stop a process, and requests elevation only for access denied.
Missing or structurally different tasks require explicit installation.
Doctor shares the same inspection rather than equating existence with health.

Benchmark adaptation: the [t3code/paseo/Cursor study](../benchmarks/2026-08-24-adopt-t3code-paseo-cursor.md)
calls for explicit runtime states. Startup configured, disabled, stopped,
running, failed and uninspectable remain distinct; no new UI or agent runtime.

## Decision table

| Conditions | Action | Required evidence |
|---|---|---|
| Fresh installation | Register a complete enabled, limited, interactive logon task atomically | Policy and Windows round-trip tests |
| Existing owned task with legacy policy | Back up XML, repair policy, preserve action/principal/triggers/enabled | Repair/idempotence tests |
| Existing task disabled | Preserve disabled; report it separately from policy | Disabled-task tests |
| Missing task | Report missing; repair does not create it | Missing-task tests |
| Unknown owner, elevated/noninteractive principal or non-tray action | Refuse repair before mutation | Structural guard tests |
| Inspection fails or output is invalid | Report unknown/error, never missing or healthy | Parser and boundary tests |
| Repair denied by Windows | Request normal Windows elevation; no success claim for handoff alone | Permission boundary tests; real approval is owner-controlled |
| Task already running and launched again | Ignore the duplicate task invocation | Windows lifecycle test |
| Action fails to start | Request up to three retries, one minute apart | Windows lifecycle test |
| Already-started tray exits with an error | Report the last result; do not promise automatic crash recovery | Native exit-1 probe; supervisor/lifetime design deferred |
| User chooses Quit, even after initialization failed | Exit zero; no failure restart | Exit regression and Windows lifecycle tests |
| Server unavailable but tray alive | Keep current status polling; do not restart server or agents | Existing tray behavior preserved |
| Sign-in, battery transition, sleep/resume on the real workstation | Validate policy and retain runtime safety | Owner-controlled acceptance; no disruptive reboot/battery tests |

## Validation

Validated on 2026-09-05:

- `make ci` passed: formatting, vet, Go tests, 495 frontend tests, package
  suites, application/public-docs builds, docs parity and Vale.
- `go test -race ./internal/desktop ./cmd/picode-desktop` passed.
- Native Windows policy round-trip passed, including disabled-choice and
  exact XML backup preservation, idempotence, missing task and foreign action.
- Native command/exit tests passed on Windows as well as Linux.
- `TestWindowsTaskLaunchRetryAndQuit` passed in 141 seconds: a timed launch
  with a temporarily unavailable test executable retried after restoration;
  a duplicate task start was ignored; exit zero remained stopped beyond the
  retry interval; exit one was recorded as a runtime failure.
- An earlier manual exit-one probe observed no automatic restart within
  90 seconds. This is not certified crash recovery. Microsoft's
  [protocol definition](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/2ff4aa5a-7bc4-449f-bbb1-27475645867f)
  scopes retries to failure to start; runtime supervision is deferred.
- The new Windows guide was read at desktop/light and mobile/dark widths;
  its missing/disabled/error next actions remained readable. No app/tray
  layout or overlay changed. The app overlay auditor does not apply to
  VitePress. Three pre-existing stale public captures were regenerated with
  the normal pipeline on an isolated fixture/browser namespace and read.

Native lifecycle tests deliberately require `PICODE_TEST_TASK_LIFECYCLE=1`
in the **Windows** environment; ordinary CI skips this two-minute acceptance
probe. Build with `GOOS=windows GOARCH=amd64 go test -c ./internal/desktop`,
then run the test executable with `-test.run=^TestWindowsTask -test.v`.
All native probes use unique disposable tasks and helper executables, never
the live tray. Physical sign-in, battery and sleep/resume acceptance remains
owner-controlled. Inspect pane IDs/PIDs and health before and after any live
tray replacement; record installed policy separately from code validation.
