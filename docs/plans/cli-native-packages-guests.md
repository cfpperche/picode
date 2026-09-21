# Native packages for the eight guest CLIs

Status: **approved and implemented on `feat/cli-pkgs-guests`, 2026-09-20.**
Slices 1–3 and 5–6 shipped (ADR-0167, `internal/clipkgs`, the guest API, both
panes, docs); slice 4 (Antigravity's text roster) ships read-only-tolerant and
is pinned by the live suite where the vendor's shape can be measured. The decision it asks for is a new ADR: it
touches persistence (PiCode manages other CLIs' plugin stores), process (PiCode
runs vendor plugin commands, git and npm network work) and security model (who
consents to a plugin's code).

- **Owner request:** Pi has the Packages pane and the other invited CLIs do
  not; study the other CLIs and cover the gap — implementing only what a CLI
  really has, and saying plainly in the panel where it has nothing.
- **Receipts:** the nine CLIs installed on this machine, probed 2026-09-20
  (`<cli> plugin --help` and the subcommands quoted below; `~/.<cli>` stores
  listed); OpenCode's plugin documentation fetched the same day
  (<https://opencode.ai/docs/plugins/>); in-house: ADR-0069, ADR-0102,
  ADR-0099/0119, ADR-0150, ADR-0163, ADR-0087/0093 (the `clijob` lane),
  `docs/architecture/cli-packages.md`, `docs/plans/cli-config-and-memory.md`
  (the Settings precedent), `docs/handoff/open/connectors-parity.md` (the
  verification pattern).

---

## 1. The gap, exactly

| CLI | `#/clis/<cli>/packages` today | Source |
|---|---|---|
| pi | full pane — machine / workspace / agent scope, install/remove/update, PiCode gallery marketplace, config descriptors | `internal/pipkg`, `internal/server/packages*.go`, `Packages.jsx` |
| the other eight | *"Packages for &lt;CLI&gt; are in development — coming soon."* | `supportsCliPackages` = `["pi"]` (`web/shared/domain/cliPackages.js:2`), `CliPackages.jsx:16` |

The placeholder is not inaccurate, it is **uninformative**: six of the eight
CLIs ship a real plugin system with a machine-readable roster and their own
marketplace, one ships a plugin list that is only a config array, and one
prints prose. Two panes away, ADR-0150 already proved the shape of the fix:
one pane, one driver per CLI, a declared capability set, and the vendor binary
as the authority.

Cutover points, all in one move: `CLI_PACKAGES`/`supportsCliPackages`
(`web/shared/domain/cliPackages.js:2`), the placeholder branch in both
`CliPackages.jsx`, the Pi-only assumption in `Packages.jsx`'s list/updates
URLs, and the `supportsCliPackages("codex") === false` assertion in
`cliPackages.test.js:36` — which is a pin on the gap, not on behavior, and is
replaced by the capability table.

## 2. What each CLI actually has (probed 2026-09-20)

All verbs below are the vendors' own, copied from `--help` on this machine.

| CLI | Native surface | Roster output | Scopes | Store |
|---|---|---|---|---|
| **Pi** | `pi install <src> [-l]`, `pi remove`, `pi list`, `pi config` (TUI on/off), `pi update` | PiCode API (`internal/pipkg` over `settings.json` + package dirs) | machine, workspace (`-l`), **agent** (`-e`, PiCode-only) | `~/.pi/agent`, `<ws>/.pi` |
| **Claude Code** | `claude plugin {list,install,uninstall,enable,disable,update,details,prune,init,tag,eval,marketplace}` | `plugin list --json` ✔ (`id`, `version`, `scope`, `enabled`, `installPath`, `installedAt`); `--available` needs `--json` | `user`, `project`, `local` (`install -s`); plus vendor-owned `synced` (claude.ai) | `~/.claude/plugins/{installed_plugins.json,known_marketplaces.json,marketplaces,cache,synced}` |
| **Codex** | `codex plugin {add,remove,list,marketplace}` | `plugin list --json` ✔ (+ `--available`, `-m <marketplace>`); rows carry `pluginId`, `marketplaceName`, `enabled`, `source`, `installPolicy`, `authPolicy` | machine only (config.toml; no scope flag observed) | `~/.codex/config.toml` (`[plugins."github@openai-curated"]` at line 466) + cache |
| **Grok** | `grok plugin {list,install,uninstall,update,enable,disable,details,validate,tag,marketplace}` | `plugin list --json` ✔ (+ `--available`) | machine only (no scope flag) | `~/.grok/{installed-plugins,marketplace-cache}` |
| **Hermes** | `hermes plugins {list,install,search,browse,update,remove,enable,disable,capabilities,doctor,compat,validate,pack}` | `list --json` ✔ (`name`, `status`, `version`, `description`, `source`, `removed`); curated catalog via `search`/`browse`; `pack install/export/show` pins commit SHAs | machine only | `~/.hermes/plugins/` (**contains PiCode's own `picode-native`**) |
| **OpenCode** | `opencode plugin <npm module> [-g] [-f]` writes config; **no list, no remove, no marketplace** | none — the roster is the config `plugin: []` array (global `~/.config/opencode/opencode.json(c)`, project `<ws>/opencode.json(c)`) plus files in `~/.config/opencode/plugins/` and `<ws>/.opencode/plugins/` | global (`-g`) / project (default) | those two configs, the two plugin dirs, `~/.cache/opencode/node_modules` |
| **Muse Code** | `muse plugins {install,list,inspect,approve,reject,enable,disable,update,remove,validate,marketplace,hook test}` | `list --json` ✔ (+ `--available`) | `user`, `project` (`install --scope`) | `~/.config/muse` |
| **Antigravity** | `agy plugin {list,import,install,uninstall,enable,disable,validate,link}` | **text only** — `list` prints `No imported plugins.`; no `--json` anywhere in its help | machine only | `~/.gemini/antigravity-cli/` |
| **Omp** | `omp plugin {install,uninstall,list,link,doctor,features,config,enable,disable,marketplace,discover,upgrade}`, `omp install` alias | `plugin list --json` ✔ (`{npm:[],marketplace:[]}`) | `user` (default), `project` | pi-shaped: npm dir under `~/.omp/agent` |

Three honest tiers fall out, and the pane must render each differently:

- **A — structured and complete (6):** claude-code, codex, grok, hermes, muse,
  omp. Roster, install, remove, toggle, and a vendor marketplace all exist
  with `--json`.
- **B — a config array, nothing else (1):** opencode. No list, no remove, no
  marketplace. Its roster is readable from the two config files and the two
  plugin directories; removal is an edit of its own `plugin` array.
- **C — prose output (1):** agy. Verbs exist (`install`, `uninstall`, `enable`,
  `disable`) but the only roster is a human sentence; `import` copies plugins
  from Gemini or Claude.

**No CLI among the nine has "nothing".** The honesty line therefore belongs to
*capabilities*, not to CLIs: opencode has no marketplace, agy has no
machine-readable roster, nobody but Pi has a per-agent scope. Each of those is
one sentence in the pane where the control would be — never a disabled button
and never a fabricated list.

Depth is also uneven in ways the pane must not flatten: Claude Code's
`plugin install` can require confirming a marketplace-declared command
(`--accept-command <sha256>`, `-y`), Grok's install takes `--trust`, Hermes
installs `--no-enable` by default for portable packages and gates plugin code
on capability consent, Muse has `approve`/`reject` over runtime capabilities.
Those are the vendors' own trust steps. **PiCode runs the vendor command
without its auto-consent flag** and shows the vendor's refusal verbatim, the
way ADR-0150 decision 2 delegates OAuth sign-in to the vendor in a terminal.

## 3. What Pi's pane has that guests must not be given

| Pi-only thing | Why not invented for a guest |
|---|---|
| **Per-agent scope** (`-e` at launch, `agent` radio) | No guest CLI has a per-terminal plugin layer. `agent` is refused with 400 and the radio is absent. |
| **Configure** (ADR-0099/0119 config descriptors) | Descriptors describe a **Pi package** manifest. A guest plugin's configuration belongs to the vendor (`claude plugin install --config key=value`); no generic editor for a schema PiCode does not have (ADR-0099 §5). |
| **The PiCode gallery as the Marketplace subtab** | Pi's marketplace is `internal/pipkg.SearchGallery`. A guest's marketplace is the **vendor's** (`--available`, `hermes plugins search`, `muse marketplace`). One shared cross-CLI catalog would invent common package semantics — the alternative ADR-0102 already refused. |
| **Write into a `synced` (claude.ai) plugin** | Pushed by the vendor's web service; PiCode lists it and refuses mutation with the reason. |
| **Treat PiCode's own Hermes plugin as a user package** | `internal/server/native_integration.go` installs `~/.hermes/plugins/picode-native` for conversation identity. It is marked *Installed by PiCode*; removal warns that activity reporting for Hermes stops. |
| **A PiCode package database or cross-CLI transfer** | ADR-0102/0150: the vendor's own store stays authoritative; PiCode keeps no registry of its own. |

## 4. The design

The ADR-0150 pattern, applied to packages: **one pane, one interface, a driver
per CLI**, Pi keeping `internal/pipkg` behind the same interface.

```go
// internal/clipkgs
type Driver interface {
    ID() string                       // catalog id (claude-code)
    Bin() string                      // vendor binary (claude)
    Caps() Caps                       // what this CLI really has
    List(ctx, Paths) (Report, error)  // normalized rows
    Install(ctx, Paths, Target, Scope) error
    Remove(ctx, Paths, Target) error
    Toggle(ctx, Paths, Target, on bool) error
    Update(ctx, Paths, Target) error
    Available(ctx, Paths, Query) ([]Row, error)   // vendor marketplace only when Caps().Marketplace
}
```

`Caps` is what the pane reads; a control exists only where the CLI declares the
capability. Normalized row: `{id, name, version, scope, enabled, source,
sourceKind, installPath, updateAvailable, managedByPiCode, raw}`.

Load-bearing choices:

1. **Roster**: the vendor's `--json` where it exists (six CLIs). OpenCode reads
   its own configs and plugin dirs through the existing codec family
   (`internal/connectors/codec.go`, JSONC-tolerant reads). agy parses text as
   tier C (§8). A driver that cannot parse **fails loudly** — never an empty
   roster presented as "none installed" (the `connectors-parity` rule).
2. **Writes are the vendor command**, run with `cmd.Dir` = the workspace for a
   project-scope action and the daemon's own environment otherwise. The single
   exception is OpenCode, whose only verb for removal and toggling is its own
   `plugin` array in `opencode.json(c)` — a string-array splice added to the
   ADR-0163 writer (insert/remove one element, unknown keys, comments and
   indentation preserved, atomic, re-read-before-write, 409 on an older
   revision).
3. **Jobs, not request handlers.** An install is a durable
   `internal/clijob` job: HTTP 202, request-key idempotency, one active job per
   CLI, bounded output tail, `cli.job` events — the ADR-0087/0093 lane, not a
   second one. The existing rule that a job refuses while that CLI's terminals
   are live (unless confirmed) matters more here than for an update: the plugin
   lands in the CLI's **next** start.
4. **Trust is the vendor's.** PiCode never passes `-y`, `--trust`,
   `--accept-hooks`, `--enable` or an `--accept-command` hash on the user's
   behalf. A vendor refusal is shown verbatim with a copyable command for a
   terminal, and the pane re-reads after the user acts.
5. **Events**: mutations publish an ephemeral `cli.packages` invalidation
   (`cli.settings`/`cli.memory`/`mcp.config` are the precedents). The plugin
   store stays authoritative; no API response carries a secret, an install
   path beyond what the vendor prints, or an environment value.
6. **Layers**: `user` and `project` where the vendor declares them (claude also
   `local`, its own third layer, labelled as such); `agent` refused by name
   before any path resolution — the ADR-0163 rule that a link carrying
   `?scope=agent` must not silently edit a machine file.

## 5. The pane, per capability

| Element | pi | claude-code | codex | grok | hermes | opencode | muse | agy | omp |
|---|---|---|---|---|---|---|---|---|---|
| Installed roster | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ (config+dirs) | ✔ | ✔ (text) | ✔ |
| Install | ✔ | ✔ | ✔ | ✔ | ✔ (catalog/git) | ✔ (npm module) | ✔ | ✔ | ✔ |
| Remove | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ (array splice) | ✔ | ✔ | ✔ |
| Enable/disable | ✔ (`pi config`) | ✔ | — (vendor has none) | ✔ | ✔ | ✔ (array: present/absent) | ✔ | ✔ | ✔ |
| Update | ✔ | ✔ | — | ✔ | ✔ | — | ✔ | — | ✔ (`upgrade`) |
| Marketplace subtab | PiCode gallery | `--available` + `marketplace` | `--available` + `marketplace` | `--available` + `marketplace` | `search`/`browse` | **none — one line** | `--available` + `marketplace` | `link`/`import` | `marketplace`/`discover` |
| Extra | descriptors | `details`, `prune`, `eval` | — | `details`, `validate` | `capabilities`, `doctor`, `pack` | — | `inspect`, `approve`/`reject`, `hook test` | `import` | `doctor`, `features` |

The blank cells are copy, not controls: *"Codex plugins cannot be disabled —
`codex plugin remove` and `add` are the only verbs its CLI exposes."* The
Marketplace tab is hidden where the CLI has none, with the reason in the
Installed tab's header line.

## 6. Decision table (the rows tests must cover)

| Conditions | Action |
|---|---|
| CLI declares the verb (install/remove/toggle/update/marketplace) | Control present, wired to the vendor command |
| CLI does not declare the verb | Control absent; one line names what the CLI exposes instead |
| Scope `agent` requested for any guest | 400 before path resolution; radio absent |
| Scope `project` requested, no workspace resolved | Blocked, one line + one action (never a silent fallback to machine) |
| Scope `project` requested, CLI has no project layer | Refused by name |
| Claude `synced` plugin | Listed; `disable`/`uninstall` refused, reason shown |
| Hermes `picode-native` row | Marked *Installed by PiCode*; removal warns activity reporting stops |
| Vendor prints a consent refusal (claude command hash, grok `--trust`, muse capability) | Refusal verbatim + copyable command; nothing retried with an auto-consent flag |
| Roster read fails / vendor output unparsable | Loud blocked state with the vendor's text and Retry; never an empty list |
| Vendor offline (codex remote marketplace, git install) | Error text from the vendor; last known roster retained when one was read |
| Install requested while that CLI's terminals are live | Confirmed first (the `clijob` rule), then the job |
| Install/remove job running | 202 + progress; a second job for that CLI refused |
| Plugin config file changed under us (opencode splice) | 409 + re-read; never a blind write |
| Install succeeds | Re-read the roster from the vendor; success only after that read |

## 7. Slices

| # | Slice | Gate |
|---|---|---|
| 1 | ADR (boundary: persistence / process / security-model; amends ADR-0069's "no package managers" clause for bounded vendor-command delegation, extends ADR-0102 and the ADR-0163 driver family); `internal/clipkgs` interface + `Caps`; catalog conformance test keeping the Go specs and the JS capability list in step; sandbox-HOME fixture harness | interface + conformance tests |
| 2 | Tier A drivers, simplest first: **grok, hermes, omp, muse**, then **claude-code** (three scopes, consent refusal), then **codex** (remote marketplace + offline) — each verified against the installed binary (`internal/connectors/live_test.go` pattern: sandbox HOME, vendor CLI as oracle) | roster/install/remove/toggle per driver; §6 rows |
| 3 | **opencode**: roster from configs + dirs, array splice for remove/toggle, no marketplace | splice golden tests, JSONC reads, §6 splice rows |
| 4 | **agy (tier C)**: text roster, or — if the installed-case format cannot be pinned (see §8) — a read-only list with the vendor's own `install` command copyable | text fixtures from a real install, or the explicit downgrade |
| 5 | Pane in both apps: capability-driven controls, per-capability copy, honest empty/blocked/error states, `clijob` progress; desktop + mobile | decision-table rows + visual review (`.pi/skills/visual-review`) |
| 6 | Docs: `docs/architecture/cli-packages.md` (guests section), changelog fragment, handoff note, `docs-site/guide/packages.md` | `make ci-scoped`; docs gates |

No new Go dependency is expected: tier A/B are JSON and the vendors' own
commands; only tier C needs a parser, and it is a line format.

## 8. Risks and what must be verified live

- **agy's installed-case output is unobserved.** No plugins are installed here,
  so `agy plugin list` was only seen empty and has no `--json`. Either pin the
  format with a real plugin in a sandbox HOME, or ship agy read-only and say
  so. Do not write a parser from a single empty-output sample.
- **`omp plugin marketplace list --json` printed prose**, not JSON, for the
  empty case — flag position or an unimplemented path. Non-empty shapes for
  omp must be measured before the driver's parser is trusted.
- **Vendors ship weekly** (omp 18.2.6, muse, agy all self-update). Text and
  `--json` shapes both move; every driver needs one live test that fails when
  the vendor's output changes, and unparsable output is a loud failure.
- **Codex remote marketplaces need auth and network** (`openai-curated-remote`
  rows). Its offline/unauthenticated window must be an honest blocked state,
  never an empty marketplace.
- **PiCode's own integration rows** (hermes `picode-native`; PiCode's injected
  opencode/omp launch plugins are *not* in those vendors' stores) must be
  identified from `native_integration.go`'s receipt rather than by name
  matching alone.
- **A plugin is code.** PiCode shows what the vendor says about a plugin and
  installs what the user asks; it never pre-approves capability consent and
  never installs anything on its own initiative (no auto-install, no auto-update).

## 9a. What shipped against §5/§6 (2026-09-20)

Three deltas between the plan and the branch, each with a reason:

| Plan | Shipped | Why |
|---|---|---|
| §6 "refusal verbatim + copyable command" | the verbatim half only | the refusal reaches the pane and the user retypes the command; named as a debt in `docs/handoff/open/packages.md` |
| §5 blank cells as copy | shipped, and every concept-keyed note (Muse's capability gate, OpenCode's local files) renders once above the tabs | a CLI-wide fact repeated per row is noise, and the note must be visible where the tab is absent |
| §2 "Antigravity prints prose" | Antigravity prints a JSON envelope once plugins exist | measured by the live harness (`agy plugin list`, 1.2.7); the prose sentence is only the empty answer |

Also measured after the plan: OpenCode's own `opencode plugin <module>` writes
`<cwd>/.opencode/opencode.json` (not the documented `<cwd>/opencode.json`), so
the roster and the splice read that location too, and the divergence is filed
against `internal/connectors` in `docs/handoff/open/connectors-parity.md`.

## 9. Open questions for the owner (with the recommendation)

1. **Breadth of v1 — recommend all eight, slices ordered A → B → C, and agy
   allowed to degrade rather than block.** The pane's value is one surface for
   every CLI the product launches (ADR-0150 shipped all nine at once, for the
   same reason), and the placeholder we are removing is exactly the "some CLIs
   only" shape. Order matters more than scope: six tier-A drivers are the same
   code path over `--json`; opencode needs a new writer and agy an unobserved
   text format, so neither is allowed to hold the other seven. If agy's
   installed-case output cannot be pinned, agy ships read-only with the
   vendor's own `install` command copyable — a stated outcome, not a slip.
2. **Verb depth — full verbs where the vendor has them, minus *update* in
   v1.** Install, remove and enable/disable are the vendor's own commands and
   the reason the pane exists. `update` is the one to defer: its button is
   cheap, but an honest "update available" badge needs the vendor's catalog
   compared per CLI (hermes catalog, muse snapshot, codex remote marketplace)
   — and PiCode has that machinery only for Pi packages today. Shipping the
   action without the signal is a blind button; the row menu names the vendor
   command instead, and availability becomes its own slice once the badge can
   be true. Named as debt in `docs/handoff/open/packages.md`.
3. **OpenCode's array splice — recommend implementing it.** Refusing list
   edits would leave the pane useless exactly where the user asked for parity:
   OpenCode's plugins *are* a config array. The risk is bounded and known — it
   is the user's own file, the codec family already parses it (JSONC-tolerant
   reads, strict writes), and the vendor only reads that array (npm install
   happens at startup into `~/.cache/opencode`) — so the splice is a
   string-array insert/remove with comments, order and unknown keys preserved,
   atomic, 409 on an older revision.
4. **Marketplace — recommend the vendor's own available list, per CLI.**
   ADR-0102 already refused one generic package backend, and the vendor's
   catalog is the only source that carries the truth the pane has to show
   (Codex rows alone carry `installPolicy` and `authPolicy`; Claude's install
   can require a marketplace-declared command's hash). A PiCode-curated
   cross-CLI catalog would mean curating and vouching for other vendors'
   plugin code — a trust decision PiCode does not own. OpenCode gets no tab
   and one line; agy gets the tab only if its roster parses.

## 10. Frozen contract (implementation, 2026-09-20)

`internal/clipkgs` declares one `spec` per CLI: scopes, a roster reader, argv
builders per verb (nil = the CLI does not expose it), marketplace builders,
and the note shown where a verb is absent. `Caps` is derived from the
declaration, never restated. Pi is not in the catalog: `For("pi")` is nil and
its pane keeps its own API.

HTTP (`internal/server/cli_packages.go`):

| Route | Body / query | Answer |
|---|---|---|
| `GET /api/cli-packages` | `cli`, `workspace`, `scope`, `refresh=1` | 200 `{cli, scopes[{id,label,note}], caps{}, notes{}, rows[], note, readAt}`; 400 for an unknown CLI, `scope=agent`, or a project scope without a workspace |
| `GET /api/cli-packages/available` | `cli`, `workspace`, `scope` | 200 `{cli, rows[], note}`; 400 where the CLI has no marketplace PiCode can list |
| `GET /api/cli-packages/marketplaces` | `cli`, `workspace`, `scope` | 200 `{cli, marketplaces[Row]}`; 400 where the CLI manages no sources |
| `POST /api/cli-packages/install` | `{cli, workspace, scope, source, requestKey?}` | 202 — the job row itself, the same shape the ADR-0087 lifecycle route returns |
| `POST /api/cli-packages/remove` | `{cli, workspace, scope, name, source, requestKey?}` | 202 — the job row |
| `POST /api/cli-packages/update` | `{cli, workspace, scope, name, requestKey?}` | 202 — the job row |
| `POST /api/cli-packages/toggle` | `{cli, workspace, scope, name, source, on}` | 200 — the whole view with the fresh rows (enable/disable is a fast vendor call, not a job) |
| `POST /api/cli-packages/marketplace` | `{cli, workspace, scope, action, source, name, ref, requestKey?}` | 202 — the job row for `add`/`update`; 200 `{cli, marketplaces[]}` for `remove` |
| `POST /api/cli-packages/inspect` | `{cli, target}` | 200 `{cli, output}` — the vendor's own text, only where `caps.inspect` |

The shape of an answer is the one the pane found in the shipped server, not the
first sketch: a job answer is the job row I (the lifecycle route's own answer),
a toggle answer is the full view, and a source removal answers the CLI's own
list. The pane adapts to what the lane already answered — a second envelope
would only be a second shape to keep in step.

Jobs carry their own payload: `store.CLIJob.Payload` is one JSON string column
in the existing job document (no SQL migration — the row is stored as JSON),
and `clijob.Start` takes it so `Resolve(cli, action, payload)` can build the
vendor argv after a restart. Actions: `pkg-install`, `pkg-remove`,
`pkg-update`, `pkg-marketplace-add`, `pkg-marketplace-update`. The lane's
existing guards apply unchanged: one active job per CLI, `ErrTerminalsRunning`
unless confirmed, 202 + request-key idempotency, `interrupted` (never replayed)
after a restart. Every mutation publishes the ephemeral `cli.packages` event
and invalidates that CLI's roster cache.

Shared domain (`web/shared/domain/cliPackages.js`): `CLI_PACKAGES` keeps Pi and
gains `GUEST_PACKAGES` (the eight ids), with a Go conformance test keeping the
two lists in step, the way `cliNative.js` and `internal/clisettings` already
do.
