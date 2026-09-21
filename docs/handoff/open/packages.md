# Packages and manifests

## Next

- Package config descriptors (ADR-0119; C0–C5 shipped). Backlog (owner):
  upstream `picode.config` proposal. Plan: `docs/plans/package-config-manifest.md`.
- Guest packages: nothing queued — what is left is under `## Debts`.

## Debts

- [ ] **No Update action for a guest CLI's plugins.** ADR-0167 ships install,
  remove and enable/disable, and leaves `update` to the vendor's own command:
  an honest "update available" badge needs the vendor's catalog compared per
  CLI (Hermes' curated catalog, Muse's marketplace snapshot, Codex's remote
  marketplace), and PiCode has that machinery only for Pi's packages. The row
  menu names the command (`hermes plugins update`, `muse plugins update`,
  `claude plugin update`, `omp plugin upgrade`, `grok plugin update`).
  Whoever builds the availability signal promotes the action in
  `internal/clipkgs/specs.go` — the argv builders already exist for Claude
  Code, Grok, Hermes, Muse and Omp.
- [x] **A consent refusal carried no command.** Paid 2026-09-21
  (`feat/pkgs-polish`): `clipkgs.Command`/`MarketCommand` render the exact line
  from the same builder the request executes (with URL credentials redacted),
  every synchronous refusal returns it, a failed package job carries it too
  (`GET /api/cli-jobs` and the 202 answer), and both panes show **Copy command**
  beside the vendor's words with "Run it in a terminal." — Grok's `--trust` and
  Claude's marketplace-declared command are the cases it exists for.
- [x] **Reads and the OpenCode splice disagreed about a `plugin` string.**
  Paid 2026-09-21 by measuring the vendor instead of guessing: OpenCode refuses
  that document itself — running its own `opencode plugin <module>` against
  `"plugin": "is-odd"` prints *"Configuration is invalid … Expected array |
  undefined"* and writes nothing. The read now fails loudly naming the file and
  the form the vendor expects (`feat/pkgs-polish`), instead of listing a module
  the CLI never loads and then refusing to remove it.
- [ ] **Muse Code's built-in first-party plugins are outside every surface
  PiCode may read.** Its TUI `/plugins` panel lists one (`TBH Reminders`,
  `built-in`/`product`) that no CLI command reports — not `plugins list
  --json`, not `--available`, not the store's own `installed.json`; the
  bundles live under `~/.local/share/muse/plugins/cache/builtin/`. PiCode
  therefore manages what the vendor's store manages and shows nothing it
  cannot change. Promoting these would mean reading the app's internal bundle
  and offering rows that cannot be removed — a vendor-side change, not a
  PiCode one.

- [x] **Four vendors' non-empty plugin-list JSON was read tolerantly.** Paid
  2026-09-21: every guest is now measured non-empty against the real binary —
  Grok, Omp, Hermes, Claude Code, Codex, Antigravity, OpenCode and Muse — with
  fixtures in `internal/clipkgs/testdata/` and the gated live harness
  (`PICODE_PKGS_LIVE=1`). The tolerant reader stays only as a fallback that
  refuses a shape it does not recognize.
- [x] **Muse Code's plugin roster was read tolerantly.** Paid 2026-09-21
  (`parseMuse`/`museRecordRow`, fixture `testdata/muse.list.json`, live harness
  installs a bundle and reads the row back). The earlier "its build refuses
  every plugin verb" reading was the vendor's per-machine feature gate measured
  with a fresh HOME — the harness now seeds that config from the machine's own
  cache.
