# Packages (ADR-0102, ADR-0167, ADR-0176)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per
> subsystem). Edit here; the index only links.

Packages is a pane of the selected CLI (`#/clis/<cli>/packages`) in each app,
and — since ADR-0176 — **one engine answers for every CLI**. `internal/pkgs`
holds the model, the interface and the registry; a **driver per CLI** sits over
the engine that keeps that vendor's facts (`internal/pipkg` for Pi,
`internal/clipkgs` for the eight guest CLIs, both untouched). Nothing in
`internal/pkgs` knows a vendor's argv or its roster shape, and nothing in a
surface branches on which CLI it is: a pane asks for the report, and the report
— scopes, capabilities, catalog, notes — decides what is drawn.

## The model and the interface

`Scope` is the caller's vocabulary, not the vendor's: `machine`, `workspace`,
`agent`, each carrying the CLI's own word in `ScopeRow.Vendor` (`user`,
`project`, `local`) so a pane can still say what the CLI says.

```go
type Scope string // machine | workspace | agent

type ScopeRow struct { ID Scope; Vendor, Label, Note string }
type Catalog string  // "" (none) | gallery (PiCode's npm search) | vendor

type Caps struct {
    List, Available, Install, Remove, Update, Toggle, Inspect bool
    Config, Marketplace                                       bool
    CatalogInstall bool // the CLI's catalog names the spec an install takes
    Lane           Transport // per verb: a vendor command, or PiCode's own write
    IsolatedSwitch bool // the agent row's "only this agent's packages" flag
}

// Transport is how the mutations the job lane can carry reach the CLI: true is
// a vendor command, false a write PiCode performs itself.
type Transport struct{ Install, Remove, Update, Marketplace bool }

type Row struct {
    CLI, ID, Name, Source, Kind string
    Scope                       Scope
    Vendor                      string
    Enabled, Installed          bool
    Version, InstalledPath      string
    ConfigKind, Status          string
    Marketplace, Behind         string
    Description, Note           string
    ManagedByPiCode             bool
}

type Report struct {
    CLI                    string
    Scopes                 []ScopeRow
    Rows                   []Row
    Caps                   Caps
    Catalog                Catalog
    Gallery                string
    Notes                  map[string]string
    Note, ReadAt, CheckedAt string
    Capabilities           map[string]bool
    WorkspacePath, WorkspaceName, AgentName string
    Isolated               bool
}
```

`Driver` is the whole surface: `ID`, `Scopes`, `Caps`, and the reads — `List`,
`Available`, `Marketplaces`, `CheckUpdates` — plus the mutation verbs
`Install`, `Remove`, `Update`, `Toggle`, `Inspect`, `Marketplace`. `DriverFor(cli)`
resolves `""` and `pi` to the Pi driver and every other id to a guest driver;
`Known` answers whether an id has a driver at all, and `CLIs()` lists them, Pi
first. A verb a driver's CLI does not expose refuses with the same fact its
`Caps` declares (`ErrNoMutation` for the verbs Pi's driver does not carry,
`ErrNoUpdateCheck`, `ErrNoCatalog`, `ErrNoMarketplaces`).

`Query` is one read and carries everything the engine may not fetch itself: the
scope class and the CLI's own word for it, the workspace path and name, the
agent's name, and — because the engine never reaches the store — the agent's
own entry list (`AgentSources`) and its isolation flag (`AgentIsolated`).
`Command` is one mutation as it can be run: the vendor executable, its argv, the
directory it runs in, and the exact line a person would type. Both halves come
from one builder, so the line a pane copies is a command PiCode itself would
run. A mutation a CLI only offers through its own config file has no argv at
all — `Exe` and `Line` are then empty and the verb runs in process.

