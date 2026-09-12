### Added
- Desktop shell: the app's own chrome now draws the window controls when
  running in the shell (frame mode, ADR-0121) — no floating overlay; every
  view's top bar reserves the slot. Until the web UI is deployed the
  shell's injected fallback frame stays in charge and retires itself once
  the new UI is live.
### Fixed
- Desktop shell: window buttons from the served UI were refused by
  Tauri's ACL; the daemon origin is now trusted for the core window
  permissions.
