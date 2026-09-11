---
description: Start PiCode Desktop at Windows sign-in, inspect its task and repair startup without changing WSL.
---

# PiCode Desktop on Windows

PiCode Desktop starts at Windows sign-in and places an icon in the notification
area. The PiCode server runs inside WSL. The tray opens PiCode in your browser,
reports whether the server answers, and holds WSL open while it runs.

## Check startup

Use a Windows terminal in the folder containing PiCode Desktop:

```powershell
.\picode-desktop.exe startup-check
```

This reports the registered task, executable, startup policy, current task
state and last result. It changes nothing and does not start WSL. `doctor`
includes the same task checks alongside the broader installation checks.

| Result | Next action |
|---|---|
| Running, policy correct | No action needed. |
| Stopped, policy correct | You may have selected Quit. To reopen, run `schtasks /run /tn PiCodeDesktop`. |
| Disabled | Enable `PiCodeDesktop` in Windows Task Scheduler if you want startup at sign-in. |
| Old duration, battery or launch retry settings | Upgrade PiCode Desktop, then repair the task below. |
| Task missing | Run `picode-desktop install` to register startup as part of installation. |
| Different account, command, or missing executable | Inspect the registration before reinstalling. Repair will not replace it silently. |
| Access or inspection error | Check task permissions. An unreadable task is not reported as missing. |

A last result is historical information, not proof of why the tray stopped.

## Repair an existing task

Upgrade the installed tray executable first. Older builds could incorrectly
report an error after a normal Quit.

```powershell
.\picode-desktop.exe startup-repair
.\picode-desktop.exe startup-check
```

Repair backs up the task definition under
`%LOCALAPPDATA%\PiCode\task-backups` and updates only its runtime policy.
It preserves the registered program, account, sign-in triggers and whether
startup is enabled. It does not restart the tray, server, agents or terminals;
it does not set up WSL again or change certificate trust.

Windows may ask for administrator approval for an existing task. The tray
itself still runs without administrator privileges. Approving the prompt is
not the final result: run `startup-check` again after repair finishes.

## What the startup policy does

- Starts at sign-in under your Windows account.
- Has no time limit and no battery, idle or network conditions.
- Ignores a second task launch while that task is already running.
- Requests up to three retries for failures to launch, one minute apart.
- Does not retry a normal **Quit** in the updated tray.

Task settings alone do not upgrade an already running executable. After an
upgrade, restart the tray to load the new program. Repository developers use
`make desktop-restart`; do not launch the tray in the background from WSL.

## Availability limits

The Task Scheduler retry setting is not a general crash supervisor: a process
that starts and later exits with an error may remain stopped. Use the task's
last result for diagnosis, then explicitly reopen the tray when appropriate.

The tray still owns the process that keeps WSL active. If the tray ends, WSL
may shut down when idle. Even a successful launch retry takes at least one
minute; existing agents and terminals are not guaranteed to survive that gap.
The Windows task does not turn PiCode into a machine-wide service independent
of sign-in. See Microsoft's [restart semantics](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/2ff4aa5a-7bc4-449f-bbb1-27475645867f).

**Quit** closes the tray without explicitly stopping the PiCode service, but
also releases the process that holds WSL open. Closing only the browser leaves the tray and
agents running. See [getting started](/guide/getting-started) for installation
inside Linux or WSL.

## See what the disk is doing

The tray keeps one line about the disk, refreshed every five minutes:

```
WSL 218 GB · ≈92 GB held by Windows · C: 27 GB free
```

Below 20 GB free the line ends in `— low`, and the tooltip adds the command
below. Hovering the icon is the whole check — no Explorer, no `df` by hand.

`picode-desktop disk` prints the two sides in full, and changes nothing:

```powershell
.\picode-desktop.exe disk
```

```
Windows
  C:                       27 GB free of 476 GB
  its disk file            218 GB on disk · not sparse
                           C:\Users\you\AppData\Local\wsl\{…}\ext4.vhdx
  held for nothing         ≈92 GB the distro has already freed
                           Windows keeps every block it ever wrote while the file is not sparse.
                           WSL 2.7.0.0 can convert it: wsl --manage Ubuntu --set-sparse true

Inside Ubuntu
  /                        126 GB used of 1007 GB
  safe to reclaim          41 GB · 30 GB of it costs only the time to rebuild
  biggest                  Go build cache 30 GB · npm cache 5.8 GB · Go module cache 2.1 GB
```

**Held for nothing** is the number Windows cannot show you anywhere: space the
distro has already freed that the disk file still occupies. WSL gives it back
on its own only while the file is **sparse**, and it can only compact or convert
the file while the distro is stopped — which is why a PiCode that keeps WSL up
also keeps that space. `picode-desktop disk` names the fix; it does not run it,
because stopping the distro costs the sessions in it. Reclaiming with a review
step is designed in `docs/plans/wsl-control.md`.

Inside the distro, `picode disk` lists every item on its own:

```bash
picode disk          # what occupies this machine, and what is safe to reclaim
picode disk --json   # the same measurement, for the tray and the app
```

Each item carries what giving it back costs:

| Label | What it means |
|---|---|
| `safe` | Only time. A build cache rebuilds itself on the next build. |
| `redownload` | Comes back from the network on the next install or test run. |
| `data` | Yours — sessions, PiCode's own history. Never a one-line command. |

The report names what it could not read instead of hiding it: paths owned by
another account (the Docker engine's own storage is the usual one) appear as
"not accounted for", not inside a category they do not belong to.
