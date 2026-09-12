# 2026-09-12 — feat/shell-app-bar: the shell owns the app bar (ADR-0122)

The owner rejected the frame-mode result against the ChatGPT desktop
reference. Every shell window is now undecorated with two webviews
(Tauri unstable multiwebview): a 40px local app bar — brand, drag region,
double-click maximize, flat Windows caption buttons (close-red hover,
restore/maximize glyph via onResized) — and the page below, restretched
in logical pixels on every resize event. The bar is a local page, so the
ACL trusts it by default and the frame works with ANY bundle the daemon
serves; no initialization script touches the served UI anymore.

Reverted: web/ frame mode (WindowControls, shellFrame, padding rules),
the reserved-slot convention in benchmarks.md, and management.html's
embedded controls. ADR-0121 marked superseded by ADR-0122 (accepted),
with alternatives and costs (unstable feature, manual relayout, 40px of
content height). Menus in the bar deliberately deferred.

Verified: cargo xwin build; ci-scoped green; exe installed + relaunched
(PID confirmed). OWNER-VISUAL PENDING — this one is exactly what the
owner asked to judge (bar look, drag, buttons, resize behavior).

Debts: resize relayout is per-event rect assignment (watch for flicker
on slow machines); menu strip (File/Edit/View) deferred; unstable
feature pin noted in the ADR.

Merge: fast-forward ready.
