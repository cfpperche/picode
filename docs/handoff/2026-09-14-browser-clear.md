# 2026-09-14 — browser-clear

Slice 3 increment 3.3c: Clear browsing data.

## Done

- Shell: `btab_clear_data` — reaches the shared profile through any open
  browser tab, casts ICoreWebView2_13 → Profile → ICoreWebView2Profile2,
  and clears a mask that deliberately excludes passwords/autofill (the
  Autofill switches own those). cargo xwin ✓.
- UI: a two-step Clear browsing data button beside Clear history; also
  clears our history store; toast feedback; no-op safe when no tab is
  open (the button lives in Settings — the error toasts).
## Notes

- Verified by compile + UI wiring only: the mask's effect needs a real
  WebView2 (owner check after deploy — cookies should drop).
- Parity checklist: 3.3d downloads, 3.3e site permissions remain.
