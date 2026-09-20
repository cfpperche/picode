# 2026-09-17 — terminal-link-destination

The owner's report: Ctrl+click on a link in an agent's terminal went to the
Windows browser instead of PiCode's.

## What was wrong

`web/shared/domain/termLinks.js` called `window.open(href, "_blank")` for every
http link, and inside the shell that is exactly what the external-links bridge
(`web/browser/src/lib/externalLinks.js`) reroutes to the system browser. So the
terminal door ignored the preference the rest of the app already had: the same
URL through the pane menu opened in PiCode, while Ctrl+click left the app.

## Fixed (owner's call 2026-09-17: terminal links are browsing actions)

- `wireTermLinks(term, cwd, onFile, liveCwd, openHttp)` takes the app's opener;
  a bare terminal keeps `window.open`.
- `linkOpenTarget(link, localOpenDest, webOpenDest)` — loopback reads the local
  preference, every other host the web one; both default to PiCode.
- `ShellTerm`/`TermSurface` carry `onOpenLink`; `App.jsx` builds it (one fresh
  prefs read, then `openWebTab` or `window.open`) and the pane menu's Open path
  now uses the same two preferences.
- Tests: `termLinks.test.js` 10/10 (the new one proves the opener is used and
  `window.open` is not), `openLink.test.js` walks the new table, `make web`
  builds, `npm --prefix web/browser test` 382 pass.

## Not verified

The real gesture: a human Ctrl+click on a printed link must land in a
work-browser tab. The oracle is the history store — the target URL appears
there when PiCode's browser hosted it and not when the system browser did.
