### Added
- **Agent CLIs: `/login` opens where you are looking.** A CLI's "open in the
  browser" (OAuth logins) no longer starts a Chromium inside WSL: the page
  opens in the desktop app's own browser tab, in a tab of the browser you
  run PiCode in, or — with no client open — in the Windows default browser.
  CLI launch PATHs carry the PiCode wrappers too, so CLIs that resolve
  `xdg-open`/`wslview` by PATH (not `$BROWSER`) are covered.
  Off switch on Agent CLIs ("browser hand-off").
