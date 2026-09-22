# Native CLI settings (ADR-0101, ADR-0163)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

Agent CLIs has CLIs and Messages tabs; a CLI's page hosts Launch, Terminals,
Sessions, Providers, Settings, Keyboard, Memory, Packages and Connectors panes
(a run/setup split in the inner tablist). Canonical settings are
`#/clis/<cli>/settings`. The shared `cliSettings` domain module parses
canonical/legacy routes; `cliNative` declares which CLIs have a guest editor
and maps the route's `layer` to the driver's `scope`. Each app owns
`CliSettings`, the embedded Pi editor and the guest editor; no presentation
crosses app boundaries. Memory has its own file,
[cli-memory.md](cli-memory.md).

## Two editors behind one pane (ADR-0163)

Pi keeps everything below: its own API (`/api/pi-settings`), its three layers,
its trust rules and its keyboard map. The other eight managed CLIs are edited
through **their own config files** by a schema-driven editor. ADR-0163 amends
only the registry that said Pi was the single CLI with any editor at all; every
row of ADR-0101 stays true for Pi.

`internal/clisettings` is one generic file driver plus a declaration per CLI:
the config paths per scope, the document format, and a typed field list whose
keys are the vendor's own dotted names. A key nobody declared is never written,
and a CLI without a declaration has no editor rather than a generic JSON box
(ADR-0099 §5). Fields are scalars only, which is what makes the writer safe.

| CLI | Machine file | Workspace file | Format |
|---|---|---|---|
| Claude Code | `~/.claude/settings.json` | `.claude/settings.json` | JSON |
| Codex | `~/.codex/config.toml` | — | TOML |
| Grok | `~/.grok/config.toml` | `<ws>/.grok/config.toml` | TOML |
| Hermes | `~/.hermes/config.yaml` | — | YAML |
| OpenCode | `~/.config/opencode/opencode.json` | `<ws>/opencode.json` | JSONC |
| Muse Code | `~/.config/muse/settings.json` | — | JSON |
| Antigravity | `~/.gemini/antigravity-cli/settings.json` | — | JSON |
| Omp | `~/.omp/agent/config.yml` | `<ws>/.omp/config.yml` | YAML |

A CLI with one file gets **no layer switcher and no invented project scope**,
and a write to a scope it does not have is refused. A CLI that *does* declare a
workspace file, opened without a workspace, still shows the layer with one line
saying what it needs and no form: hiding it made a link carrying
`?layer=project` silently edit the machine file instead (found in live QA,
2026-09-20).

