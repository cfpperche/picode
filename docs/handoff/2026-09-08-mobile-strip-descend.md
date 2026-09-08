# 2026-09-08 — feat/mobile-strip-descend: tab buttons as low as standalone allows + strip probe
Shipped: in letterboxed standalone the tab content and its pill anchor to
the bar's bottom edge (was centered — ~11px of dead nav under the labels).
Standalone detection now measures (screen.height vs. visualViewport.height,
flag only when the gap > 24px) because WebKit 317153 shows the behavior
depends on iOS generation and icon install date — edge-to-edge installs
keep their insets. Opt-in `?strip-probe=1` extends `#m-app` by
`env(safe-area-inset-top)` into the unreachable strip to settle whether
element painting survives below the layout viewport.
Verified: ci-scoped PASS; scratch 390×844 dark, letterbox simulated
(`#m-app{bottom:47px}`), pill bottom 1px / label 5px above bar bottom,
overlayAudit ok; probe-success reference captured at 390×891.
visual-review: PASS (stripdesc-dark-low.png, stripdesc-probe-success-ref.png — read)
Not done / debts: owner to run `?strip-probe=1` on the iPhone (re-add the
home-screen icon from that URL, query params survive) — if labels survive
at the physical bottom, make the probe offset the default; else delete the
4-line rule and the param.
Merge: fast-forward ready.
