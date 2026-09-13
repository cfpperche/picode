# Work browser tabs (Phase 3 slice 1) — ACL blocker + exact fix

2026-09-13, browser-lab worktree (branch feat/browser-tabs, merged through
0c480b3e; slice-1 React fixes landed on main through 83916715/b42e02b3/
e6f9b770/a8783811; async-commands + error surfacing 656045cd).

## State

- Browser tabs render (strip, toolbar, start state) but **navigate is
  blocked by the Tauri ACL**: the toolbar shows "Command btab_meta not
  allowed by …". App commands in tauri 2.11 are NOT auto-allowed and
  CANNOT be referenced as bare `allow-btab-*` in capabilities (build
  fails validation) nor `__app__:allow-*` (namespace not generated —
  `__app__-permission-files/` is empty; tauri-build does not autogenerate
  app-command permissions from `generate_handler!`).
- Disk commands (disk_report etc.) work from the same webview — find how
  they are wired before assuming anything (maybe a permissions manifest
  already exists for them, maybe they predate the ACL split).

## Exact fix path

Tauri 2 documents app-command permissions via **in-crate permission
manifests**:

1. `desktop-shell/permissions/btab/default.toml`:
   ```toml
   [default]
   description = "Work browser tab commands (Phase 3)"
   permissions = []
   ```
   plus per-command `[[permission]]` entries with
   `identifier = "allow-btab-navigate"` and `commands = ["btab_navigate"]`
   (one file per command, or a set).
2. `build.rs`: pass
   `tauri_build::Attributes::new().app_manifest(tauri_build::AppManifest::new())`
   to `tauri_build::try_build` so the crate's own permissions are collected
   (the `__app__` namespace then resolves them).
3. Capabilities: add `"allow-btab-*"` (or the set id) to
   `capabilities/default.json` for windows `["main"]` /
   webviews `["main-content"]`.

Then rebuild; the toolbar error disappears and navigate paints the page.

## Also open

- Slice-1 UX requests (owner): globe icon beside the Inspector toggle
  (opens a new browser tab) + a Browser entry in the user menu leading to
  a Browser settings page — NOT yet implemented.
- The parity spec (docs/plans/desktop-v2.md) carries every requirement;
  slices 2–4 (CDP bridge, policy engine, policy UI) per ADR-0128.
- The other session's rewinds stopped since the guard landed; verify with
  `git log --oneline main` before merging.
