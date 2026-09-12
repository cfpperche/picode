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
