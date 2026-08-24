#!/usr/bin/env bash
# Install PiCode as a systemd user service (with weekly cert renewal).
set -euo pipefail
cd "$(dirname "$0")"

mkdir -p ~/.config/systemd/user
cp systemd/picode.service systemd/picode-cert.service systemd/picode-cert.timer ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now picode.timer 2>/dev/null || true
systemctl --user enable picode

echo "installed:"
echo "  systemctl --user status picode"
echo "  journalctl --user -u picode -f"
