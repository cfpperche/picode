# 2026-09-14 — browser-autofill

Slice 3 increment 3.3b: password autosave + general autofill toggles.

## Done

- Shell: btab_set_prefs pushes (passwordAutosave, generalAutofill) into a
  static; ensure() applies both to every new webview
  (ICoreWebView2Settings9) and btab_set_prefs re-applies to all live
  btab webviews. Defaults match WebView2 (both on).
- Go: prefs GET/PUT carry both booleans (PUT = full save of the editor
  state; destinations validated); test updated + extended.
- UI: Autofill and passwords card (two On/Off switches), invoking the
  shell after each save.
- Scratch: toggle → label flips → stored false, overlay audit ok.

## Notes

- Parity checklist: 3.3c clear browsing data, 3.3d downloads, 3.3e site
  permissions remain.
