# 2026-09-12 — feat/appbar-brand: one wordmark per window

Owner round on the app bar: two PiCode wordmarks stacked (app bar +
sidebar brand row) read as a defect. In shell mode the served UI hides
the sidebar's wordmark button (the tab rail stays; drag still wired by
the shell) and the app bar's wordmark takes the action: clicking opens
the dashboard via the picode-open-dashboard DOM event — the same
pattern as picode-open-file — dispatched by the new open_dashboard
command through Webview::eval into main-content. Browser unchanged
(flag exists only inside the shell). Web 294/294, make web, cargo
xwin build, ci-scoped green; exe installed + relaunched, PID confirmed.
Owner-visual pending (single wordmark, click-through to dashboard).

Debts: dashboard state is per-load (a reload resets the pin) — same as
the browser behavior; menu strip in the bar still deferred.

Merge: fast-forward ready.
