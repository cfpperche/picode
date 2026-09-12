# 2026-09-12 — feat/sketch-ui-mobile: Excalidraw's controls arranged for a phone

Reference: upstream's `excalidraw--mobile-toolbar` layout (excalidraw.com on a phone, master) — tools island
at the bottom, shape actions above, no library/lock/hand column; npm's latest is 0.18.1, which still renders
the top row, so the arrangement is pinned in `web/mobile/src/styles/app.css` (mobile-only, `#m-app`-scoped).
Both `SketchEditor.jsx` copies pass `canvasActions: { loadScene, saveToActiveFile, export, saveAsImage: false }`
(background and reset stay); the desktop layout is untouched.

Verified (scratch `uitest`, iPhone 16 Pro emulation 402×874; screenshots um-01…um-04 read):
| condition | result |
|---|---|
| phone, empty | tools at 818 (16px from the bottom), actions at 764, hint under the header, no misc column |
| phone, drawing | pen stays active across strokes; Insert → chip, sheet reopens |
| phone, menu | Find / Help / Reset / links only — no Load, Save, Export, Save-as-image |
| phone, dark | same arrangement, dark canvas |
| composer pin sketch | same rects (tools 818, actions 764, misc hidden) |
| desktop 1280×800 | Excalidraw desktop layout unchanged; menu trimmed the same |
| two-finger pan/zoom | source-confirmed (2-pointer gesture); not gestured on device — debt |

visual-review: PASS (`__picodeOverlayAudit()` ok everywhere; card 5/5)

Debts: no physical-phone gesture pass (pan/zoom, and the hand tool is now absent from the pad); the CSS
targets Excalidraw 0.18.1 internals — drop it when the package ships the mobile-toolbar layout;
`docs/handoff.md` is at the 8 KB cap, so this note carries the detail.
Merge: fast-forward into main after `make close`.
