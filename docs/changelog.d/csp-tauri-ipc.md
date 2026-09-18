### Fixed
- **Windows shell caption buttons and native commands work again.** Covering
  `/desktop/` with the app CSP had blocked Tauri's IPC (`http://ipc.localhost`),
  so every `invoke()` died in the console. The desktop shell's policy now
  names that host; the browser and mobile shells do not.
