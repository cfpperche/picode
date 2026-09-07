# 2026-09-06 — feat/opencode-cli: OpenCode terminals (PR1)

PR1 in flight: catalog id `opencode`; sessions from
`~/.local/share/opencode/opencode.db` (or `$XDG_DATA_HOME/opencode/opencode.db`)
read-only; skip children/archived/empty; timestamps in milliseconds; resume
`opencode --session <id>` (OpenCode 1.18.29). Presence wrapper only — no
`OPENCODE_CONFIG`, no data-dir overlay, no write to `~/.config/opencode`.
Maintenance subcommands (`session`, `auth`, `run`, …) skip the lease; a
path positional is TUI. Lifecycle: `opencode upgrade` /
`opencode uninstall --keep-config --keep-data --force` when DetectMethod
returns npm (bun global lives under node_modules).

PR2 (not this commit): plugin via `OPENCODE_CONFIG` merge, session.status /
permission.asked map.

Verified this session: `make test`, `make test-js`, `make fmt-check`,
`make vet`. Visual-review PASS on scratch :8471 (catalog OpenCode row +
Sessions empty “No OpenCode sessions yet”; overlayAudit ok).
