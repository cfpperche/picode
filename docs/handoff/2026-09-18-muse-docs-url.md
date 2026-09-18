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
- The transient "detection correct but lifecycle all-false" state seen once
  today (self-healed after a check/resolve) is unexplained — watch for menus
  vanishing after a service restart.
