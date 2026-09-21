# 2026-09-21 — feat/desktop-ctrl-r: Ctrl+R / F5 reload the page on screen in PiCode Desktop

Shipped: commit ee091f3f, plus merges of main. In Desktop, Ctrl+R and F5 reload the page on screen:
a visible work-browser/web-app tab or an agent split reloads that native page (`btab_reload`); a
terminal pane keeps Ctrl+R for readline; everywhere else reloads PiCode. How: the chrome webview
(`main-content`) turns off WebView2's browser accelerators
(`ICoreWebView2Settings3::SetAreBrowserAcceleratorKeysEnabled(false)`, desktop-shell/src/main.rs) so
the engine no longer eats the chord on the chrome; page webviews KEEP them, so a focused page
still reloads itself. `web/browser/src/lib/desktopReload.js` is the decision table (`isReloadKey`,
`desktopReloadAction`), wired in `web/browser/src/App.jsx` behind shellChrome + `window.__TAURI__`.

Verified: `desktopReload.test.js` decision table (7 tests, node --test); cargo xwin check green;
desktop-test green; full ci-scoped matrix green twice (close rite). Docs: docs-site/guide/keyboard.md
and web-apps.md updated; architecture notes in docs/architecture.md and
docs/architecture/work-browser.md; changelog fragment docs/changelog.d/desktop-ctrl-r.md.

## Next up

- The shell half (main.rs) reaches Windows only via `make desktop-restart` — the UI half ships with
  the next deploy. Live acceptance on Windows (Ctrl+R in a work tab reloads the page, in a terminal
  stays reverse-search) is the owner's run; the swap is now serialized behind /tmp/picode-mutate.lock
  (feat/harden-restarts, landed and deployed).
- Born before the 2026-09-21 incident; its first deploy attempt was the one that raced
  desktop-restart (see `2026-09-21-harden-restarts.md`). No debt of its own.
