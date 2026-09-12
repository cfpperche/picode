# 2026-09-11 — picode-shell-brand: sharp icon + product identity

Shipped (fast-forward ready, installed live): the taskbar icon was blurry
and inverted — the ico ladder started at 16px and the first image is the
one both the resource group and Tauri's runtime decode use. The ladder is
now 256-first (desktop-shell/tools/mkicon.go). The WSL Disk window used a
hand-rolled dark theme; it now carries the canonical tokens from
web/shared/tokens/theme.css (light, ADR-0072) copied with a sync note.

Verified: cargo xwin build; exe installed to %LOCALAPPDATA%\PicodeShell
and relaunched (PID confirmed); owner validates the taskbar sharpness and
the light identity.

Debts: tokens are copied by hand until a build step ships theme.css with
the shell; toast Start Menu shortcut still pending (Phase 2 installer);
no measurement cache (~10 s per open).

Merge: fast-forward ready.