**A row says what happens if you leave it alone.** A boolean is the Radix
switch every pane uses (owner's call, 2026-09-20: one control for one job). A
switch has no third state, so an unset key is drawn at **the CLI's own
default**, which every boolean field declares and
`TestEveryBooleanDeclaresItsDefault` holds against a table of where each value
was read from. The source line carries the provenance instead — "Set here",
"From Global", or "<CLI> default" — and a default that is conditional
("On while memories are on") stays as the row's help line. Declaring a default
is a claim about someone else's software: Hermes' `display.show_reasoning`
shipped as "Off" here while its own `config_defaults.py` says `True`, which is
the failure this table exists to catch. A
dangerous value carries its own one-line cost, not a shared sentence: two rows
in one group printed the same seventeen words twice, and
`TestDangerNotesAreDistinctWithinAGroup` now refuses that.

**Reads parse, writes splice.** A read uses the real parser for the format.
Setting a value locates the byte span of exactly one scalar and replaces those
bytes, so comments, key order, indentation and every key PiCode does not know
come out unchanged — `TestGoldenSingleKeyWriteIsSurgical` compares the text, not the
parsed tree. A key the file does not carry is inserted where the format expects
it; Handing a key back (**Use inherited**) removes its whole line, so a comment
*on that line* goes with it; comments and blank lines that follow the container
stay, and an emptied container goes with its last key (the ADR-0099 rule). A
container that still holds a subtable is not emptied and its header stays.
The result must parse before it is written, the write is atomic (tmp + rename),
and the file's mode is preserved. One accepted difference: appending to a file
that had no final newline adds one, and a later reset cannot take it back.

**What the writer refuses rather than guesses** (adversarial review,
2026-09-20): a TOML array of tables, since its elements are user data no
declared key can address; a JSON key that appears twice, since the writer would
splice the one every parser ignores; a JSON path whose parent exists but is not
an object; a YAML value that is a block, a block scalar, an anchor or an alias.
TOML multi-line strings and YAML block scalars are located and stepped over, so
a `[table]`-shaped line inside one no longer retargets a write.

**Two writers.** The CLI writes these files too. Every layer carries a
`revision`; a save built on an older read answers **409** and the file is
untouched until the editor re-reads — today the pane reports the refusal and
reloads; the `force` override exists in the API and has no control yet. A
document the parser rejects is reported with the layer marked unwritable and is
never overwritten (ADR-0099 §4).

**Launch and Settings name each other, and stay separate.** The 2026-09-17
study refused merging them on purpose: Launch composes one invocation's
arguments, Settings writes the file every invocation reads, including launches
made outside PiCode. A field whose value loosens a safety boundary carries a
one-line warning beside the choice rather than hiding it.

API: `GET /api/cli-settings?cli=&workspace=` returns the fields and one entry
per layer with the keys that layer sets; `PATCH /api/cli-settings` takes
`{cli, scope, set, reset, revision, force}` and answers with the fresh report.
Both publish an ephemeral `cli.settings` event — the file stays authoritative
(ADR-0048 invalidates views, it does not carry the config).

## Model roles: a row whose path the vendor supplies (ADR-0181)

ADR-0163 said fields are scalars only, and that was what made the splice safe.
Omp needed two shapes it could not express, so the engine gained exactly two
kinds and nothing more:

- **`role`** is still one scalar — a model selector — but its path is composed
  per request from the vendor's catalog plus the files. Omp has fifteen
  built-in roles (`src/config/model-roles.ts`, transcribed into
  `omp_roles.go` and pinned by `TestOmpRoleCatalogIsTheVendorsOwn`), and the
  union rule is the vendor's own: built-ins first, then a role named by
  `cycleOrder`, by an assignment, or by `modelTags`. A role only the workspace
  sets is still a row on the machine layer, showing what that layer inherits.
- **`list`** is an ordered list of strings — `retry.fallbackChains.<key>` and
  `cycleOrder`. It is written by `spliceList`, which is now the whole body of
  ADR-0174's `Doc.SetStrings`: one implementation, so the settings writer and
  the key-map writer cannot drift on a file with no final newline or a value
  the user broke across lines.

A field may carry `Path`, because `retry.fallbackChains.openai/gpt-4.1-mini`
cannot be split on dots. `Path` never leaves the server — the pane echoes the
opaque key and the writer rebuilds the path from the declaration whose prefix
owns it, which is also the only place that decides a **new** name is writable.
A role id must match omp's own rule (`^[a-zA-Z][A-Za-z0-9_-]*$`, its hub's);
a chain key must be addressable as one line by the YAML walker (no colon, no
quote, no leading indicator). Anything else is refused **by name** and the file
is untouched.

**What the writer does not do is judge a selector.** `openrouter/z-ai/glm-4.7@cerebras:high`
is one model id with a routing suffix and a thinking suffix; only the first
slash splits the provider, and an id may legitimately end in `:max`. So PiCode
enforces only what its own writer must guarantee — a scalar is one line, and a
value with a line break is refused rather than written as a YAML block that the
next save would duplicate — and leaves the meaning to Omp. The pane splits a
selector for display and joins back exactly the two pieces it split, so a value
PiCode did not recognise round-trips byte for byte.