**Pi is a driver like any other, and keeps its own mutations.** Its driver reads
`~/.pi/agent/settings.json` and `<ws>/.pi/settings.json` through `pipkg`, adds
the agent's stored list, and declares `List`, `Available`, `Install`, `Remove`,
`Update`, `Config`, `CatalogInstall` and `IsolatedSwitch` — `Catalog` is
PiCode's own npm gallery (`CatalogGallery`), its scopes are `Global`,
`This workspace` and `This agent`, and `Report.Capabilities` carries the derived
`webSearch` fact. `Caps.Lane` names no verb — nothing of Pi's is reserved as a
job — and its `Install`/`Remove`/`Update`/`Toggle`/
`Inspect`/`Marketplace` on this interface refuse with `ErrNoMutation`: Pi's
mutations run through `pipkg` in the server's own handlers on the same
`POST /api/packages`, `POST /api/packages/update` and `DELETE /api/packages`
paths, told apart from a CLI's own verb by the `cli` the request carries (none,
for PiCode's own call), and they answer the CLI's fresh list directly. A read
that named no workspace or no agent declares
no radio for that layer (`scopesForContext`), so a control that could not work
is never offered.

**A guest driver classes its CLI's own declaration.** `clipkgs` still owns the
vendor's argv builders, parsers, fixtures and file writes; the driver maps them
onto the model — scopes with their vendor words (`claude-code` keeps `local`,
the uncommitted project layer), `Caps` derived from the CLI's own verb table, a
`CatalogVendor` where the CLI has an available list, the CLI's `notes` verbatim,
and `Lane` derived from that same table — a verb the CLI reaches with its own
command is the lane's, a verb whose only path is PiCode's write of its config
file is not, and the marketplace fetches are the lane's wherever the CLI manages
sources. The declaration is per verb because a CLI can be mixed: OpenCode's
install is its own `plugin` command while its removal is a splice of its own
config file, so no single bool could describe it.
`IsolatedSwitch` is declared by `omp` alone. The agent layer is PiCode's own
list, so a read that asks for it is answered from `Query.AgentSources` with no
vendor call at all — under both names a pane may use (`scope=agent` and
`vendor=agent`).

The unified answers are the model itself. Pi's legacy payload is still derived
— `internal/pkgs/legacy.go` maps a `Report` back to Pi's JSON for the routes
Pi's own pane has always parsed — while the guests' payload is now the model
directly: the mappers that reconstructed their old bytes
(`internal/pkgs/guest_view.go`) went with the alias below.

## The routes

