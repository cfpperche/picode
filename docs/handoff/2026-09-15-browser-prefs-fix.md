# 2026-09-15 — browser-prefs-fix

Owner: "o switch nasce habilitado, se clica nele nao tem como habilitar
novamente" — the Ask where to save downloads switch could not be turned
back on.

## Root cause

`BrowserPage.jsx` mapped the daemon's prefs in two places (the load
effect and the PUT's answer). The second copy never got `askDownload`,
so every save wrote the whole object back into state with that field
undefined → the switch read off. Verified in the scratch: before, click
1 → checked → back to unchecked; now unchecked → checked → unchecked →
checked, and checked survives a reload with the daemon storing true.

## Done

- One reader: `src/lib/browserPrefs.js` (`readBrowserPrefs` +
  `DEFAULT_BROWSER_PREFS`), used by both call sites; the inline copies are
  gone. Three tests, including the round trip that broke.
- Suíte: 343 pass, 0 fail. Visual: switch toggles both ways, persists
  across reload.

## Next up

- Browser permissions (camera/mic), Developer mode; and the ACL lesson
  (a new shell command is three edits) to the traps list in
  `docs/handoff/open/work-browser-tabs.md`.
