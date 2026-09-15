### Fixed
- **The work browser's settings switches and actions now actually reach
  the app.** The shell's ACL manifest (`build.rs`) never listed the
  commands added after the first slice — `btab_set_prefs`,
  `btab_clear_data`, `btab_open_external` and the whole Downloads set —
  so every invoke was refused with "not allowed by ACL" and the page
  swallowed the error. The commands are in the manifest and the
  capability now, with their generated permission files.
- **The download folder is picked with the app's folder browser** (the
  same one the Add workspace dialog uses) instead of a typed path, and
  the picked place is translated from the daemon's tree to the Windows
  drive the browser writes on (`/mnt/c/...` → `C:\...`). A folder
  outside a Windows drive is refused with a line saying why.
