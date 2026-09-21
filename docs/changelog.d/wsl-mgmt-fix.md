### Fixed

- **The Management window answered "not allowed by ACL" on every tab.** Its
  seven commands — `disk_report`, `disk_compact`, `disk_compact_dry_run`,
  `clean_list`, `clean_apply`, `wslconfig_read`, `wslconfig_write` — sat on the
  ACL guard's exception list under the wrong assumption that no webview invokes
  them. The Management window is a served webview, so Tauri refused each call.
  They now live in a dedicated `management` capability
  (`desktop-shell/capabilities/management.json`), the exception list shrinks to
  the one genuinely tray-internal command (`computerlab_open`), and the guard
  test enforces the management commands too.
- **`clipboard_files` was missing from `build.rs`'s command manifest.** It was
  registered in `generate_handler!` and granted in `capabilities/default.json`
  but absent from `AppManifest::new().commands(&[…])` — the one-line gap the
  same guard test flags, and one a fresh permission regen would have surfaced
  as a dangling capability reference.

### Removed

- **The tray's disk line and Give back item.** The tray no longer shows the
  `WSL … GB · ≈… held by Windows · C: … free` line or **Give back ≈N GB…**.
  The disk facts and the give-back flow live in the Management window's **Disk**
  tab — the surface this change also makes work. Removed with them: the tray's
  disk poller, the Rust-side compact flow (the Go `disk-compact` command keeps
  the readiness interlock), and `desktop-shell/src/diskline.rs`.
