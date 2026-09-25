### Removed

- **Windows: the Go tray migration.** `picode-desktop` no longer accepts
  `--tray` or `startup-repair --retarget-shell`; startup repair works on the
  shell resident only. Your startup task already starts the shell.
- **Old-version fallbacks in the desktop shell and server.** The Disk tab no
  longer retries without streaming, browser automation calls always carry
  their domain list, and a Pi terminal still running a receiver from before
  it reported its wrapper no longer recovers its runtime until it restarts.
- **Leftovers:** the `cmd/uicheck` debug tool and the cleanup of a dashboard
  preference retired long ago.
