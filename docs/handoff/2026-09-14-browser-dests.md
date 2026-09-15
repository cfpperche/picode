# 2026-09-14 — browser-dests

Slice 3 increment 3.3a: web/local open destinations (ChatGPT Work
settings parity, owner mandate 2026-09-14).

## Done

- Prefs extended: webOpenDest/localOpenDest (app | external, default
  app), GET folds them, PUT validates; table-tested round trip + 400.
- Shell: `btab_open_external` — scheme-checked, `cmd /C start` with
  CREATE_NO_WINDOW (no keepalive duty; the tray owns the VM lifetime).
- UI: Open destinations card (two native selects on the grant-row
  rhythm); the btab://new popup handler reads the pref fresh per popup,
  routes localhost to the local destination, everything else to the web
  destination, and falls back to adopting when the read fails.
- cargo xwin build ✓; screenshot var/screenshots/open-destinations.png.

## Notes

- Parity mandate context: this is one of six increments; the checklist
  in the session tracks 3.3b (autofill), 3.3c (clear data), 3.3d
  (downloads), 3.3e (site permissions).
