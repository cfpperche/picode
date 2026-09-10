# 2026-09-10 — term-key-capture: fullscreen hands the browser's reserved keys to the page
Shipped: focus mode (fullscreen) now also locks the keyboard — `navigator.keyboard.lock()` before
`requestFullscreen()`, same gesture, `unlock()` on the leave-first exit (`web/desktop/src/lib/useFocusMode.js`).
`web/shared/domain/browserChord.js` is the Chromium reserved set as data (Ctrl rows = Windows/Linux,
super rows = macOS equivalents); guard test keeps every app-keys default off it. Docs: routes.md
focus-mode paragraph, docs-site/guide/keyboard.md + sidebar, changelog fragment.
Verified: `make ci-scoped` PASS (fmt, vet, hooks, test-js, build, docs/vale); scratch instance: chord
enters/exits mode cleanly, mode survives reload restore, `__picodeOverlayAudit` ok, console clean.
visual-review: PASS (termkey-2-focus-on.png read; toast text verified in served bundle — toast card
itself not captured on screen, sonner mounts lazily)
Not done / debts: (1) lock capture not provable in headless — control test showed headless passes
Ctrl+T to the page even unlocked, so real-window proof of capture + hold-Esc toast UX is on the
owner's Chrome; (2) AppKeys accepts binding a reserved chord with no warning (dead outside
fullscreen); (3) reserved set is a snapshot of Chromium today — re-measure when browsers move;
(4) surfaces that cannot take the lock (webview embeds) degrade silently by design.
Merge: fast-forward ready — `cd /home/goat/picode && git merge --ff-only feat/term-key-capture && make ci`.
