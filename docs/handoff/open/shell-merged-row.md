# Merged top row (shell)

The owner's sketch applied: in shell mode the window's top edge is one
row — brand (mark + wordmark, click = dashboard), the sidebar rail tabs,
the agent tabs with their inspector end-slot, and the Windows caption
buttons at the right end. The sidebar starts at its own content; the
workspace view drops its separate strip. Browser layout untouched
(composition branches on the shellChrome prop). The shell window is a
single undecorated webview again — unstable multiwebview, the relayout
handler and titlebar.html are gone. Management keeps its own header
controls. Drag via data-tauri-drag-region + the granted remote origin.

Constraint found 2026-09-17: **nothing inside the brand button may carry
`data-tauri-drag-region`.** Tauri's drag script answers the mousedown on
the element it lands on, so the wordmark's own `<span>` claimed the press,
started a native window drag and the button's `click` never fired — the
wordmark dragged the window instead of opening the dashboard. The
attribute belongs on `header.shell-row` and `.shell-brand-cluster` only.
The click also has to leave a non-workspace route (Clis, Browser,
Preferences, Devices): the dashboard renders inside `#workspace-view`,
which those routes keep hidden.