| Route | Answers |
|---|---|
| `GET /api/packages/report?cli&scope&vendor&workspace&agent&refresh=1` | the unified `Report` for any CLI; `cli` absent or `pi` means Pi |
| `GET /api/packages/updates?cli&vendor&workspace` | the badge read: the rows the CLI's own catalog has moved ahead of (`scope` is the fallback for `vendor`) |
| `GET /api/packages/available?cli&scope&workspace` | the CLI's own installable list (`Driver.Available`), as the unified `Report`; a CLI with no catalog refuses (`ErrNoCatalog` or the vendor's own reason) |
| `GET /api/packages/marketplaces?cli&scope&workspace` | the CLI's own marketplace sources, in the unified row shape; a CLI that keeps none refuses (`ErrNoMarketplaces`) |
| `GET /api/packages?workspace&agent` | Pi's read, in Pi's JSON (`Report.Legacy`) — `cli` is not consulted |
| `POST /api/packages`, `POST /api/packages/update`, `DELETE /api/packages` | install, update and remove: Pi's own mutations through `pipkg` when the request names no CLI, the CLI's own verb (a job, or the driver's write answering the fresh report) when it names one |
| `POST /api/packages/toggle`, `/api/packages/marketplace`, `/api/packages/inspect` | the CLI's own toggle, source management and inspection — PiCode's own calls have no such verbs |
| `GET /api/packages/gallery`, `/api/packages/config`, `/api/packages/describe` | Pi's gallery search and its config descriptors (ADR-0099/0119) |

One path per verb, for every CLI. The `cli` a request carries is what resolves
its driver, and PiCode's own calls never carry one: Pi's mutations run through
`pipkg` on those same three paths, and an agent-layer write is PiCode's own list
for every CLI (`directMutation`, the pane's half of the same rule). An unknown
`cli` is a 400 naming the CLIs which have a driver; `cli=pi` on a CLI's own verb
is a 400 too — PiCode manages no plugins for Pi through that surface, because
Pi's are PiCode's own calls. The `/api/cli-packages*` family this replaced was
the alias ADR-0176 gave one release; the owner closed that window on 2026-09-22
and the handlers, the mappers and the legacy tests went together
(`feat/packages-alias`).

Failures keep their vocabulary: 400 for what PiCode refuses (a scope the CLI
does not declare, a verb it does not expose, an unreadable request), 409 for a
file that changed since the read (`ErrStale`), for the job lane's
terminals-running guard and for a CLI lifecycle conflict, 502 for what the
vendor failed to do — including an unparsable roster (`ErrRosterShape`), which
is never reported as "none installed". A refusal whose fix is a command only a
person in a terminal can answer carries that command in the body, rendered by
the same builder the request executed.

The pane's own routes and hashes are unchanged: `#/clis/<cli>/packages` and
`/config/<pkg>` for a config page. The Pi-era redirects for `#/packages*`,
`#/clis/packages*` and mobile `#/more/packages*` (ADR-0167) were retired on
2026-09-25.

## The job lane, and the verbs that answer directly

A driver's `Caps.Lane` decides the transport, one verb at a time, and that is
the only difference between the two. Pi's mutations are calls that answer the
new list, so its pane runs them, shows the transcript and re-reads the report,
and it declares no verb on the lane at all. A guest's install, removal, update
and marketplace add/update **reserve a durable job** in the lane
ADR-0087 built (`internal/clijob`) — that is what its `Lane` says true — while a
verb it leaves false is a mutation PiCode performs itself, in process, answering
the CLI's fresh list. The declaration is per verb because a CLI can be mixed
(**OpenCode**: its install is `opencode plugin <module>`, its removal a splice of
its own `opencode.json`), which a single bool could only have contradicted.
`Lane.Marketplace` covers the two source actions that fetch; a marketplace
removal is always a local change, so it is never on the lane.

For the lane's verbs the answer is 202 with the job row, the
request carries a request-key so a retry is the same job, at most one package
job is active per CLI, and the lane refuses while that CLI's terminals are
running (a plugin lands at the next start) unless the caller confirms.
`Resolve` rebuilds the argv from the job's own payload after a restart — the
driver is asked twice, once before the job is reserved and once inside the lane
— and a job that succeeds publishes the ephemeral `cli.packages` event and drops
that CLI's roster cache. The pane follows `cli.job` events and re-reads the
roster when one settles; a failed job carries the command it ran, so the copy
affordance works for the asynchronous half too. The pane's own rule is the same
declaration: `laneMutation(caps, verb)` picks the job path, and a verb off the
lane takes the flow Pi's direct mutations take — the transcript while it runs,
then the CLI's fresh list read back.

Three verbs run in the driver and answer immediately, because their answer
depends on the result: `Toggle` (the CLI's own enable/disable verb — or, for one
of Omp's own `extensions`, the `disabledExtensions` write below — then the fresh
list), `Inspect` (the vendor's own text, verbatim, with the command that
produced it) and a marketplace **removal** (a local change — the sources that
are left). Add and update fetch, so they are the lane's.

Two mutations have no argv at all and are PiCode's own writes, answered with the
CLI's fresh list instead of a job, and both are declared (`Lane` false for their
verb) rather than inferred: **OpenCode's removal**, because the CLI exposes
`opencode plugin <module> [-g]` to add and nothing to remove or disable — PiCode
deletes one element from the `plugin` array of the config file that
names it (the precedence `internal/connectors` measured, `.jsonc` first), with
comments and every other byte preserved, the result re-parsed and compared
before the atomic write, and a shape the scanner does not recognize refused
(409) rather than saved. A module the CLI's own configs do not name is refused
by name (`ErrStale`, 409), so the answer is never a silent success; and
**Omp's workspace `extensions` entry**, spliced out
of `<ws>/.omp/settings.json` by the same splice (its id leaves
`disabledExtensions` too), while the user layer goes through `omp config set`.
An agent-layer write is PiCode's own call for every CLI, which the pane's
`directMutation` states once. A driver can also be mixed per *row* — an Omp
`extensions` entry is that write while the same CLI's plugin rows are lane
commands — so a mutation answered with the CLI's fresh list is read as that write
wherever the declaration could not predict it.

**An Omp extension's toggle is that array read the other way.** The CLI's plugin
verbs cannot move a configured extension — measured 2026-09-21: `omp plugin
disable <entry>` answers *"not found in runtime config"* — so `Toggle` asks the
engine for one before it asks for the vendor's verb, exactly as `Remove` does.
The workspace layer splices the entry's id into `disabledExtensions` (out of it
when the row is being enabled, and the key itself in when the file has none),
every other byte preserved and the result re-parsed and compared; the user layer
runs the CLI's own `omp config set disabledExtensions '<json array>'` in the
user's directory, which is where that command always writes. Both directions
answer the CLI's fresh list, the line the toggle ran rides the answer on a
refusal — and the workspace write has none to ride, the same empty line the
in-process OpenCode toggle has always answered. A row the plugin store owns
keeps the vendor's own verb.

