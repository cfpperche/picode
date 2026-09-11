### Added
- **PiCode Desktop: give the held space back from the tray.** When the disk
  file is holding space the distro freed, the tray offers **Give back ≈92
  GB…**. It first asks PiCode whether anyone is working (the same interlock as
  a deploy — someone mid-turn is named, and it stops), then asks once in a
  dialog that names the cost, stops Ubuntu, converts the disk file to
  **sparse**, starts Ubuntu again and reports the before/after measured on the
  file. From then on WSL returns freed blocks on its own.
- **`picode-desktop disk-compact`** — the same flow from a terminal.
  `--dry-run` prints the plan and stops nothing; `--yes` runs it; `--force`
  overrides the working check; `--method optimize-vhd` compacts with Hyper-V's
  Optimize-VHD from an administrator terminal (Windows Home: upgrade WSL for
  the sparse path instead).

### Changed
- **A failed compact no longer leaves the machine without its distro.** The
  restart after `wsl --terminate` runs even when the conversion fails, and the
  tray re-arms the keepalive child the terminate killed — without it, WSL
  idles out sixty seconds after a compact.
