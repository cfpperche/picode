# 2026-09-21 — feat/packages-omp: Omp's own extensions are rows (ADR-0176, slice 5)
Shipped: the omp roster read merges the CLI's own `extensions` beside its plugin store
(`internal/clipkgs/omp_extensions.go`) — `<ws>/.omp/settings.json` parsed with the standard
library, the user level through `omp config get <key> --json`, never a YAML parser for
`~/.omp/agent/config.yml`. One row per entry: the entry as written, the CLI's own name for it,
the resolved path, `Scope` machine or workspace, `Enabled` from `disabledExtensions`. A row
the plugin roster carries is not duplicated. Removal writes what the row came from: the
workspace entry is spliced out of that file (entry and its `extension-module:<name>` id, every
other byte preserved), the user layer through `omp config set extensions`; a mutation the
driver runs itself has no argv, so `/api/cli-packages/remove` answers the CLI's fresh list
instead of reserving a job that would run nothing. `web/` diff empty.
Verified: `make ci-scoped` PASS (fmt,vet,hooks,go[6]; 11 path(s) vs main); fixture tests for
the file (present, absent, empty, disabled, non-list, malformed), the vendor answer, the row
shape, the roster overlap, both removals, plus a route test over a store workspace. Live on
this machine, an isolated `picode` (temp `PICODE_DATA`, port 8477): the read at the workspace
scope answers the `browser` row with its resolved path, and a removal in a temp workspace
answers 200 + the fresh list, leaving the settings file with the other entry, the other key
and no stale disabled id. Not run: `PICODE_PKGS_LIVE=1`; no write to the real `~/.omp`.
visual-review: UNVERIFIED — no browser pass; the row, the route answer and the file after a
removal are what was observed. Merge: fast-forward ready.

## Debts

- An extension row draws Enable/Disable and Omp's disable verb does not know it. Owned by
  `docs/handoff/open/packages.md`.
