### Added
- Desktop shell: the tray item is now **Management** — a window with three
  views over the WSL distro. **Disk** shows what the distro holds and runs
  the Give back flow with live progress; **Clean** measures the prunable
  caches (`picode clean --list`) and prunes the selected ones — Pi sessions
  and the PiCode database are never cleanable; **Config** edits
  `.wslconfig` (memory, processors, swap, sparse disk) with an automatic
  backup, leaving unknown settings untouched; changes apply at the next
  full WSL restart.
- `picode clean` — new subcommand that prunes the caches `picode disk`
  measures (`--list` to measure, `--apply id1,id2 --yes` to prune,
  `--json` for tools). Data directories are refused even when asked for by
  name.
- Desktop shell: sharper taskbar/window icon (the icon file's largest
  image now comes first) and the shell's local pages use the product's
  design tokens.

### Changed
- `picode-desktop` gained a `clean` subcommand that passes `picode clean`
  through to the distro, streaming its progress.
