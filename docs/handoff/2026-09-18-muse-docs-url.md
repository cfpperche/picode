# 2026-09-18 — muse-docs-url

Follow-up to the lifecycle investigation: Meta moved the Muse Code docs to
dev.meta.ai and the old host now answers 500.

## Done
- Catalog row and lifecycle docs links point to https://dev.meta.ai/docs/muse-code
  (config.go, lifecycle.go). Tests green.
- Vendor facts re-confirmed on the new docs: install is a one-line installer
  (`dev.meta.ai/install.sh`) putting a native `muse` on PATH; no update or
  uninstall subcommand documented — the launcher-as-updater plan and the
  guided uninstall stay right. Note: this machine still runs the older
  launcher-script shape (api.meta.ai URLs inside), which the classifier
  already covers.

## Debts
- [x] The transient "lifecycle all-false" state: explained and fixed in
  feat/lifecycle-flip — a vendor self-update rewrites its launcher mid-swap,
  one request's Stat read it as absent, and the not-installed branch strips
  the lifecycle (no npm package for these CLIs). The resolver now retries
  once before declaring a CLI absent.