## The agent layer

PiCode's own list for one agent is one store field (`agents.packages`), plus the
`agents.packagesIsolated` switch, and the engine never reads the store: both
travel as `Query.AgentSources` and `Query.AgentIsolated`. The unified read
resolves them from the agent named in the query — its entries, the flag, and the
agent's name for the badge (or the workspace's, for the unnamed default agent) —
and an agent that is gone contributes no agent rows rather than an error.

The layer means one thing: **the CLI's launch passes the entries on.** Pi's list
rides its own launch as `pi -e <entry>` per entry, with `--no-extensions
--no-skills --no-prompt-templates --no-themes` when the agent is isolated (the
`CLIFlags` store field it has always used). Omp's new scope is the same fact
through its own flags: `omp -e <entry>` per entry, and `--no-extensions
--no-skills` when isolated — the two flags that CLI has, never one it would
refuse (`agentOmpScopeFlags`, ADR-0176 slice 4). Pi and Omp are the two CLIs
that declare the layer; the others have no per-agent plugin layer to declare,
and a CLI's own verb — which runs the vendor's command — refuses `scope=agent`
by name (`ErrAgentScope`), because that layer is PiCode's own list on the agent
row rather than something the vendor knows. The CLIs that take MCP
servers at launch get PiCode's own tools through the connector lane
([picode-mcp.md](picode-mcp.md)) rather than this pane's scope.

`Caps.IsolatedSwitch` is offered exactly where a launch honours it (`pi`, `omp`),
and the switch reads and writes `PATCH /api/agents/<id> {packagesIsolated}`. On
that layer the vendor's own verbs are absent rather than dead: a toggle would
run a command that has never heard of the entry, and the vendor's catalog does
not exist there, so the pane draws no enable/disable, no Inspect and no
Marketplace tab — Remove and Update, which are PiCode's own calls, stay.

## The pane

One component per app — `web/browser/src/components/Packages.jsx` and
`web/mobile/src/components/Packages.jsx` — mounted under `#cli-packages-view` by
each app's `CliPackages.jsx`, which names no CLI: the route names one, the pane
asks the engine what that CLI is, and an unknown CLI is refused by the engine's
own sentence. The whole browser-side contract lives in
`web/shared/domain/cliPackages.js` (pure, tested without a browser): the hash
parsing and the legacy redirects, the request builder (`packagesApi`), the
vocabulary, and the rules that gate a control.

**One pane, two vocabularies, kept verbatim.** `packagesSurface(report)` answers
`gallery` when the report's catalog is PiCode's own npm search and `vendor`
otherwise, and `PANE_WORDS` carries what the two panes said before they merged,
sentence by sentence: `Install to` against `Plugins go to`, the source
placeholder, the access line, the installed filter and its empty answer, the
empty title, the fallback scope, and the isolation line (`vendor` has none). A
CLI's rows are the CLI's own — its plugins — while Pi's are packages, and
nothing else about the load differs.

