# 2026-09-12 — feat/sketch-mobile: the phone sketch pad fits the usable screen

Shipped: mobile `SketchEditor`/`PinSketch` portal into `#m-app` (not `document.body`); the pad is
`absolute; inset: 0` inside it, the head takes `.m-head` metrics (52px over
`env(safe-area-inset-top)`, `--bg-panel`), Cancel/Insert are 44px, the pad carries
`var(--sa-bottom)`, the phone opens with the pen, and the canvas is white in both themes
(Excalidraw's dark filter inverts it) so dark mode is dark and the PNG stays a plain sheet.
Docs: `docs/architecture/cli-terminal-launch.md` + `docs/changelog.d/sketch-mobile.md`; no ADR.
Decision table (probed on `iosketch`, iPhone 16 Pro emulation):
| condition | result |
|---|---|
| phone + dark | dark canvas/chrome, white PNG export |
| phone + light | light UI, white canvas |
| simulated insets 47/34px | head clears the clock, footer clears the home indicator |
| keyboard open (`kb-open`, 500px) | pad follows `--vv-height` |
| desktop | unchanged (body portal, no safe areas) |
| real iPhone `env()` insets | not run on device — simulated by injected CSS; debt |

visual-review: PASS (sm-01…sm-06 read; `__picodeOverlayAudit()` ok; card 5/5)

Not done / debts: safe-area insets were simulated, not measured on iOS (the standalone dead strip
is the shell's html/body background and read continuous); the pin annotate flow still asks for a
dark `viewBackgroundColor` in dark mode, rendered light by the filter; `docs/handoff.md` sits at
8.1 KB of the 8 KB cap, so those debts live only here.
Merge: fast-forward into main after `make close`.
