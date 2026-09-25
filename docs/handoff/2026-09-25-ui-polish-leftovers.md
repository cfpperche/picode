# 2026-09-25 — ui-polish-leftovers: host placeholder, focus ring, one divider, phone tabs

Shipped: account chip placeholder until /api/system answers ("local" only on failure); `.btn:focus-visible` accent
ring (both apps); a section's top rule only after a visible section (no double divider in Preferences); phone
Preferences remembers its tab (localStorage, per viewer); phone PageFrame re-masks partly visible tabs on scroll and
reveals the selected tab on selection change/resize — a swiped-in tab stayed hidden and untappable (pre-existing
since faea89b8c; Preferences → Landing work unreachable at 390 px).
Verified: `make ci-scoped` PASS; visual-review PASS on scratch (chip timeline never shows "local"; ring 2px accent
offset 2 on .btn-primary/.btn-ghost light and dark; one divider on all tabs desktop + phone; tab remembered across
reload; swiped tab visible and tapped by a real click; Agent CLIs pane tabs still work).
Blind spot: the strip scroll was driven by eval (a real drag/wheel did not scroll it in the headless browser); taps
were real. Seen, not fixed: pref tabs, selects, inputs and switches still use the browser's default focus ring.
