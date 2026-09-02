#!/usr/bin/env bash
# Swap the Windows tray + native-host exes and relaunch the tray through
# its logon task.
#
# The rule this script exists to enforce (2026-09-02 incident): NEVER
# launch picode-desktop.exe in the background from a WSL shell (`exe … &`).
# That process dies with the shell, the keepalive child dies with the tray,
# and the 60s WSL idle timeout reclaims the whole VM — the server, every
# tmux session and every managed agent go down with it. The logon task
# (`schtasks /run /tn PiCodeDesktop`) launches the tray detached from WSL,
# which is the only supported way to (re)start it from here.
set -euo pipefail

cd "$(dirname "$0")/.."

[[ -f bin/picode-desktop.exe && -f bin/picode-nmh.exe ]] || {
  echo "desktop-swap: bin/picode-desktop.exe / bin/picode-nmh.exe missing — run 'make desktop' first" >&2
  exit 1
}

win_user=$(/mnt/c/Windows/System32/cmd.exe /c "echo %USERNAME%" | tr -d '\r\n')
dest="/mnt/c/Users/${win_user}/AppData/Local/PiCode"
[[ -d "$dest" ]] || { echo "desktop-swap: $dest not found (is PiCode Desktop installed?)" >&2; exit 1; }

echo "Stopping the tray if it is running…"
/mnt/c/Windows/System32/taskkill.exe /IM picode-desktop.exe /F >/dev/null 2>&1 || true
sleep 1

echo "Copying bin/picode-desktop.exe and bin/picode-nmh.exe…"
cp -f bin/picode-desktop.exe bin/picode-nmh.exe "$dest/"

echo "Re-registering the Chrome native host…"
"$dest/picode-nmh.exe" extension-install

echo "Relaunching the tray via the logon task (detached from WSL)…"
/mnt/c/Windows/System32/schtasks.exe /run /tn PiCodeDesktop >/dev/null
sleep 3

if /mnt/c/Windows/System32/tasklist.exe /FI "IMAGENAME eq picode-desktop.exe" 2>/dev/null | grep -q picode-desktop.exe; then
  echo "Tray is running."
else
  echo "desktop-swap: the tray did not come up — check the Windows task 'PiCodeDesktop'" >&2
  exit 1
fi
echo "Done. Server health: curl -sk https://localhost:8445/api/health"
