# Packages and manifests

## Next

- Package config descriptors (ADR-0119; C0–C5 shipped). Backlog (owner):
  upstream `picode.config` proposal. Plan: `docs/plans/package-config-manifest.md`.
- Guest packages: nothing queued — what is left is under `## Debts`.

## Debts

- [x] **The update action shipped with its signal.** Paid 2026-09-21
  (`feat/pkg-update-badge`, ADR-0167 amendment): `GET
  /api/cli-packages/updates` compares the CLI's roster with its own catalog
  (`clipkgs.CheckUpdates`, `pipkg.Newer`), the pane runs it once per mount where
  the CLI has the verb, and Update appears only on the rows the catalog says
  are behind (`→ 0.3.0`). Measured end to end in the live harness: Muse
  installed at 0.2.0, the marketplace snapshot refreshed with the vendor's own
  command to 0.3.0, the row came back `BEHIND latest=0.3.0`. A catalog that
  cannot be read leaves rows unmarked with the reason; a version pair the
  comparator cannot parse is not a claim. Omp gets the badge without an Install
  because `discover` carries versions.
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
- [ ] **`GET /api/packages/report` still drops the isolation switch.** The badge
  half of this debt was paid 2026-09-21 (`feat/packages-guests`, slice 2): the
  four guest reads answer from `pkgs.DriverFor(cli)` — a pure mapper
  (`internal/pkgs/guest_view.go`) reproduces the pane's bytes, proven over
  fixtures and over the real routes — and every guest driver now maps the
  vendor's whole capability set, `Caps.Update` included. So
  `/api/packages/updates?cli=<guest>` answers the vendor's own check where the
  CLI has the update verb instead of 400. What remains: `GET
  /api/packages/report` does not pass `Query.AgentIsolated`, so "only this
  agent's packages" is still visible only through the legacy route. Closes when
  one pane renders the unified shape (slice 3).
  Plan: `docs/plans/packages-unification.md`.

- [ ] **Omp's own `extensions` list is invisible in the guest packages pane.**
  Measured 2026-09-21 while giving Omp the browser tool: the pane renders the
  vendor's plugin roster (`omp plugin list --json`), but the extension loads
  from a project `extensions` entry — `<ws>/.omp/settings.json` (ours) or
  `.omp/config.yml`; user level is `~/.omp/agent/config.yml`, where
  `omp config set extensions '<json array>'` writes. `omp config get extensions
  --json` returns the value, so the pane can show it: read the CLI's own key
  beside the roster and render one row per entry (LOCAL badge + path), the way
  the Pi pane shows `.pi/settings.json` path packages. `omp plugin link <path>`
  is *not* the answer — for a local package its scope is the machine
  (`--scope=project` wrote nothing into the project; the flag is marketplace
  installs only), which breaks the workspace-scope model.

- [ ] **OpenCode's removal is unreachable from the pane.** Its declaration says
  `Remove: true`, because OpenCode's plugin list is its own config array and
  `clipkgs.Run` writes it in process — but every removal goes to the durable job
  lane, which runs an argv, and OpenCode's removal has none. So
  `POST /api/cli-packages/remove` answers 400 *"this CLI does not expose that
  operation: opencode remove"* while the pane draws the button (`Caps.Remove`
  is true), and no route removes an OpenCode plugin. Measured 2026-09-21
  (`feat/packages-mut`, slice 2b: the driver answers the lane's verbs with the
  engine's own refusal, byte-identical to the handlers it replaced). Closes when
  a mutation can be an in-process write on the lane, or OpenCode's removal moves
  to the synchronous path its toggle already uses.
