# 2026-09-15 — browser-downloads

Owner review, next section: Downloads (Location / Ask / Download history),
dialogs instead of sub-routes.

## Done

- Shell: `btab_download_dir` / `btab_set_download_dir` (profile
  DefaultDownloadFolderPath, empty = system), `btab_set_ask_download`, and
  a `DownloadStarting` handler on every tab (_4 interface) that silences
  the runtime UI unless asked, then emits `btab://download` on start and on
  outcome (StateChanged → completed/interrupted). `btab_open_path` /
  `btab_reveal_path` for the row menu (absolute paths only).
- Go: migration 049 + store (upsert by path, search, delete, clear) with a
  table test and four rows in the events invariant; the five endpoints,
  tested; `browser.askDownload` pref with a round-trip assertion.
- UI: the Downloads section in the reference shape; the folder dialog; the
  Download history dialog (search, Clear all, empty state read against the
  reference's); the app records the shell's reports through the API.
- Visual: section, populated dialog, empty state and folder dialog read;
  overlay audit ok in all four.

## Notes

- The reference's "Show a save dialog": the platform offers the runtime's
  own download UI, not ours — with Ask on, that is what appears. Said so in
  the row copy ("Show a save prompt").
- Untested on Windows: the actual DownloadStarting path (needs a real
  download in the desktop app) — first thing to check after the swap.

## Next up

- Browser permissions (Site settings camera/mic, History, Enable site
  tools), then Developer mode.
