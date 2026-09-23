---
description: Start PiCode Desktop at Windows sign-in, inspect its task and repair startup without changing WSL.
---

# PiCode Desktop on Windows

PiCode Desktop starts at Windows sign-in and places an icon in the notification
area. The PiCode server runs inside WSL. The shell holds WSL open while it
runs, reports whether the server answers, and opens PiCode in its own window.

Closing the window only hides it — the tray icon keeps the service running.
Reopen the shell and work resumes where it was. **Quit**, in the tray menu,
is what actually ends the resident.

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

A last result is historical information, not proof of why the shell stopped.

## Repair an existing task

Upgrade PiCode Desktop first. Older builds could incorrectly
report an error after a normal Quit.

```powershell
.\picode-desktop.exe startup-repair
.\picode-desktop.exe startup-check
```

Repair backs up the task definition under
`%LOCALAPPDATA%\PiCode\task-backups` and updates only its runtime policy.
It preserves the registered program, account, sign-in triggers and whether
startup is enabled. It does not restart the shell, server, agents or terminals;
it does not set up WSL again or change certificate trust.

Windows may ask for administrator approval for an existing task. The shell
itself still runs without administrator privileges. Approving the prompt is
not the final result: run `startup-check` again after repair finishes.

## What the startup policy does

- Starts at sign-in under your Windows account.
- Has no time limit and no battery, idle or network conditions.
- Ignores a second task launch while that task is already running.
- Requests up to three retries for failures to launch, one minute apart.
- Does not retry a normal **Quit**.

Task settings alone do not upgrade an already running executable. After an
upgrade, restart the shell to load the new program. Repository developers use
`make desktop-restart`; do not launch the shell in the background from WSL.

## Availability limits

The Task Scheduler retry setting is not a general crash supervisor: a process
that starts and later exits with an error may remain stopped. Use the task's
last result for diagnosis, then explicitly reopen the shell when appropriate.

The shell still owns the process that keeps WSL active. If the shell ends, WSL
may shut down when idle. Even a successful launch retry takes at least one
minute; existing agents and terminals are not guaranteed to survive that gap.
The Windows task does not turn PiCode into a machine-wide service independent
of sign-in. See Microsoft's [restart semantics](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/2ff4aa5a-7bc4-449f-bbb1-27475645867f).

**Quit** closes the shell without explicitly stopping the PiCode service, but
also releases the process that holds WSL open. Closing only the window leaves
the resident and agents running. See [getting started](/guide/getting-started)
for installation inside Linux or WSL.

## See what the disk is doing

The tray menu's **Management** item → **Disk** tab shows the same facts.
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

Measuring starts the distro if it was stopped — the shell keeps it up anyway,
but a laptop where you quit the shell will see WSL boot under the command.

**Held for nothing** is the number Windows cannot show you anywhere: space the
distro has already freed that the disk file still occupies. WSL gives it back
on its own only while the file is **sparse**, and it can only compact or convert
the file while the distro is stopped — which is why a PiCode that keeps WSL up
also keeps that space.

### Give the space back

When there is something to give back, the Management window's **Disk** tab
offers **Give back held space**. Choosing it

1. asks PiCode whether anyone is working — the same check `picode deploy` uses;
   if someone is mid-turn it names them and stops;
2. asks once, in a dialog that names the cost: stopping Ubuntu ends every
   agent, terminal and tmux session inside it, and they do not come back;
3. stops Ubuntu, converts the disk file to **sparse**, starts Ubuntu again;
4. reports before and after, measured on the file — not promised.

Afterwards the file is sparse and WSL returns freed blocks on its own. This is
the one-time fix, not a chore to repeat.

From a terminal the same flow is `disk-compact`:

```powershell
.\picode-desktop.exe disk-compact --dry-run   # the plan; nothing stops
.\picode-desktop.exe disk-compact --yes       # stop Ubuntu, compact, restart
```

`--force` overrides the working check; `--method optimize-vhd` compacts with
Hyper-V's Optimize-VHD instead and needs an administrator terminal (Windows
Home has no Hyper-V module — upgrade WSL for the sparse path). When the compact
finishes, Ubuntu starts again by itself; the sessions do not.

Inside the distro, `picode disk` lists every item on its own:

```bash
picode disk          # what occupies this machine, and what is safe to reclaim
picode disk --json   # the same measurement as JSON (picode-desktop disk reads it)
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

## The Management window

The tray menu's **Management** item opens a window with three tabs over the
WSL distro. Opening it starts a scan: one row for Windows and one for the
distro, each with a spinner and a clock, and each card fills as soon as its
half is read — Windows in about a second, the distro after it walks the home
directory. No terminal window opens; everything runs in the background.
**Scan again** re-measures and keeps the last numbers on screen until the
new ones land.

**Disk** — what the distro holds: free space, the size of the disk file,
and how much of it is *held for nothing* (freed inside the distro but not
given back to Windows). **Give back held space** stops the distro, converts
the disk file to sparse (or runs `optimize-vhd`), and starts it again; every
step streams into the window. Stopping the distro ends everything inside it
— agents, terminals, tmux — and the window says so before it acts. A
readiness interlock refuses the run while someone is mid-turn, unless you
force it.

**Clean** — the caches the scan measured, with sizes: build caches,
package caches, downloaded engines and models. Select and prune; nothing
stops. Caches marked `redownload` come back from the network the next time
something needs them. Agent CLI sessions and the PiCode database never appear
here — their cleanup is not a delete. This runs `picode clean`; the
subcommand refuses any id that is not a cache, even when asked for by exact
name.

**Config** — the WSL settings file (`.wslconfig`): memory, processors,
swap, and the sparse-disk flag. Saving backs the file up to
`.wslconfig.bak` first and leaves unknown settings untouched. Changes
apply at the next full WSL restart; after a save the tab offers **Restart
WSL to apply**, which asks first and states the cost.

**System** — memory and the WSL version. Memory shows the limit the WSL
virtual machine runs with, how much of it Linux is using (and how much of
that is cache it can drop), what Windows holds for it right now, swap, and
Windows' own free RAM; a swap more than half used gets one line saying what
to change. The WSL card shows the installed version and kernel and checks
Microsoft's release page for a newer one; when there is one it offers
**Update WSL**. **Restart WSL** and **Update WSL** stop all of WSL: every
distro and every session inside it ends. Both first ask PiCode whether
anyone is working and refuse while someone is mid-turn, then ask you once.
