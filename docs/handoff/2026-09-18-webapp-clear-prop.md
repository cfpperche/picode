# 2026-09-18 — webapp-clear-prop: forward onClearWebappData through Sidebar

Fix: Clear data threw "h is not a function" in the confirm dialog —
`App.jsx` passed `onClearWebappData` to `Sidebar`, but `Sidebar` never
forwarded it to `AppsGrid`, so the menu handler called `undefined`
(minified `h`). One-line prop forward; the Rust command was never
at fault. Found by the owner in the live shell; the WSL QA could not
reach it because the menu item is desktop-only.
Verified: ci-scoped PASS; build green. visual-review: n/a (prop wiring).
Merge: fast-forward ready.
