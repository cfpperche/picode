#!/usr/bin/env bash
# Swap the Windows tool + native-host + shell exes and relaunch the resident
# through its logon task (ADR-0142: the task's target is the resident — the
# Go tray before the migration, the shell after).
#
# The rule this script exists to enforce (2026-09-02 incident): NEVER
# launch a resident exe in the background from a WSL shell (`exe … &`).
# That process dies with the shell, the keepalive child dies with it, and
# the 60s WSL idle timeout reclaims the whole VM — the server, every tmux
# session and every managed agent go down with it. The logon task
# (`schtasks /run /tn PiCodeDesktop`) launches the resident detached from
# WSL, which is the only supported way to (re)start it from here.
#
# The shell's Tauri capability list compiles into the exe, so a stale
# picode-shell.exe refuses commands the page legitimately sends (2026-09-14:
# `btab_cdp_call` answered "not allowed by ACL" for days after the CDP
# bridge landed). `make desktop-restart` builds every exe before calling
# this script; run alone, it swaps the shell only when a build exists and
# warns otherwise. A pre-migration shell that was running comes back through
# a detached Windows start — it carries no keepalive duty there, so cmd
# start cannot strand the VM the way a WSL-backgrounded exe would.
#
# DRY_RUN=1 prints the plan and touches nothing (the read-only probes run).
set -euo pipefail

cd "$(dirname "$0")/.."

dry=${DRY_RUN:-}
run() { if [[ -n "$dry" ]]; then echo "  [dry-run] $*"; else "$@"; fi; }
stop_exe() {
  if [[ -n "$dry" ]]; then echo "  [dry-run] taskkill /IM $1 /F"; else
    /mnt/c/Windows/System32/taskkill.exe /IM "$1" /F >/dev/null 2>&1 || true
  fi
}

[[ -f bin/picode-desktop.exe && -f bin/picode-nmh.exe ]] || {
  echo "desktop-swap: bin/picode-desktop.exe / bin/picode-nmh.exe missing — run 'make desktop' first" >&2
  exit 1
}
shell_src="desktop-shell/target/x86_64-pc-windows-msvc/release/picode-shell.exe"
have_shell=false
[[ -f $shell_src ]] && have_shell=true
if [[ $have_shell == false ]]; then
  echo "desktop-swap: no shell build at $shell_src — the installed picode-shell.exe stays as is (run 'make desktop-shell')" >&2
fi

win_user=$(/mnt/c/Windows/System32/cmd.exe /c "echo %USERNAME%" | tr -d '\r\n')
dest="/mnt/c/Users/${win_user}/AppData/Local/PiCode"
[[ -d "$dest" ]] || { echo "desktop-swap: $dest not found (is PiCode Desktop installed?)" >&2; exit 1; }
win_dest="C:\\Users\\${win_user}\\AppData\\Local\\PiCode"

shell_running=false
if /mnt/c/Windows/System32/tasklist.exe /FI "IMAGENAME eq picode-shell.exe" 2>/dev/null | grep -q picode-shell.exe; then
  shell_running=true
fi

# Who the task starts is who comes back: the tray before the migration, the
# shell after. A task that cannot be read is a pre-migration machine.
resident="tray"
if task_xml=$(/mnt/c/Windows/System32/schtasks.exe /query /tn PiCodeDesktop /xml 2>/dev/null); then
  if grep -q "picode-shell.exe" <<<"$task_xml"; then
    resident="shell"
  fi
else
  echo "desktop-swap: cannot read the PiCodeDesktop task — assuming the tray is the resident" >&2
fi
echo "Resident: $resident (the PiCodeDesktop task's target)"

# The distro keepalive (ADR-0154): the scheduled task holds the distro open
# independent of the resident, so the taskkill below cannot strand it into
# WSL's idle reclaim — the failure that ends every tmux session, agent and
# terminal at once. Ensure it BEFORE any kill; on failure warn and continue
# (that is the pre-fix behavior, not a worse one).
echo "Ensuring the distro keepalive task (PiCodeDistro)…"
if [[ -n "$dry" ]]; then
  echo "  [dry-run] powershell: Register-ScheduledTask PiCodeDistro + Start-ScheduledTask"
else
  /mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe -NoProfile -Command - <<'PS' || echo "desktop-swap: the keepalive task could not be ensured — continuing without it" >&2
$ErrorActionPreference = 'Stop'
try {
  $action = New-ScheduledTaskAction -Execute 'wsl.exe' -Argument '--exec /bin/sleep infinity'
  $trigger = New-ScheduledTaskTrigger -AtLogOn -User $env:USERNAME
  $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -ExecutionTimeLimit ([TimeSpan]::Zero) -MultipleInstances IgnoreNew -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1) -StartWhenAvailable
  Register-ScheduledTask -TaskName 'PiCodeDistro' -Action $action -Trigger $trigger -Settings $settings -Force | Out-Null
  Start-ScheduledTask -TaskName 'PiCodeDistro'
} catch { Write-Error $_; exit 1 }
PS
fi

echo "Stopping the tray if it is running…"
stop_exe picode-desktop.exe
if [[ $shell_running == true ]]; then
  echo "Stopping the shell if it is running…"
  stop_exe picode-shell.exe
fi
sleep 1

echo "Copying bin/picode-desktop.exe and bin/picode-nmh.exe…"
run cp -f bin/picode-desktop.exe bin/picode-nmh.exe "$dest/"
if [[ $have_shell == true ]]; then
  echo "Copying the v2 shell…"
  run cp -f "$shell_src" "$dest/"
fi

echo "Re-registering the Chrome native host…"
run "$dest/picode-nmh.exe" extension-install

echo "Relaunching the resident via the logon task (detached from WSL)…"
if [[ -n "$dry" ]]; then
  echo "  [dry-run] schtasks /run /tn PiCodeDesktop"
else
  /mnt/c/Windows/System32/schtasks.exe /run /tn PiCodeDesktop >/dev/null
fi
sleep 3

if [[ $resident == "shell" ]]; then
  want="picode-shell.exe"; came_up="Shell is running."
  missing="desktop-swap: the shell did not come up — check the Windows task 'PiCodeDesktop'"
else
  want="picode-desktop.exe"; came_up="Tray is running."
  missing="desktop-swap: the tray did not come up — check the Windows task 'PiCodeDesktop'"
fi
if /mnt/c/Windows/System32/tasklist.exe /FI "IMAGENAME eq $want" 2>/dev/null | grep -q "$want"; then
  echo "$came_up"
else
  echo "$missing" >&2
  exit 1
fi

if [[ $shell_running == true && $resident != "shell" ]]; then
  # Pre-migration coexistence: the task brought the tray back; the shell
  # that was running comes back through a detached start.
  echo "Relaunching the shell…"
  (cd /mnt/c/Windows/Temp && run /mnt/c/Windows/System32/cmd.exe /c start "" "${win_dest}\\picode-shell.exe")
  sleep 3
  if /mnt/c/Windows/System32/tasklist.exe /FI "IMAGENAME eq picode-shell.exe" 2>/dev/null | grep -q picode-shell.exe; then
    echo "Shell is running."
  else
    echo "desktop-swap: the shell did not come back — start it from the tray" >&2
    exit 1
  fi
fi
echo "Done. Server health: curl -sk https://localhost:8445/api/health"
