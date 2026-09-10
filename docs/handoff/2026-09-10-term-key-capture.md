# 2026-09-10 — term-key-capture: fullscreen hands the browser's reserved keys to the page
Shipped: focus mode (fullscreen) now also locks the keyboard — `navigator.keyboard.lock()` before
`requestFullscreen()`, same gesture, `unlock()` on the leave-first exit (`web/desktop/src/lib/useFocusMode.js`).
`web/shared/domain/browserChord.js` is the Chromium reserved set as data; guard test keeps every
app-keys default off it. First-run toast copy updated. Docs: routes.md focus-mode paragraph,
docs-site/guide/keyboard.md + sidebar, changelog fragment.
Verified: `make ci-scoped` PASS (fmt, vet, hooks, test-js 46 incl. 4 new, build, docs/vale); scratch
instance (qa-scratch termkey): chord enters mode, `Ctrl+T` reaches the page (xterm textarea) with
mode intact, toggle-off exits fullscreen cleanly, `__picodeOverlayAudit` ok, console clean.
visual-review: PASS (termkey-2-focus-on.png read; toast text verified in served bundle — toast card
itself not captured on screen, sonner mounts lazily)
Not done / debts: real-window proof of the hold-Esc toast UX happens on the owner's Chrome
(headless has no tab chrome); Firefox/Safari keep today's behavior by design (no Keyboard Lock).
Merge: fast-forward ready — `cd /home/goat/picode && git merge --ff-only feat/term-key-capture && make ci`.
