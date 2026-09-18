# 2026-09-17 — fix-desktop-tab-dnd: desktop tabs reorder again (HTML5 drag/drop unblocked)

Shipped: `desktop-shell/src/main.rs` main-window builder now calls
`.disable_drag_drop_handler()`. Tauri's native drag-drop handler was swallowing
the HTML5 `dragstart`/`drop` events on Windows, so editor-tab reordering
(`AgentTabs.jsx`, `moveTab`) worked in the browser app but not in the shell.
Window dragging is unaffected (`data-tauri-drag-region` is pointer-driven).
Verified: `cargo xwin check --locked --target x86_64-pc-windows-msvc
--all-targets` (2 pre-existing warnings), `node --test openTabs/tabStrip`
24/24, `make ci-scoped` PASS (full). `cargo fmt --check` reports a pre-existing
repo-wide baseline drift; not touched.
visual-review: UNVERIFIED (needs the real WebView2 shell on Windows).
Not done: native confirmation on a Windows machine — drag a tab both
directions, drag the title bar, drop a file onto the composer.
Merge: fast-forward ready after main is merged in (main moved 3).

## Debts

- native Windows check of the rebuilt shell (drag reorder, titlebar, file
  drop) — owned by `docs/handoff/open/desktop-shell-dnd.md`
