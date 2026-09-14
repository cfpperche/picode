# 2026-09-14 — feat/html-preview: HTML file preview v1 (ADR-0136)

Shipped on `feat/html-preview`: `.html`/`.htm` render as real pages in the
file pane, the file tab, a canvas file panel and the mobile Files screen.
`POST /api/previews` mints an hour-long, session-bound ticket; the
method-less `GET|HEAD /preview/<token>/<path>` route serves the document and
its relative assets through the closed MIME allowlist, under the owner's
canonical folder, with dotfile + symlink containment. Every response carries
`CSP: sandbox …` (never `allow-same-origin`; verified: a sandboxed frame
sends no cookie and its writes are refused as `Origin: null`),
`Referrer-Policy: no-referrer`, `no-store`, `Access-Control-Allow-Origin: *`
for modules/fetch. The pane mints via `usePreviewTicket` (browser and mobile),
HEAD-preflights, pauses while the editor is dirty ("Unsaved changes aren't in
the preview." + Save and preview), and offers Reload / Open in browser.
Docs: `docs/architecture/file-preview.md`, `routes.md`,
`security-model.md`, `docs-site/guide/files.md`, ADR-0136, changelog
fragment. Visual evidence in `var/screenshots/html-preview-*.png`.
Verified: `make ci-scoped` PASS (fmt, vet, hooks, go[5], test-js, build,
docs; 37 paths) — includes the new Go decision-table tests and 783+357+337
frontend tests. Visual QA on scratch instance `html-preview` (HTTP :8471,
fixture in `var/qa/html-preview/fixture`): ready light+dark, empty (desktop
and 390×844 mobile), dirty notice, gone-file error, external tab, mobile
toolbar with all four controls; `window.__picodeOverlayAudit()` ok on
desktop and mobile; every screenshot read. Docs captures re-run
(`make docs-shots`, 5 surfaces, 0 px differ except app-inspector at 79 px
within budget) and the manifest committed.
visual-review: PASS (card 5/5; no overlay in this flow)
Not done: no live reload (v1.5), unsaved HTML is not previewed by design,
no console capture (opaque origin), documents >1 MiB cannot mint (the pane's
text read gates Raw).
Merge: not merged — `make close` is blocked by the handoff board already
over its byte cap on `main` (see `docs/handoff/open/process.md`); every other
close step is green and the tree is clean.
