# 2026-09-15 — browser-acl-fix

Owner test after the Downloads deploy: the Ask toggle and the folder
Change did nothing, with "Command btab_set_download_dir not allowed by
ACL" on screen.

## Root cause (and the wider hole)

`desktop-shell/build.rs` carries the explicit app manifest: the commands
listed there get `allow-<name>` / `deny-<name>` permissions, and only
those can appear in `capabilities/*.json`. It was never extended after
the first slice, so `btab_set_prefs`, `btab_clear_data` and
`btab_open_external` have been refused since they shipped — every call
sat behind `.catch(() => {})`, which is why the switches looked fine and
did nothing. Lesson for the traps list: **a new `#[tauri::command]` is
three edits** — `generate_handler!`, `build.rs`, `capabilities/default.json`.

## Done

- build.rs gains the eight missing commands; the build regenerated their
  permission files (25 → 33) and the ACL schemas; the capability lists
  them. cargo xwin ✓.
- Folder picking reuses `FolderPicker` (the Add workspace dialog's own),
  with `/mnt/<drive>` ↔ `<DRIVE>:\` translation; a non-Windows folder is
  refused with an explanation. The Change button's stale `setDirDraft`
  reference (a ReferenceError that kept the dialog from opening) is gone.
- Shell refusals now toast instead of vanishing (setPref, ask, wipe).
- Visual: picker opened at C:, listing the real drive; picking it set
  Location to `C:\` and confirmed with a toast.