Every control is gated by the report: `Caps` decides install, remove, toggle,
update, inspect, the marketplace and the config editor, `Catalog` decides
whether there is a Marketplace tab at all, and a verb a CLI does not have is
rendered as the declaration's own sentence (`packagesNotes`) instead of a button
that could only fail. The scope radios come from `Report.Scopes`, each with the
line under it that the declaration wrote, and a layer the read cannot honour is
not offered. A load that fails keeps the last good roster on screen and shows
the CLI's own words with **Try again** — never an empty installation.

The badge read (`paths.updates`) runs once per mount where the CLI has an update
verb, and a row offers **Update** only where the CLI's own catalog says it is
behind: a version pair the comparator cannot parse gets no badge rather than a
wrong one, and a catalog that cannot be read leaves every row unmarked with the
reason in the note. Where a vendor refuses without PiCode's help, the refusal
carries the exact command and the pane offers **Copy command** beside "Run it in
a terminal."

The list is the card grid the Pi pane always drew, with the vendor's own words
on each row, group headers when a list mixes origins (the vendor's own
provenance word, from `sourceGroupKey`/`groupInstalledRows`), a filter that
highlights why a row matched (`matchParts`), the filter riding in a sticky block
where a catalog runs to hundreds of rows, and an empty installed list that reads
the CLI's catalog once and offers up to three real entries with Install where
the CLI can install.
Both apps carry the same pane and the same words; the phone adds **Open desktop
layout** for the config page.

## The vendor facts

The roster is the CLI's own, and the shapes below are measurements, not
conventions:

| CLI | Roster | Notes |
|---|---|---|
| Claude Code | `claude plugin list --json` (+ `--available`) | `id@marketplace`, scopes `user\|project\|local`, plus vendor-owned `synced` rows |
| Codex | `codex plugin list --json` (+ `--available`) | remote marketplaces need the network and a sign-in |
| Grok | `grok plugin list --json` (+ `--available`) | |
| Hermes | `hermes plugins list --json`, catalog via `plugins search --json` | `picode-native` is PiCode's own integration |
| OpenCode | its own configs and plugin directories | no list, remove, disable or marketplace command exists — the removal is PiCode's own splice of the `plugin` array, declared off the lane (`Lane.Remove: false`) |
| Muse Code | `muse plugins list --json` (+ `--available`) | the id lives in the row's `record`, not on the row; capability trust stays with the user. The CLI gates its whole plugin surface per machine through its own cached feature config — where that gate is off, every verb answers *"plugins are not available in this build"* and the pane shows that sentence. The catalog does **not** mark an installed plugin (`status: "available"` survives an install), so `catalogNeedsRoster` joins it with the CLI's own roster; and the CLI's list omits the app's built-in first-party plugins, which only its TUI shows (all measured 2026-09-21) |
| Antigravity | `agy plugin list` (a JSON envelope once plugins exist, one sentence while none do) | its subcommands take no flags: a leading `--help` is read as the plugin name (measured 2026-09-20) |
| Omp | `omp plugin list --json`; catalog via `omp plugin discover` (prose — `--json` is accepted and ignored); plus the configured `extensions`/`disabledExtensions` the list never names | the catalog is information only: discover prints a name and a version and never names the source an install needs, while `omp plugin install <name>` resolves through npm (measured 2026-09-21: it 404s on the registry), so `catalogInstall` is false and the list note states the `name@marketplace` form |

A verb a CLI lacks carries the declaration's one-line note instead of a disabled
button: Codex cannot disable a plugin, OpenCode has no marketplace, Antigravity
has no update command. Rows show the vendor's own words, and `Enabled` and
`Installed` are separate facts — a CLI that can disable a plugin without
removing it says so, and a catalog row is neither installed nor enabled.

**Availability, and why Update is conditional.** `CheckUpdates` compares the
CLI's own roster with its own catalog and marks the rows the catalog publishes a
newer version of; the pane runs it once per mount where the CLI has an update
verb, and a row offers **Update** only there. Versions are compared with
`pipkg.Newer`, so a vendor that versions outside semver gets no badge rather than
a wrong one, and a catalog that cannot be read leaves the roster unmarked with
the reason in the note (never "everything is up to date"). Omp's version-only
catalog is enough for the badge even though it cannot drive an Install; Codex,
OpenCode and Antigravity have no update verb and so no Update action at all.

