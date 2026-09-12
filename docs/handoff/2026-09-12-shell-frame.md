# 2026-09-12 — feat/shell-frame: undecorated windows (owner request)

Shipped (fast-forward ready): the native title bar sat above the app's own
header — two bars for one window. Both shell windows are undecorated now;
the app renders its own controls gated on `window.__PICODE_SHELL__`
(initialization script; browsers never set it). ShellFrame pins a
min/max/close cluster top-right and tracks maximized state via onResized;
the sidebar brand row and the Management header are drag regions
(target-hit only — buttons inside keep working); management.html carries
the same cluster. First Tauri capabilities file grants the core window
permissions.

Verified: make web builds; cargo xwin build; ci-scoped green (vale flaked
once on a cold cache — same shape as the go-test cache flake, rerun green).
Installed live + owner-visual pending (screenshot before merge was the
decorated build).

Debts: no snap-layout flyout on hover (Win10 has none anyway); dragging by
the exact "PiCode" text/button does not drag (target-hit semantics);
Maximize glyph depends on onResized events (edge: restore-by-snap may show
stale glyph until next event).

Merge: fast-forward ready.
