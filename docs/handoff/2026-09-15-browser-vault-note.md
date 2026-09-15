# 2026-09-15 — browser-vault-note

Owner asked why the reference's Password manager can show a list/Add/import
while ours says it cannot.

## Done

- Measured the WebView2 SDK surface (webview2-com-sys 0.38.2) and recorded the
  finding in `docs/handoff/open/work-browser-tabs.md`: the password/autofill
  API is the two toggles and nothing else, while the SDK does expose
  `OpenTaskManagerWindow` where a runtime UI exists — so no password UI is
  openable. The reference's screens are its own vault or the OS page.
- Two paths recorded, both owner-gated: opener rows (no vault) and
  PiCode-owned vault (ADR, security model).