A value a layer holds in a shape the writer will not rewrite is reported in the
layer's `unreadable` list, dropped from its values, and drawn as one line plus
**Open the file**.

**The picker asks the CLI.** `GET /api/cli-models?cli=&workspace=`
(`internal/climodels`) runs the vendor's own read-only catalog command in the
workspace being edited, when a picker opens and never on mount. The directory
decides the answer: measured 2026-09-22, the same command returned 55 models in
one folder and 0 in the next, because the project set `disabledProviders`. Zero
is ambiguous — a disabled provider and a missing credential look identical from
outside — so the pane says the list is empty and does not guess why.

**Why this pane and not the CLI's own command** (measured 2026-09-22, omp
18.2.8): `omp config set` writes the *global* file wherever it runs, `--scope`
is not an option, and it prints the **effective** value rather than what it
wrote — inside a project that sets `symbolPreset: ascii`, `omp config set
symbolPreset nerd` wrote `nerd` globally and answered `[ok] Set symbolPreset =
ascii`. The workspace layer is reachable only by writing the file.

**Two defects the first real write found** (2026-09-22, driving the pane on a
scratch instance instead of stubbing its requests): the pane handed a raw
object to `fetch`, so every guest CLI's save had answered "invalid request
body" since the pane shipped on 2026-09-20 — the body is JSON-encoded now, the
way every other pane sends one; and `yamlInsert` created a nested key's whole
chain from the root when only the *middle* was missing, appending a second
`modelTags:` block beside the one the file had, which the parser refused. It
now creates only the missing tail under the deepest prefix the document
already has, the walk `jsonInsert` has done since its own review. A
two-level key never hit either path; the fixtures did not either
(`docs/handoff/open/process.md`'s "fixtures do not test a writer").

**Pickup is a restart**, measured live the same day: with an Omp TUI running,
an edit to `config.yml` changed nothing passively, after two resizes or after a
keystroke; the same binary restarted in the same directory came up on the new
value. There is no watcher in the CLI's config layer, and `reloadFromDisk()`
has one caller (a structured subagent). PiCode launches every Omp terminal with
its own `--session-dir` but a shared `~/.omp/agent`, so several live terminals
share this file: the last save wins it, and each other terminal picks the
change up when *it* restarts.

## The Models pane (ADR-0181, slice 2)

**Where the matrix is drawn** (owner, 2026-09-22): in the Models pane, not
Settings. omp's own model hub keeps roles and the model list on one screen,
and a role picks from the very catalog that pane shows, so a role pointing at a
model the folder cannot reach, or one the allowed list excludes, is said on the
role's own row. Every field the roles declaration builds, and the retry knobs
and `modelRoleStorage` beside it, carry `pane: "models"`; the Settings pane
keeps `defaultThinkingLevel`, memory and interface. The file, the revision and
the writer are the same either way — `pane` only decides where a row is drawn,
which is why the move needed no change to ADR-0181.



`#/clis/<cli>/models` exists only for a CLI PiCode can ask for its catalog
(`internal/climodels`; omp today, and `TestJSListMatchesTheReaders` holds the
JS list `MODELS_CLIS` equal to the server's). It shows what the CLI reports it
can reach, grouped by provider, with kind chips, a filter, context size and the
vendor's price pair, and it edits the two lists that decide which of those the
CLI may use: `enabledModels` (**Allowed**) and `disabledProviders` (**Hide
provider** / **Show**).

Both lists are ordinary `list` rows of the settings report, marked `pane:
"models"` so the Settings pane leaves them out: one file, one revision, one
writer, and no key edited in two places that disagree about what it means.

Two vendor facts decide the writes, both read from omp's own docs and source
on 2026-09-22:

- **Arrays replace across layers.** A workspace `disabledProviders` is the
  whole list there, not an addition to the global one ("the most common
  surprise", omp's `docs/settings.md`). So every toggle writes the layer's
  **complete** list, starting from what the layer inherits when it sets none
  (`toggledList` in `web/shared/domain/cliModels.js`). Measured on a scratch:
  hiding `openai` in a workspace that inherited `[groq]` wrote
  `["groq", "openai"]`, and the global file was untouched.
- **`enabledModels` is a hard filter, not a favourite.** Once it holds
  anything omp uses only the models it matches, and none matching means no
  usable model (`resolveAllowedModels`, `src/config/model-resolver.ts`). The
  pane says "uses only the N allowed models", and when every allowed entry is
  an exact selector the CLI no longer reports, it says the folder has **no
  model to use** — the case the QA found, a global allow-list naming a model
  whose provider the workspace hid. A glob or a fuzzy entry is listed as
  written and never judged: what it matches is the CLI's call.

A folder-scoped entry (`{path, models}`) is a shape the writer does not
rewrite; the layer reports the key `unreadable` and the pane offers "Open the
file" instead of switches.

**The catalog is cached by the files that decide it.** One probe costs about
eleven seconds. `climodels.Read` keeps an answer while both config layers and
`models.yml` are unchanged (path, size, mtime) and for at most ten minutes, so
a reopen is instant and a save in PiCode's own panes is never answered from
before it. The credential store and the catalog cache are deliberately not in
the fingerprint: omp rewrites both on every run (the first version keyed on
them and never hit), and several omp terminals share them. **Refresh** sends
`fresh=1`, which asks the CLI again regardless.

## Pi (ADR-0101, machine rows added 2026-09-20)

Pi keeps its own API (`/api/pi-settings`), its three layers, its trust rule and
its live-apply. What it gained is the rest of its own settings file: pi
persists about forty keys and the pane showed eight, so a machine set to
`theme: dark` and `hideThinkingBlock` saw neither.

Five more rows now sit on the **This machine** layer, the only place pi writes
them (`globalSettings.*` in its own settings manager): `theme`,
`hideThinkingBlock`, `quietStartup`, `defaultProjectTrust` and `shellPath`.
Each name and value domain was read out of the installed pi bundle before it
was declared — `defaultProjectTrust` carries exactly `ask | always | never`
because pi's getter resolves anything else to `ask`, and the row would
otherwise never look set. `npmCommand` is deliberately absent: pi stores it as
an argv array, and this pane writes scalars only.

A machine-only key sent to the workspace or agent layer is refused by name
before trust or path resolution answers, so the message says what is wrong
rather than blaming the folder.

**Pi's rows are a table too** (2026-09-20). `web/shared/domain/piRows.js`
declares all eleven — label, kind, group, order, and what each one reads — and
the pane renders whatever the table says, the way the guest pane already did.
Three of them are not scalars and are declared as their own kinds rather than
flattened: `model` is the three coupled selects the catalog feeds, `patterns`
is the free list of scoped models, `tools` is the grid over pi's fixed tool
set. A row marked `machine` is offered only on the Global layer.

`web/shared/domain/resolveLayer.js` is **derived** from that table
(2026-09-20): every field declares its type and its `unset` value — what pi
itself does when no layer sets the key — and the resolver walks the table
instead of restating it. A row is one line, and the resolver cannot fall
behind the pane, which is what let the Theme row ship rendering empty while
the file said `dark`.

The rule the derivation preserves: a layer that **sets** a key keeps its value
even when that value is the type's zero — an explicitly empty tool list means
no tools, not the built-in set, and an explicit `false` is false rather than
pi's default `true`. A layer that sets nothing inherits its parent, and where
the parent has nothing the field's `unset` runs. Handing a key back removes
this layer's override; it does not restore a value the layer held before.

One difference from the guest pane remains, and it is pre-existing: Pi's
writer re-encodes the document (`json.MarshalIndent` of a map), so key order is
normalised rather than preserved.


The workspace agent menu separates **Launch settings** (the bound terminal's
common launch editor) from **Settings** (this agent's model, reasoning, tools
and checklist). An agent without a terminal offers Settings only. The launch
editor links back to the scoped Settings route; native settings links keep
their existing meaning.

The native settings view does not load terminal inventory or installation jobs.
Explicit agent IDs are validated against Pi's report before looking up their
workspace. Free agents have no project layer; missing identities and unsupported
CLIs show recovery actions without falling back to Pi or another agent.
Pi settings/keys APIs, native files and trust remain unchanged. A transient
context refresh failure keeps the mounted editor and its drafts, with writes
blocked until a successful retry. A missing agent/workspace removes the editor.
Native defaults have a separate loading/error boundary from agent controls and
key bindings; the mobile quick sheet remains editable when the native file is
unreadable, without turning guessed defaults into agent overrides.
A failed desktop restart propagates to the editor as partial success after
PATCH; a failed stop prevents start. Success is reported only after the full
sequence completes. Desktop and mobile retain their own runtime UI adapters.

Each app owns an `AgentClisFrame` for all Agent CLIs tabs. The desktop frame
uses one 1240px maximum width, a consistent unpadded card and a stable scrollbar
gutter; mobile uses the full page width. Route IDs do not control page sizing.

## Pane shape (2026-09-12, `docs/plans/cli-settings-ux.md`)

The pane edits **one layer at a time**. A labelled switcher (*This machine*,
the workspace, the agent) writes `layer=global|project|agent` onto the route
beside `agentId`, the body renders only that layer's rows, and the file it
writes is named under the switcher. Values a layer does not set come from its
parent, and the row says so (`From Global`, `Pi default`); a row this
layer sets carries the accent bar, `Set here`, and **Use inherited**, which
sends `patch.reset[]` so `pisettings.Apply` deletes exactly those keys (an
empty `compaction` object goes with its last key; an unknown name is refused
with 400 rather than reporting an inheritance that did not happen). Compaction,
steering and follow-up are live-applied from the *effective* values after a
reset, so a running agent never keeps the override that just left the file.

The agent layer keeps PiCode's own fields (Model, Tools, Checklist) and the
`PATCH /api/agents/{id}` path, and it sits outside the native defaults
fieldset: an unreadable `settings.json` leaves it editable (ADR-0101 recovery).
The `layer` param survives a reload; an unknown value is dropped and the pane
falls back to the default layer — the deepest layer the route names (an agent
link opens the agent's layer), or the machine layer for a `focus=scoped-models`
shortcut. Uncommitted pattern text is kept per layer while the pane stays
mounted. Clicking the pane tab itself is plain navigation: it lands on the
defaults.

The keyboard map is the **Keyboard** pane next to Settings
(`#/clis/pi/keyboard`, owner's call 2026-09-12: two tab rows inside one pane
read as nesting). It is machine-wide — `keybindings.json`, its own endpoint
`/api/pi-keys` — so it has no layer switcher and keeps its own filter, and a
failing `settings.json` read cannot lock it. Its link carries the settings
context (`agentId`, `layer`) so a round trip lands back on the same agent and
layer; the route ignores the rest. A `?tab=keys` link from the sub-tab day
redirects to it.

## The Keyboard pane (2026-09-21, `docs/plans/keyboard-pane.md`)

The map is a screen of its own: one row per action (90, in 12 groups), the
chord drawn as a keycap at its own height rather than `--ctl-h`, the row's own
**Add key** / **Reset** in a fixed right column instead of an Add button
chasing the chips, and a toolbar with the filter, **Find by key** (press a
chord; the list narrows to the actions that answer to it) and counting facets
— **Changed**, **Shared**, **Off**.

**The row is four cells, each placed by `grid-column`, never auto-flowed**:
`label (minmax(9rem, 18rem)) | keycaps (minmax(0, auto)) | note (minmax(8rem,
1fr)) | actions (auto)`. The label is bounded so the keycaps follow it instead
of floating at the card's far edge; the note — a shared chord, or a chord a
browser keeps — takes the space the keycaps leave and is the row's only air.
Two traps worth knowing before editing this CSS:

- The base `.key-label` rule (AppKeys' language, kept verbatim) carries
  `flex: 0 1 12rem` from the layout where the label sat in a *row*, where a
  12rem basis is a width. Inside the pane's `.key-name` column the same basis
  is a height, and the label — with its cell, its grid row and the whole row —
  measured 192-216px. The pane resets it (`flex: none; padding-top: 0`).
- A conditional note before the actions column shifts the actions into the
  flexible column when there is no note (and onto the next line when there are
  two). Hence the explicit placement.

Measured: rows 32-90px (a note wraps on the longest rows), 3 199px for all 90,
12px between the label and the first keycap; the phone 44-106px and 5 058px.
`qa-cli-settings.mjs` asserts the shape now — a label cell one line tall, the
action and its keycap on one line, the keycaps within 48px of the label, and a
row/total height ceiling per app.

**Two doors, one screen** (P1, 2026-09-21): `/api/pi-keys` stays pi's own store
and contract, and `/api/cli-keys?cli=<id>` answers the pane for any CLI the
registry knows (`internal/clikeys`). Pi's report is *wrapped* into that envelope
— never restated, so the catalog and the write path stay in one place — and a
CLI whose editor has not shipped answers with its state and nothing else: no
file, no actions, and a `PUT` refused by name ("PiCode cannot write Codex's key
map yet" versus "Grok does not allow its keys to be remapped"). The envelope
carries what a guest editor will need and pi does not use yet: `state`,
`keymap`, `pickup`, `vocab` and `contexts`. The pane talks only to the envelope,
so a guest's editor is a server-side declaration, not a second component.

The registry declares each CLI's vendor facts — the map's shape, when the CLI
picks an edit up, its contexts — and every row cites where those were read,
because that is a claim about software PiCode does not own. `unknown` is a
first-class pickup: nothing vendor-published says, so the pane promises nothing.
`web/shared/domain/cliKeys.js` holds the copy (the pickup sentence per state,
and one line + one action for a pane with no editor);
`TestJSListMatchesTheKeyboardRegistry` holds its ids, states and pickups equal to
the Go registry's, and `make keys-drift` re-reads the installed pi's own
`docs/keybindings.md` against `pikeys.Catalog` — the probe that would have caught
the 89-of-90 catalog and the nine platform defaults.

**Every CLI answers for itself** (P2a, 2026-09-21): the pane takes its CLI from
the route and the registry decides what it shows. A row whose map PiCode can
edit — Pi today — falls through to its own editor; every other row draws its own
answer: one line and one action from `blockNote()`, plus the pickup sentence only
when there is a map to pick up. `noteIsExternal()` decides how the action opens
(a vendor's page in a new tab, an in-app route in the tab), and the harness
asserts both, because a note whose action does nothing is not an action. The
"in development — coming soon" placeholder is gone from this pane: it was a
promise about PiCode dressed as a fact about the CLI.

Hermes is the first row whose answer is not "wait": it keeps no key map file, so
its three rebindable keys — `voice.record_key`, `copy_shortcut`,
`display.busy_input_mode` — are declared rows in its Settings pane (group
**Keyboard**, defaults and value ranges read from the CLI's own
`hermes_cli/config_defaults.py` on 2026-09-21), and its Keyboard pane sends the
reader there. Its registry row stays `planned` because there is nothing for a
key-map writer to do — the *shape* says so (`Keymap: Partial`) and names it in
the refusal: "Hermes Agent keeps no key map file; the keys it does allow are in
Settings". The front table carries that shape too, and
`TestJSListMatchesTheKeyboardRegistry` compares it with the registry's.

**The flat engine and Omp** (P2b, 2026-09-21): one file, one object of action id
→ chord list. `internal/clikeys` declares where a CLI's file lives, what its
catalog is and which platform vocabulary it speaks, and reads and writes rows
through four primitives exported from `clisettings` (`internal/clisettings/keymap.go`
— `OpenDoc`, `Strings`, `SetStrings`/`RemoveKey`, `Save`): the same parser,
byte-span splice, atomic rename and revision check the settings editor has used
since ADR-0163, rather than a second implementation of guarantees two adversarial
reviews taught (a file with no final newline, a `[table]`-shaped line inside a
string, a `// comment` before a closing brace). A key map's value is a *list*,
which the settings engine never writes, so `listLiteral` is the one new piece of
syntax — and the only format it refuses is TOML, which no key map PiCode writes
is written in.

What the pane sees comes through the same envelope as pi's: `file`, `exists`,
`revision`, `actions` (the CLI's own catalog), `user` (the rows the file sets),
`unreadable` (rows the file holds in a shape PiCode will not rewrite — dropped
from the list, never written) and the CLI's own `platform` vocabulary. A write
sends back the revision the pane read: a file that moved answers 409, and
`ErrShape` is what a write to an unrewritable row returns.

Omp's declaration is the first one. Its catalog is 70 action ids (32 `tui.*` +
38 `app.*`, labels from the CLI's own descriptions, `omp_catalog.go`), read out
of the installed bundle; its file is one machine-level file in the CLI's agent
dir — `keybindings.yml`, else `.yaml`, else the legacy `.json`, which the CLI
itself migrates to YAML — resolved through the CLI's own config root
(`$PI_CONFIG_DIR` or `~/.omp`) and profile (`$OMP_PROFILE`, else `$PI_PROFILE`);
and `[]` unbinds. Its pickup is **restart**, measured rather than assumed: the
manager reads the files when it is created and nothing in the bundle calls its
own `reload()`, so the pane says "restart it" instead of promising a live
reload. A row whose value is not a string or a list of strings is reported as
unreadable and left alone, and `Reset all` removes only the rows the catalog
knows.

**Antigravity joins it** (2026-09-21): the same flat engine, one declaration —
`~/.gemini/antigravity-cli/keybindings.json`, 36 actions in ten id namespaces,
the vendor's *override* file (removing a row hands that action back to the
binding built into the binary, which is exactly what the engine's reset does).
Its pickup is **measured, not assumed**: with the file remapped under a running
session, the TUI kept the map it loaded at start — the old key still cycled the
mode, the new one did nothing — so the registry row and the pane say "restart".
Twenty rows carry the vendor's own descriptions from the two doc pages that match
the installed build; the other sixteen are labelled from their id, which those
pages leave unnamed.

**OpenCode joins it** (2026-09-21): a flat `keybinds` object one level down in
the CLI's user `tui.json` — the same engine, `FlatMap` paths again. Three
vendor-specific facts live in its declaration and tests: the file is
`tui.json`/`tui.jsonc` (a `.jsonc` sibling wins; the main `opencode.json` is
never written, because the CLI's own startup migration moves legacy keybinds
into a `tui.json` behind PiCode's back); an action disabled by the vendor's
literal `"none"` reads as unbound rather than as a chord named none; and the
scope is the user file only — a project `tui.json` deep-merges over the user's
rows and wins, which the pane's file line states. Its pickup is restart, from
the loader's own code: the config is snapshotted once when the TUI starts (the
SIGUSR2 reload RPC refreshes only the server's `opencode.json` cache). The
catalog is 162 actions from the loader's own Definitions table at the installed
tag; the leader-key configuration and the vendor's object-form paste binding are
deliberately absent, each noted in the catalog header.

**A nested map, and a vocabulary that is not the pane's** (P3, 2026-09-21):
Codex's map is the same engine with a different declaration — one table per
context, every row at `[tui.keymap.<context>]` as `action = "chord"` (not a table
per action: the plan's `[tui.keymap.<context>.<action>]` would be a TOML type
error, corrected against the vendor's schema). `FlatMap.Path` is the whole
difference, and the file is the one the Settings tab already edits — both panes
touch different keys of it through the same document primitives, so each save
re-reads what the other left. Its catalog is 149 keys in 12 contexts, read out
of the vendor's artifacts at the tag the installed build pins (the runtime
inventory in `tui/src/keymap/bindings.rs`, the labels in its generated
`config.schema.json`, the chords in `built_in_defaults()`), and a row's ID is
`<context>.<action>` because the bare name is not unique: `move_left` exists in
the editor and in `vim_normal`, and the pane addresses each row by one key.

Codex's file is also the first whose *syntax* differs from the pane's: it writes
`ctrl-alt-m` where the pane captures `ctrl+alt+m`, and it validates its whole
keymap when it starts — a chord in the wrong spelling is a CLI that does not
start. So a declaration can carry `Normalize`, which renders a captured chord
into the file's own vocabulary (`pageup` → `page-up`, `-` → `minus`, `escape` →
`esc`, `f1`..`f24`) and **refuses the rest by name** (`super+m`: "codex has no
"super" modifier"); a chord already written the file's way is checked and kept.
The pane does the same on the way out: `formatChord(chord, vocab)` keeps the
CLI's separator in the display rather than translating one vocabulary into
another, and `reservedChordOf(chord, vocab)` recognises codex's spelling when
deciding whether the browser would eat a chord.

Two things the pane deliberately does not claim:

- **A shared key is not a conflict.** 52 of pi's 90 actions share a chord with
  another: pi's contexts overlap and pi does not declare them, so calling them
  conflicts would be a lie. The row says *Also on …* and offers **Show**; the
  facet is named **Shared**. A CLI that *does* declare contexts can call them
  conflicts later (`docs/plans/keyboard-pane.md` §4).
- **The default printed is this machine's.** Nine actions bind differently on
  Windows and WSL — `pikeys.Catalog`'s `Alt` map, read out of pi's own
  `docs/keybindings.md` on 2026-09-21. `tui.editor.undo` is `ctrl+z` on
  Windows and `alt+z` on WSL, `app.suspend` has no binding at all on native
  Windows, the rest swap (`ctrl+v` → `alt+v`, `alt+enter` → `ctrl+q`,
  `ctrl+shift+f` → `ctrl+f`, …). One platform's binding *is* the whole default
  there — never an addition — and a declared empty list means pi binds
  nothing. `TestAltDeclaresEachPlatformItCovers` holds the encoding: an empty
  list must travel as `[]`, not `null`, because the pane reads the map by
  presence.

`GET /api/pi-keys` answers `{actions, user, file, exists, platform}` — `file`
is this machine's real path (home shortened to `~`) so the pane can name what
it writes, and `exists` lets it say the file is not created yet. `PUT` gains
`{resetAll: true}` for **Reset all**: `pikeys.ResetKnown` deletes every
override the catalog knows in one write and leaves a key the catalog does not
know exactly where it was (the rule `Set` already keeps for one action), and a
no-op writes nothing. Chords a browser keeps are marked on the row from
`web/shared/domain/browserChord.js` — five of pi's own defaults sit on that set
— because inside PiCode a browser tab never receives them.

The toolbar is **sticky** at the top of the CLI page's own scroller
(`#agent-clis-view`), so the filter and the facets stay reachable 90 rows down;
the group headers are not, because they would need a hard-coded offset equal to
the bar's height, which changes when the bar wraps. (An earlier attempt reverted
the sticky rule on a bad measurement — a probe that scrolled `window`, which
this app never scrolls; the scroller is the CLI page element.) The phone sets
the bar back to `static`: its shell scrolls a different element and the bar is
a short hop from the top there.
