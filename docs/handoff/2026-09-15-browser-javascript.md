# 2026-09-15 — browser-javascript

Closes the second row of Browser permissions.

## Done

- Shell: `btab_set_scripts(enabled)` applies `ICoreWebView2Settings::
  IsScriptEnabled` to every live tab and remembers the value for the ones
  created later (`apply_scripts`, next to the autofill application).
- `browser.scriptsEnabled` pref (default on) with the round-trip assertion,
  read through `readBrowserPrefs` and pushed to the shell on every save.
- UI: the JavaScript row with its switch; refusals toast.
- ACL: the three edits, generated permission committed (`btab_set_scripts`).
- cargo xwin ✓, web ✓, JS tests 343 ✓, Go prefs test ✓.

## Still open in Browser permissions

- The **Ask prompt** — the third policy state. It needs the deferral map on
  the UI thread (COM objects are not Send: keep them in a thread-local, the
  same trap as the event receivers) plus a prompt surface in the tab. That
  is its own slice; until then the select offers Allow/Block/default and the
  dialog says so.