**Omp's `extensions` are rows.** The omp read merges the two layers the CLI
loads from: the workspace's `<ws>/.omp/settings.json`, parsed with the standard
library, and the user level through `omp config get <key> --json` (never a YAML
parser for `~/.omp/agent/config.yml`). One row per entry — the entry as written,
the CLI's own name for it, the resolved path when something is there, machine or
workspace scope, and `Enabled` from `disabledExtensions` — with a row the plugin
roster already carries left unduplicated. Measured 2026-09-21 on 18.2.8:
`omp plugin list --json` is the plugin store and never names a configured
extension; `omp config get extensions --json` answers the layer the working
directory resolves to, so a project that declares `extensions` replaces the
user's; `omp config set` writes the user file wherever it runs; and no vendor
verb adds or removes a configured extension.

A read that fails is never an empty list: unparsable vendor output is a 502
carrying the CLI's own text, a missing binary a 400, a changed file a 409.
`internal/clipkgs/live_test.go` exercises every driver against the real binaries
in a sandboxed HOME (`PICODE_PKGS_LIVE=1`), which is where a vendor's shape
change is meant to be caught; `internal/pkgs` and `internal/server` pin the
model, the mappers and the routes over fixtures.

### How a guest list reads (2026-09-21)

The defaults are the machine's scope: the web clients **omit** `scope` for it, so
`clipkgs.List`, `Available` and `CheckUpdates` resolve the scope once at the top
(`normalizedScope`) and use the resolved id for the vendor's argv, the rows'
filter and the cache key. Passing the empty string through was a defect, not a
shorthand: Claude Code's parser compares a row's scope with the requested one,
its rows say `user`, so the pane showed an empty list for a CLI that has plugins
(regression test `TestTheDefaultScopeReadsTheMachineScope`).

`web/shared/domain/cliPackages.js` decides what the pane shows without touching
the rows: `groupInstalledRows` groups a list that mixes origins by the vendor's
own provenance word (`sourceKind` — the specific kinds before the marketplace
fallback, since Claude names a synced plugin's marketplace `synced` too) and
returns nothing for a single-origin list, so a header only ever explains a
mixture; `matchParts` splits text around the filter's needle for the highlight.
Both are pure and tested in `cliPackages.test.js`.

## Configure a package (ADR-0099, ADR-0119)

Configuration is Pi's, and it is a driver capability: a package is configurable
in this view when a config descriptor resolves for it (`Caps.Config`). Descriptors
resolve in order — the user's own description
(`DataDir/package-configs/<id>.json`), then `picode.config` in the extension's
`package.json`, then PiCode's catalog (`internal/pipkg/configdescriptor.go`).
The generic engine writes the declared file (agent or workspace scope, per
descriptor) with the descriptor's typed fields — enum, boolean (tri-state:
untouched = unset), number with min/max, string, secret — preserving unknown keys
and refusing to replace an unparseable file without explicit force. The config
page names the source of the descriptor (user, catalog, manifest). A user
description is created with **Describe config…** on an undescribed card and
edited or deleted from the config page; deleting it never touches the described
file. pi-roles keeps its bespoke two-layer editor. No descriptor resolves, no
Configure button; a CLI whose report declares no `Config` refuses the config link
instead of drawing a roster under a configuration hash.

## Open work

`docs/handoff/open/packages.md` carries what is measured but not yet done:
Muse's built-in first-party plugins stay outside every surface PiCode may read.
An extension row's Enable/Disable used to draw the CLI's plugin verb, which does
not know an extension; it now writes `disabledExtensions` itself (slice 5's
debt, paid 2026-09-22 by `feat/packages-toggle`). OpenCode's removal used to
answer 400 from the pane because every removal was reserved as a job and that
mutation has no argv; the transport declaration is per verb now
(`Caps.Lane`), so the removal is PiCode's own write and the pane shows its
transcript (paid 2026-09-22 by `feat/packages-opencode`).
