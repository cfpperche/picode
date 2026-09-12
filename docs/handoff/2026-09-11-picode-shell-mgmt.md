# 2026-09-11 — picode-shell-mgmt: the WSL Disk window (Phase 2 session A)

Shipped (fast-forward ready, installed live): the management window inside
the shell. Tray item WSL disk... opens ui/disk.html (local page, global
Tauri invoke) rendering the two-sided report from picode-desktop.exe disk
--json -- Windows (free/allocated/held/sparse) and the distro
(used/free/safe/consumers). Give back wired to disk-compact --yes --json
for session B; readiness refusals render in the window. Tool lookup:
sibling exe first, then %LOCALAPPDATA%\PiCode (desktop-swap keeps it
fresh).

Go: disk-compact --json (additive) -- one outcome object, always exit
zero; refused/note/error travel as data. Human text output unchanged.
Verified live from the installed exe: dry-run JSON with held ~92 GB.

Owner-corrected scope recorded in docs/plans/desktop-v2.md: WSL control is
shell-only; the daemon gets no routes/app (no ADR-0121 needed); keepalive
stays in the Go tray -- porting it was rejected in review.

Incident: this is the rebuild of feat/desktop-v2-mgmt(2), both destroyed by
branch-name collisions with other sessions (focus-split; captures refresh).
Reported to the owner. Rebuilt from session contents; branch name now
unique (picode-shell-mgmt); committed and ff-merged immediately to shrink
the collision window.

Verified: Go tests, cargo xwin build, make ci-scoped PASS (full); shell +
Go tray running side by side.

visual-review: the WSL Disk window is owner-visual (tray -> WSL disk...).

Debts: no measurement cache (~10 s per open); the remote UI can invoke the
disk commands (our own UI only); toast Start Menu shortcut still pending.

Merge: fast-forward ready.
