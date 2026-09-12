# 2026-09-12 — the plane answers its own right-click

`fix/canvas-owns-right-click` → `main` (bbfda3b4). Shipped broken hours
earlier in the same session.

## What was wrong

PiCode's generic context menu lives on `document` (`App.jsx` →
`components/ContextMenu.jsx`). Radix's `ContextMenu.Trigger` on the plane
never stopped it, so a right-click on a canvas got Copy / Paste / Reload
PiCode. With the chrome hidden that menu was the *only* answer, and **Show
controls** lives in the canvas menu — a lockout with no UI way out. The
escape hatch, for anyone who hits an old build:
`localStorage.setItem("picode-canvas-chrome","shown")`.

## The fix

`onPlaneContextMenu` decides per event: `preventDefault` + `stopPropagation`
for the plane (the host listens below React's root, so stopping there is
enough), untouched for a `.cv-panel` so the host's terminal menu still
answers, stopped-but-silent after a right-drag pan. The menu is a controlled
`DropdownMenu` on a zero-size anchor at the cursor — what the host's own menu
does, and its file says why. `@radix-ui/react-context-menu` removed.

## Why the QA missed it

The earlier test dispatched a synthetic `contextmenu` straight at
`.cv-canvas-flow`, which skipped the document listener that was going to win.
**Use the real button**: `agent-browser mouse move x y` → `down right` →
`up right`, ~0.3 s apart.

## Also learned, the hard way

Another session split the surfaces the same day (`1db0881e`): the web app is
now `web/browser` at **`/browser/`**, and `/desktop/` is the shell. The canvas
tree moved with it — git carried my in-flight edits across the rename without
a conflict. A plain browser opening `/desktop/` on a scratch build renders
unstyled; QA the web app at `/browser/`.
