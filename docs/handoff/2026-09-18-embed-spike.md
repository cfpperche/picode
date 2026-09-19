# 2026-09-18 — feat/embed-spike

Owner accepted a spike: can Windows apps open inside PiCode's window, in a
pane like the work browser? `desktop-shell/src/embed.rs` adds two commands
reachable only from the Computer lab (`capabilities/computerlab.json`):
`computer_embed(window)` strips the caption/frame styles, sets `WS_CHILD`,
`SetParent`s the window under the main window (mixed DPI hosting on) and
places it in the right half; `computer_unembed(window)` restores style,
parent and rectangle. Cross-process parent/child is legal and unsupported;
the child dies with the parent, so unembed before closing PiCode.

Measurement plan (after `make desktop-restart`): Windows Terminal, Explorer,
VS Code, Windows 11 Notepad — render inside? keyboard after a click? where
do dialogs and menus open? resize of the main window? unembed intact? The
matrix and the recommendation go to
`docs/benchmarks/2026-09-18-embed-windows-apps.md`; the decision (mirror,
reparent, or capture + input) is the owner's.

## Next up
- Owner: `make desktop-restart`; run the four apps through the lab with me and read the matrix.

## Debts
- Embedded windows are not given back on shell exit (`embed::embedded` lists them; a shutdown hook is the fix if the spike becomes a feature).
