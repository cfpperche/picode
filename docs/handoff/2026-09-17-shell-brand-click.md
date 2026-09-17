# 2026-09-17 — shell-brand-click

The owner's report: clicking “PiCode” in the desktop app did not open the
dashboard.

## What was wrong

Two defects on the same click, both in `web/browser/src/App.jsx`.

1. The shell top row's brand is a `<button>` inside a
   `data-tauri-drag-region` header, and both inner spans (`shell-mark`,
   `shell-name`) still carried the attribute from when the brand was a
   `<span>`. Tauri's drag script claims the mousedown on the element it
   lands on, so clicking the wordmark started a native window drag
   (`ReleaseCapture` + `WM_NCLBUTTONDOWN HTCAPTION`); the webview never saw
   the mouseup and the button's `click` never fired. The logo *icon*
   worked (the SVG is not an `HTMLElement`, so the drag walk skipped it) —
   which is how the defect survived the 2026-09-12 owner round.
2. The click only pinned the dashboard, never the route. The dashboard
   renders inside `#workspace-view`, `hidden` while the route is not
   `workspace`, so from Agent CLIs / Browser / Preferences / Devices the
   sidebar wordmark (browser layout) also looked dead.

## Fixed

- `openDashboard` (App.jsx) is now the one handler for both wordmarks: it
  pins, closes the mobile nav, and returns to the workspace route when the
  click comes from another one (a tab's own `#/agent/<id>` hash is left
  alone).
- The two spans lost `data-tauri-drag-region`; the row and the brand
  cluster keep it. `docs/architecture/routes.md` and
  `docs/handoff/open/shell-merged-row.md` carry the constraint.

## Evidence

Scratch `brand-click` (:8474), tauri 2.11.5's own `drag.js` injected as an
init script with a recording `__TAURI_INTERNALS__.invoke`:

| Build | Click | Route | Action | Result |
|---|---|---|---|---|
| shell | control probe with the attr | any | drag | `start_dragging` logged — proves the injection is live |
| shell | wordmark text | `#/agent/<id>` | dashboard | no drag, dashboard on, hash kept |
| shell | logo mark | `#/agent/<id>` | dashboard | no drag, dashboard on |
| shell | wordmark text | `#/clis` | dashboard | no drag, workspace view un-hidden |
| browser | sidebar wordmark | `#/clis` | dashboard | dashboard on |
| any | — | no data | blank slate | unchanged by this diff |

`__picodeOverlayAudit()` ok on both layouts; screenshots read:
`var/screenshots/brand-click-dashboard.png` (dashboard over the open tab),
`var/screenshots/brand-click-hover.png` (hover still paints the wordmark
accent). No unit test exists for a React click handler (no DOM harness in
`web/`), so the table is covered by the browser run above; the Windows
native drag swallow itself is verified by the injected script, not by a run
inside the shell. Owner-confirmed 2026-09-17: the wordmark opens the
dashboard in the desktop app on DESKTOP-BGG95NA.

visual-review: PASS (brand-click-dashboard.png + brand-click-hover.png,
overlayAudit ok; card 5/5)

## Debts

- None new.
