# 2026-09-18 — webapp-chromeless: installed web apps open app-mode in the shell

Shipped: owner chose option B (in-shell, no separate window). An installed
web app whose manifest declares `standalone`/`fullscreen`/`minimal-ui`
opens chromeless: no back/forward/reload/URL bar — a minimal titlebar with
the app's own name left and the ⋮ menu right (Reload + Copy address added
at the top for app mode; the rest of the browser actions unchanged).
`display: browser` and plain work tabs keep the full toolbar. Installed
tabs are labeled with the app's name and manifest icon in the tab strip
(AgentTabs now receives webapps). ADR-0147 amendment records the decision;
the standalone-window door stays refused while the owner works in-shell.
Verified: `webappChromeless` matrix + launch-rule tests (8 JS tests);
scratch /desktop/ with a `__TAURI__` stub — chromeless tab renders
titlebar only, menu shows Reload/Copy address, `overlayAudit` ok, regular
work tab keeps its toolbar (screenshots read).
visual-review: PASS for layout (titlebar/menu/tab-label); the page canvas
is stub-blank by construction — the live page fills it only in the real
shell (owner verifies on Excalidraw).
Merge: fast-forward ready.

## Next up

- Owner clicks Excalidraw in the desktop shell: page fills the surface, tab strip label and ⋮ menu behave (recorded in open/installed-webapps.md).

## Debts

- docs/handoff/open/installed-webapps.md: standalone-window (Chrome-style) stays refused while the owner works in-shell; revisit costs an ADR.
