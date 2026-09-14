#!/usr/bin/env bash
# Swap the Windows tray + native-host + v2-shell exes and relaunch the tray
# through its logon task (the shell through a detached Windows start).
#
# The rule this script exists to enforce (2026-09-02 incident): NEVER
# launch picode-desktop.exe in the background from a WSL shell (`exe … &`).
# That process dies with the shell, the keepalive child dies with the tray,
# and the 60s WSL idle timeout reclaims the whole VM — the server, every
# tmux session and every managed agent go down with it. The logon task
# (`schtasks /run /tn PiCodeDesktop`) launches the tray detached from WSL,
# which is the only supported way to (re)start it from here.
#
# The v2 shell (ADR-0120) is swapped in the same pass: its Tauri capability
# list compiles into the exe, so a stale picode-shell.exe refuses commands
# the page legitimately sends (2026-09-14: `btab_cdp_call` answered
# "not allowed by ACL" for days after the CDP bridge landed). `make
# desktop-restart` builds both exes before calling this script; run alone,
# it swaps the shell only when a build exists and warns otherwise.
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

echo "Stopping the tray if it is running…"
stop_exe picode-desktop.exe
if [[ $have_shell == true && $shell_running == true ]]; then
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

echo "Relaunching the tray via the logon task (detached from WSL)…"
if [[ -n "$dry" ]]; then
  echo "  [dry-run] schtasks /run /tn PiCodeDesktop"
else
  /mnt/c/Windows/System32/schtasks.exe /run /tn PiCodeDesktop >/dev/null
fi
sleep 3

if /mnt/c/Windows/System32/tasklist.exe /FI "IMAGENAME eq picode-desktop.exe" 2>/dev/null | grep -q picode-desktop.exe; then
  echo "Tray is running."
else
  echo "desktop-swap: the tray did not come up — check the Windows task 'PiCodeDesktop'" >&2
  exit 1
fi

if [[ $have_shell == true && $shell_running == true ]]; then
  # A plain detached Windows start: the shell carries no keepalive duty (the
  # tray owns the VM lifetime), so cmd start cannot strand the VM the way a
  # WSL-backgrounded exe would.
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
