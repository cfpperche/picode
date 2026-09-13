# 2026-09-12 — feat/inspector-hide

- Removed the duplicate "Hide inspector" button from the inspector panel
  header (`Inspector.jsx`); closing the rail stays with the `InspectorToggle`
  at the end of the editor tab strip. Dropped the now-unused `onToggle` prop
  and the header's chord helper; `App.jsx` no longer passes `onToggle`.
- One fix covers browser and desktop: the desktop entry boots the same
  browser app (`web/desktop/src/main.jsx` → `@picode/browser/src/bootstrap.jsx`).
- Verified on a scratch instance (`qa-scratch.sh start insphide`, seeded):
  toggle hides/shows the rail, aria-pressed and label flip, state survives
  reload, overlay audit `ok:true`, rows aligned. Screenshots read:
  `var/screenshots/insphide-open.png`, `insphide-hidden.png`.
- visual-review: PASS (insphide-open/hidden + overlayAudit ok; card 5/5)
