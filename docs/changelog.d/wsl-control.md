### Added
- **PiCode Desktop: the tray now tells you what the disk is doing.** One line
  under the status — `WSL 218 GB · ≈92 GB held by Windows · C: 27 GB free` —
  refreshed every five minutes, ending in `— low` under 20 GB free, with the
  tooltip carrying the same sentence and the command that shows the rest.
- **`picode-desktop disk` reports both halves of a WSL disk in one screen.**
  The Windows side (the VHDX path from WSL's own registry, the file's real
  size, whether it is sparse, free space on the volume) and the distro side
  (filesystem, the caches with the exact command that gives each back, the
  top-level breakdown of home). It shows **held for nothing**: the space the
  distro has already freed that the disk file still occupies — invisible in
  Explorer, and the reason a full `C:` stays full. Read-only.
- **`picode disk` lists what occupies the machine and what is safe to
  reclaim**, one line per item with `safe`, `redownload` or `data`, plus the
  paths it could not read named as such instead of folded into a category.
  `--json` feeds the tray and, later, the Storage app.

### Changed
- A VHDX that is not sparse is named in the report as the reason WSL cannot
  give freed blocks back, with the one command that changes it — it is not run,
  because converting or compacting the file stops the distro and its sessions.
