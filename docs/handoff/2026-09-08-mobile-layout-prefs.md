# 2026-09-08 — feat/mobile-layout-prefs: Preferences → Layout (bottom bar dials)
Shipped: mobile Preferences gains a Layout tab — Bottom bar buttons
(auto / low / screen edge) and Bottom bar height (48/56/64px). Prefs live
in `web/mobile/src/lib/layoutPrefs.js` (localStorage `picode-layout`,
same pattern as theme/toastPrefs), applied pre-paint from `main.jsx`.
"Auto" resolves against the bootstrap's letterbox measurement
(`data-standalone`), so edge-to-edge installs keep real insets. "Screen
edge" is the old strip probe as a user-controlled dial (clipping risk
stated in the option label). `?strip-probe=1` removed — superseded.
Side fix: Preferences tabs are component state now; the desktop-hash
deep links (`#/preferences/<tab>`) silently fell back to Appearance on
the phone, so the chosen tab never survived navigation.
Verified: layoutPrefs.test.js (4) + full test-js, ci-scoped PASS; scratch
390×844 dark: Layout panel renders with persisted values, selects flip
`data-bottombar`/`--m-nav` live, survives reload, overlayAudit ok,
pushed + tab screens checked.
visual-review: PASS (layoutprefs-panel-dark.png, layoutprefs-now-dark.png — read)
Not done / debts: on-device confirmation on the owner's iPhone; if "Screen
edge" clips labels there, the honest ceiling for that device is "Low".
Merge: fast-forward ready.
