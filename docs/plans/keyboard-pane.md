# The Keyboard pane — a redesign, and keymaps for the other agent CLIs

Status: **proposal** (2026-09-21). Nothing implemented. Two changes to the
same pane, in this order: the pane Pi already has is hard to use, and six of
the other eight managed CLIs *do* expose a keymap file a tool may write —
so the pane stops being Pi's.

Sources: live reads of `#/clis/pi/keyboard` and of the installed CLIs on this
machine (pi 0.86.1, omp 18.2.7, codex-cli 0.155.1, Claude Code 2.1.278, grok
1.0.40, opencode 1.18.31, plus `muse`, `agy`, `hermes`), the vendor docs named
per row in §4, the four web sweeps behind
[../benchmarks/2026-09-21-keyboard-pane.md](../benchmarks/2026-09-21-keyboard-pane.md),
and the in-repo precedents: [cli-settings-ux.md](cli-settings-ux.md),
[cli-native-settings.md](cli-native-settings.md), ADR-0099, ADR-0101,
ADR-0163.

This supersedes one documented line: `CliSettings.jsx:14-15` (*"The keyboard
map stays Pi's: no guest CLI exposes a key map PiCode can write"*) and the
handoff debt that repeats it. The claim was true when it was written; it is
false for Claude Code, Codex, OpenCode, Antigravity CLI and Omp as of the
versions above. Everything else ADR-0101/0163 say about the pane stays true.

## 1. What the pane is today, measured

`#/clis/pi/keyboard` = `PiKeys.jsx` (browser and mobile are byte-identical) over
`GET/PUT /api/pi-keys` → `internal/pikeys` → `~/.pi/agent/keybindings.json`.

| Fact | Value | Source |
|---|---|---|
| Actions / groups | **89 / 12** | `internal/pikeys/catalog.go` |
| Documented by pi | **90** — the pane is missing `app.thinking.save` (`ctrl+s`) | pi's installed `docs/keybindings.md` |
| Defaults with a **platform alternate** the pane flattens | **9** — e.g. `tui.editor.undo` is `ctrl+-` (Linux) but `alt+z` on WSL, `app.suspend` has *no* binding on native Windows | same, lines 73/112-114/128-130/153/165-166 |
| Unbound (empty by default) | **10** (4 Sessions, 4 Transcript, 2 Editor) | catalog |
| Distinct chords / shared by >1 action | **68 / 25** — **52 of 89 actions** share a chord with another; `ctrl+d` serves 4 | catalog |
| Defaults that sit on a **browser-reserved** chord | **5** (`ctrl+t`, `ctrl+w`, `ctrl+n`, `ctrl+pageUp/pageDown`) — while the pane says nothing about them | catalog × `web/shared/domain/browserChord.js` |
| Chip height (`--ctl-h`) | **36px** — a keycap drawn at input height | `app.css:4283` |
| Row action | an **Add** ghost button on all 89 rows, laid out *after* the chips — so its column is ragged | `PiKeys.jsx:88-123` |
| Scroll | ~4500px of the ~6000px pane (2026-09-12 measurement) | `benchmarks/2026-09-12-cli-settings-ux.md` |

Not done today, code-confirmed: no conflict or shared-key view; no
modified marker beyond `Reset` appearing; no default to compare against; no
search by key; no group navigation; no per-CLI anything; `Escape` and bare
letters are uncapturable (correct, but the pane never says why); the filter
input has no label and the capture has no focus/announcement (a11y gaps).

## 2. What the references do

Ranked from the sweep (VS Code, JetBrains, macOS, Raycast, Linear, Obsidian,
Zed, Notion, Slack, Figma, TanStack's recorder); §3 names where each lands.

| Adapt | Source | The pattern |
|---|---|---|
| 1 | Raycast + TanStack `useHotkeyRecorder` | capture **in the cell**: placeholder swaps to "Press a key", Esc cancels, blur cancels, one recorder open at a time, `preventDefault+stopPropagation` |
| 2 | VS Code | **modified gutter bar** at the row's leading edge + per-row Reset — cheaper and clearer at 13px than JetBrains' bold-for-changed |
| 3 | JetBrains + VS Code "Show Same Keybindings" | conflict **inline under the row**, naming the rivals, with *Show* / *Keep both*; never a banner, never a modal |
| 4 | Obsidian `All / Assigned / Assigned by me` | **faceted filter chips**, not VS Code's `@source:` type syntax |
| 5 | Notion / Slack / macOS | section headers + per-row **enable** semantics; plain mono chords (− Notion) read better than 3D keycaps |

Refused, with the reason: chord-sequence recording (needs timeouts and
prefix resolution; guests' sequences render read-only — §4); Raycast's
auto-save countdown (a timer decides for you); JSON-escape-hatch editing in
the pane (two writers on one file — refused already in the 2026-09-12 study);
full-width conflict banners; bold/blue-for-changed; macOS's no-confirm
"Restore Defaults"; heavy keycap CSS; **plain `Escape` as a recordable
chord** (it is the cancel key every recorder treats as exit — and `Backspace`
stays recordable, because pi binds it, which is why *Backspace clears* is
refused too: it would fight the key being captured).

## 3. Part 1 — one pane for every CLI

The pane is rebuilt once, for all CLIs, from a **report the server owns**; Pi
is the first consumer. Target:

```
Keyboard                                                              Aa ↺
Actions for this machine. Writes ~/.pi/agent/keybindings.json —
Pi applies it on /reload or the next run.
┌ Filter by action, group, key or context…   ⌨ ┐  (All 89)(Changed 4)(Shared 52)(Off 10)
EDITOR                                                                      14
│ Move cursor up                        up                       ⊕   ← hover/focus only
▌│ Move left                     left  ctrl+b  ×                 ↺   ← 2px accent bar = set here
│ Next prompt                  (Off)  + Add key                  ⊕
│ Undo                     ctrl+-   ⚠ browser keeps Ctrl+PageUp  ↺
TRANSCRIPT                                                                  14
…
```

- **Row = 32px**, grid `label | keys | row-action`; the row action is a fixed
  right column revealed on hover / `:focus-within` (always visible when
  `(hover: none)`), which removes the ragged 89-button wall.
- **Keycap** = its own token class, not `--ctl-h`: mono 12px, 22px tall, 1px
  `--border-strong`, 4px radius, `2px 6px` padding — the treatment
  `.hotkey-row kbd` already uses (`app.css:417`), with `×` appearing on chip
  hover only. Chords render human (`Ctrl+B`, `Page ↑`, `Shift+Enter`) via
  `formatChord` (`web/browser/src/lib/appKeys.js:44`), never raw `ctrl+b`.
- **Group headers are sticky** with a count; the filter, the facets and the
  `⌨` button (press a key to filter to the rows that use it) are the
  navigation. Groups are refused as 12 collapsed `<details>`: a keymap is a
  reference, and hiding rows behind twelve disclosures adds clicks.
- **Facets**: `All`, `Changed`, `Shared`, `Off` — a facet whose count is 0 is
  not drawn. *Shared*, not *Conflicts*: 52 of Pi's 89 actions share a chord
  because pi's contexts overlap, and pi does not declare its contexts, so a
  "conflict" claim would be a lie. For a CLI that *does* declare contexts
  (§4), the same facet is a real conflict and gets the warn treatment.
- **Capture** stays commit-on-keypress (the server validates and refuses; a
  rejected chord writes nothing), inside the row, with the existing chips
  still visible beside the "Press a key" slot, an accent ring, one 11px hint
  (`Esc cancels · plain letters need a modifier`), `aria-live` on the
  result, and focus returned to the row. No Save button: one click per row
  for a reversible patch is worse than the app's own set-row switch, which
  saves on change.
- **Reserved chords** are marked where they are used: `isReservedChord` +
  `reservedBrowserAction` from `web/shared/domain/browserChord.js` (already
  built for exactly this, "for honest UI copy") drive a warn chip and a
  one-line reason — *"The browser keeps Ctrl+T; it never reaches a terminal."*
  5 of pi's own defaults sit on that set today and the pane never says so.
- **Platform alternates** (the 9 rows above) render as a muted second chip
  with the platform named, and the pane reads the platform it is on
  (`navigator.platform`-free: the server already knows the host — one field
  in the report).
- **Copy** loses `Keys` (the tab says Keyboard; "keys" means credentials one
  pane away), and the under-title line names the file and the reload
  semantics per CLI instead of *"Same map as Pi."*
- **States**: skeleton rows at the real 32px height; an error line with the
  path in mono and one action; filtered-empty and `Off`-empty each get one
  line (no "0" chrome anywhere).
- **Mobile**: label above a horizontally scrolling chip strip (as today), with
  the row action pinned right so the ragged column goes away there too; 28px
  tap targets.

## 4. Part 2 — the guest keymaps

What each managed CLI actually offers, on the versions installed here:

| CLI | Keymap file | Shape | Writable | Pickup | Evidence |
|---|---|---|---|---|---|
| `pi` | `~/.pi/agent/keybindings.json` | `id → string\|string[]` | yes (shipped) | `/reload` or next run | pi's installed `docs/keybindings.md` |
| `omp` | `~/.omp/agent/keybindings.{yml,yaml,json}` | `id → string\|string[]` | yes | UNCONFIRMED (measure) | installed bundle: the three filenames are one constant; **116** action ids |
| `agy` | `~/.gemini/antigravity-cli/keybindings.json` | `id → string[]` | yes | UNCONFIRMED (measure) | **the file is on this machine** with 37 actions in 8 namespaces — no vendor guess needed |
| `codex` | `~/.codex/config.toml`, `[tui.keymap.<context>.<action>]` | `action → string\|string[]`, `[]` unbinds | yes | restart (its `/keymap` refreshes live) | vendor config reference; contexts `global, composer, chat, editor, vim_*, pager, list, approval, agents` |
| `opencode` | `~/.config/opencode/tui.json` (+ project `tui.json`) | `keybinds: action → string\|string[]`, `<leader>` sequences, `"none"` disables | yes | restart (UNCONFIRMED) | vendor keybinds docs |
| `claude-code` | `~/.claude/keybindings.json` (global) | **inverted**: `bindings:[{context, bindings:{chord: action\|null}}]` | yes | **hot-reload** | vendor keybindings docs; needs v2.1.18+ |
| `hermes` | `~/.hermes/config.yaml` | **3 scalars only**: `voice.record_key`, `copy_shortcut`, `display.busy_input_mode` | partial | restart (UNCONFIRMED) | vendor docs; issue #4256 asks for the rest |
| `grok` | — | — | **no** | — | vendor docs: *"built in and cannot currently be remapped"*; `[ui]` holds `vim_mode`/`simple_mode` only |
| `muse` | — | — | **no** | — | vendor docs: `/keymap` is view-only; `settings.json` has no keymap key |

Four groups, and the pane must answer for each honestly:

- **A — same shape as Pi** (`pi`, `omp`, `agy`, and `opencode` modulo
  sequences): a flat `action → chord[]` map. One editor, one catalog format.
- **B — nested by context** (`codex`): the same map one table deeper, in a
  file PiCode already reads and writes.
- **C — inverted by context** (`claude-code`): the file keys *chords* to
  actions, so a row's "add key" is a different edit; and it is the only one
  that hot-reloads.
- **D — partial or none** (`hermes`; `grok`, `muse`): Hermes' three keys are
  scalars and ride the **existing** descriptor (no keymap engine, no new
  code path); Grok and Muse get one line + one action naming the CLI's own
  list command (`Ctrl+.` in Grok, `/keymap` in Muse) and a vendor link —
  a read-only reference table is refused, because it would be hand-maintained
  vendor data that drifts (the same reason memory tiers are declared, not
  guessed).

**Engine.** A new `internal/clikeys` owns the per-CLI declaration — layers and
paths (riding the existing `Paths{Home,Cwd}` convention), document shape,
catalog (id, context, label, default, and where the default was read from),
chord vocabulary, unbind verb, and the reload sentence. It reuses the format
layer of `internal/clisettings` (decode, span-locate, splice, insert, remove,
`writeAtomic`, `Revision`) by exporting those primitives unchanged; the
alternative — a `keys.go` inside `clisettings` — is refused because the
settings report and patch are *scalar-only by design* (ADR-0099 §5) and a
keymap row is a list. **Refusals carry over unchanged**: a TOML array of
tables, a duplicate JSON key, a non-object parent, a YAML anchor/alias, an
unparseable file (never overwritten), and a stale `revision` (409) — the same
table `TestUndeclaredKeyIsRefused`/`TestStaleFileIsRefused` already pins.
`PUT` gains an `unbind` verb beside `keys`/`reset`, and both endpoints publish
a `cli.keys` feed event (ADR-0048).

**Endpoint.** `GET /api/cli-keys?cli=&workspace=` answers one envelope —
`{cli, file, exists, reload, vocab, contexts[], platform, actions[],
user{}, unknown[]}` — and `PUT /api/cli-keys` takes
`{cli, workspaceId, action, keys|reset|unbind, revision}`. Pi's own
`/api/pi-keys` keeps its contract and is **wrapped** into the same envelope
server-side, so one component renders both. Pi's `PUT` keeps no `revision`:
pi is the only writer of that file, unlike Codex's `config.toml`, which
`/keymap` also writes.

**Vocabulary.** There is no canonical chord spelling: pi says `pageUp`,
Codex `page-down`, Antigravity `pgdown`, and its undo is `ctrl+_`. A
`web/shared/domain/cliKeys.js` table (mirroring `cliNative.js`) declares the
vocabulary, unbind verb, context list and reload note per CLI, with a Go seam
test (`TestJSListMatchesTheKeyboardCatalog`) so the two sides cannot drift;
`fromEvent`'s app-native chord is translated into each CLI's spelling on the
way out and back for display.

**Drift, the failure mode that matters.** A vendor's action list grows under
us: pi documents 90 actions to the pane's 89 (and 9 platform alternates we
flatten), Omp carries 116, and Codex's catalog lives in a generated schema.
So every catalog default declares *where it was read from*, the way
`TestEveryBooleanDeclaresItsDefault` already forces for settings, and a
`make keys-drift` probe re-reads the installed CLI's own artifact (pi's
`docs/keybindings.md`, Omp's bundle, Antigravity's shipped file) and fails
when the declared catalog no longer matches — skipping, loudly, when the CLI
is absent. Actions the file carries but our catalog does not know are
**shown, marked and read-only**, never hidden: the file is the truth, and an
editor that hides rows lies about it.

## 5. Decision table

| Conditions | Action | Coverage |
|---|---|---|
| CLI declares a keymap, file readable | Editor: rows = catalog, chips = effective keys, facet counts computed | per-CLI golden read test |
| CLI declares a keymap, file absent | Editor on defaults; the line says the file is not created yet; first write creates it atomically | `TestMissingFileIsNotAnError` (carried over) |
| CLI declares a keymap, file unparseable | Notice + path + no write, ever | `TestUnparseableFileIsNeverOverwritten` |
| CLI not installed | One line ("not installed") + no editor | new |
| CLI has no writable keymap (`grok`, `muse`) | One line + one action naming the CLI's own list command + vendor link | new |
| CLI partial (`hermes`) | The three scalar rows through the **settings** descriptor, one line that the rest are built in | new |
| Row unbound (`[]` / `null` / `"none"`) | `Off` chip slot, layout unchanged; `Reset to default` returns the vendor default | per-adapter test |
| Chord already used by another action, same context | Inline warn line naming the rivals + *Show* / *Keep both* | conflict index test |
| Chord shared across different contexts | Muted `Shared` facet only — no warning (Pi's normal state) | facet test |
| Chord in `RESERVED_CHORDS` | Warn chip + *"The browser keeps Ctrl+T; it never reaches a terminal."* — allowed, never blocked | `browserChord` unit test |
| Chord the CLI cannot express (reserved by the vendor, wrong vocabulary, plain `Escape`) | Capture refuses inline with the reason; nothing written | per-CLI refusal test |
| Chord sequence in the file (`ctrl+k ctrl+s`, `<leader>n`) | Rendered as chips with a `then` divider, read-only, one line saying so | adapter test |
| Unknown action in the file | Row shown in an "Other, not in this catalog" group, read-only, with the file path | drift test |
| File changed by the CLI between read and write | 409, the pane re-reads, nothing written | `TestStaleFileIsRefused` (carried over) |
| Two layers declared (only `opencode` today) | Layer switcher appears, exactly as the settings pane's rule; one file → no switcher | `cliKeys` table test |
| Write succeeds | Toast names the pickup semantics: *"Saved. Claude Code picks this up as you save."* / *"Saved. Restart Codex to apply it."* | per-CLI reload note test |

## 6. Phases

| Phase | Deliverable | Files |
|---|---|---|
| **P0 — the pane** | Row anatomy, keycap tokens, sticky groups, facets, `⌨` filter-by-key, inline capture, shared-key index, reserved-chord warn, platform alternates, modified bar, a11y, mobile column fix, copy; `.key-row`/`data-align-row` hooks kept so `qa-cli-settings.mjs` still drives it | `web/{browser,mobile}/src/components/PiKeys.jsx`, both `styles/app.css` + `mobile-settings.css`, `internal/server/pi_keys.go` (report gains `file`/`platform`/`reload`), `internal/pikeys/catalog.go` (+`app.thinking.save`, platform alternates), `docs-site/guide/keyboard.md`, `docs/architecture/cli-settings.md` |
| **P1 — the envelope** | One report shape for every CLI, `/api/pi-keys` wrapped into it; `web/shared/domain/cliKeys.js` table + seam test; `make keys-drift` | `internal/server/pi_keys.go`, new `web/shared/domain/cliKeys.js`, `web/shared/domain/cliKeys.test.js`, `scripts/` |
| **P2a — the pane answers for every CLI** ✅ shipped 2026-09-21 | the pane takes its CLI from the route and lets the registry decide: the editor where one exists, one line + one action where none does; Hermes' three rebindable keys become rows in its Settings pane | both `CliSettings.jsx`, new `CliKeyboard.jsx` (both apps), `web/shared/domain/cliKeys.js`, `internal/clisettings/specs.go` |
| **P2b — the engine + Omp** | `internal/clikeys` (declaration, adapters, refusals, `/api/cli-keys`), primitives exported from `clisettings`, `omp` full editor (same shape as pi), `hermes` three rows | `internal/clikeys/*`, `internal/clisettings/*` (export only), `internal/server/cli_native.go`, `web/shared/domain/cliLaunch.js` (`cliPanes`), both `CliSettings.jsx`, new `KeyboardPane.jsx` |
| **P3 — Codex + Antigravity** | Nested TOML and flat JSON adapters; catalogs from the vendor schema and from the shipped file; restart/live notes measured | `internal/clikeys/codex.go`, `agy.go`, their golden files |
| **P4 — Claude Code + OpenCode** | Inverted-context adapter (context column + context facet), `tui.json` with read-only `<leader>` sequences and the project layer | `internal/clikeys/claude.go`, `opencode.go` |
| **P5 — the honest non-editors** ✅ shipped with P2a 2026-09-21 | Grok, Muse, Hermes and the four adapters still to come: one line + one action; the placeholder sentence is gone from the Keyboard pane | both `CliSettings.jsx`, new `CliKeyboard.jsx`, `web/shared/domain/cliKeys.js`, `docs-site/guide/keyboard.md` |

Each phase ends with the rite; P0 ships alone and is the one the owner can
judge by eye.

**P2 owes one ADR** (`make adr NAME=guest-keymaps`): it adds an endpoint and
writes into other vendors' configuration directories — a protocol and a
persistence/security boundary, on the model ADR-0163 set for settings. It must
state the per-CLI keymap write contract, what the writer refuses, and that the
revision/atomic guarantees are inherited rather than re-derived. P0 and P1 cross
no boundary: a UI refinement and a report envelope, so they update
`docs/architecture/cli-settings.md` only (the `resetAll` field extends
ADR-0101's own patch contract, the way `patch.reset` did).

## P1 shipped (2026-09-21, `feat/keyboard-envelope`)

The envelope is live and Pi answers through it: `internal/clikeys.Registry` (nine
rows, each citing where its vendor facts were read), `GET/PUT /api/cli-keys`, and
`web/shared/domain/cliKeys.js` (the copy + the seam test). Pi's report is wrapped
rather than restated, `/api/pi-keys` is untouched, and the pane now talks only to
the envelope — the six keyboard rows of `qa-cli-settings.mjs` pass through it on
both apps, writing with one door and reading with the other.

Two things it added beyond the plan, both because the work asked for them:

- **`make keys-drift`** — the probe the P1 row named. It re-reads the installed
  pi's own `docs/keybindings.md` and compares ids *and* defaults with
  `pikeys.Catalog`, skipping loudly on a machine without pi. Run here against pi
  0.87.0: 90 actions, 10 unbound, no drift — and it fails on a deliberately
  broken catalog, so it is not a probe that always says yes.
- **A tenth registry row's worth of honesty**: a CLI whose editor has not shipped
  answers with `state` and `writable: false` and *no* file and *no* actions. The
  pane can then say "Codex keeps its own key map; PiCode's editor for it has not
  shipped yet" instead of promising one, which is what P5 was going to have to
  invent copy for anyway.

## P2a shipped (2026-09-21, `feat/keyboard-guests`)

The pane now answers for every CLI the registry knows, which is P5 of this plan
plus the half of P2 that needed no writer. A row whose map PiCode can edit — Pi
today — falls through to its own editor unchanged; every other row draws its own
answer from `blockNote()`: one state line and one action, and the pickup sentence
only when there is a map to pick up. `CliKeyboard.jsx` (one file per app, the
same markup) is the whole of it; `noteIsExternal()` picks how the action opens,
and the harness asserts both kinds — a vendor's page in a new tab, an in-app
route in the tab — because a note whose action does nothing is not an action.

With it: **Hermes stopped being a "wait"**. It keeps no key map file at all, so
its three rebindable keys are declared rows in its Settings pane (group
**Keyboard**), with defaults and value ranges read from the CLI's own
`hermes_cli/config_defaults.py`: `voice.record_key` (`ctrl+b`), `copy_shortcut`
(`auto | ctrl_c | ctrl_shift_c | disabled`), `display.busy_input_mode`
(`interrupt | queue | steer`). Its registry row stays `planned` — there is
nothing for a key-map writer to do — and `Keymap: Partial` is what says so; the
refusal names it too ("keeps no key map file; the keys it does allow are in
Settings"). One state vocabulary, two facts told apart by shape.

The placeholder sentence `"…are in development — coming soon"` is gone from this
pane. It promised a feature instead of describing the CLI, and it was wrong for
seven of the nine rows the moment the registry existed.

Two harness findings while proving it, both in `scripts/qa-cli-settings.mjs`:
a stale assertion the P0 period fix had invalidated (`/keybindings\.json$/` could
never match `/keybindings.json.`), and the app's toast stack sitting on the
phone's bottom edge at capture time — the guest captures now wait for it to
leave, which also let two rows recorded red on 2026-09-20 (the mobile trust
matrix) pass. One red row remains, the blocked-project-layer `ready()` named in
`docs/handoff/open/agent-clis-native.md`; it is not this pane's.

P2b is next: the adapters, Omp first, with the ADR above already accepted.

## P2b shipped (2026-09-21, `feat/omp-keymap`)

Omp's key map is editable, and the engine every other flat CLI will use is the
one it was built on. The four primitives the plan asked for were exported from
`clisettings` **unchanged in behaviour** (`internal/clisettings/keymap.go`):
`OpenDoc`, `Strings`, `SetStrings`/`RemoveKey`, `Save`. What is new there is one
piece of syntax — a *list* literal, which the settings engine never writes — and
the refusals around it: a row held as a table is `ErrShape` on read (reported as
`unreadable`, dropped from the pane's list, refused on write) and TOML has no
literal at all, so the format is refused rather than half-supported.

On top of it, `internal/clikeys` gained the flat engine (`flat.go`): a
declaration says where a CLI's file lives, its catalog and its platform
vocabulary; `ReadFlat`, `WriteFlat` and `ResetFlat` do the rest, with the
revision the caller read travelling with every write. The envelope grew
`revision`, `unreadable` and the CLI's own `platform`, and its `actions` are now
`clikeys.Action` — pi's catalog is converted in one place rather than joining the
two packages, because `pikeys` also answers `/api/pi-keys` from its own type.

The Omp declaration is `omp_catalog.go` (70 rows: 32 `tui.*` + 38 `app.*`, each
label the CLI's own description, the five it ships unbound, and the one row that
binds differently by platform) plus `ompKeyMapFile`: `keybindings.yml`, else
`.yaml`, else the legacy `.json`, in the agent dir the CLI itself resolves
(`$PI_CONFIG_DIR` or `~/.omp`, then `profiles/$OMP_PROFILE` or `$PI_PROFILE`,
then `agent`) — so PiCode edits the file the CLI reads, and never creates a
second one beside it. `[]` unbinds; `Reset all` removes only rows the catalog
knows.

**One fact in "P2's inputs" turned out stronger than the research could prove at
the time: the pickup is `restart`, not unknown.** The bundle's keybinding manager
reads its files when it is created and nothing in the bundle calls its own
`reload()` — no watcher, no command — so a running session keeps the map it
started with. The registry row says so, and P1's invariant ("a shipped row must
say when a change lands") is what forced the measurement instead of an
`unknown` that would have satisfied nobody.

Measured on the fixture: `qa-cli-settings.mjs` renders Omp's 70 rows on both
apps, captures `Ctrl+Alt+O` into the first row, reads it back out of Omp's own
file, and proves `Reset all` hands it back. Before that, the engine's own tests:
create-the-file, the three-name precedence, a legacy JSON map edited in place, an
unknown action refused by name, a stale revision refused with 409, reset leaving
unknown keys and comments alone, and an unreadable row reported rather than
swallowed.

## P3 (Codex) shipped (2026-09-21, `feat/codex-agy-keymap`)

Codex's key map is editable, and the plan's own row about it was wrong in one
place: the shape is `[tui.keymap.<context>]` with `action = "chord"` **rows inside
the context's table**, not a `[tui.keymap.<context>.<action>]` table per action —
the latter is a TOML type error, and the vendor's generated schema says so. The
declaration is `FlatMap.Path` and nothing else: same engine, same document
primitives, one table per context.

Its catalog is 149 keys in 12 contexts, read out of the vendor's artifacts at
the tag the installed build pins (`codex-cli 0.155.1`): the generated `config.schema.json`
(the key list and the labels — the schema, not the runtime inventory, because the
inventory omits three `global.*` fallback keys the struct accepts and an unknown
key makes codex refuse the file at start, observed live),
the chords in `built_in_defaults()` plus `vim_search.rs`. A row's ID is
`<context>.<action>` — the vendor's own path minus its `tui.keymap` prefix —
because the bare name is not unique (`move_left` lives in the editor *and* in
`vim_normal`), which the catalog's own test caught on the first run.

Two things this slice added to the engine, both because Codex demanded them:

- **A vocabulary that is not the pane's.** Codex writes `ctrl-alt-m` where the
  pane captures `ctrl+alt+m`, and it validates its whole keymap at startup: a
  chord in the wrong spelling is a CLI that does not start. `FlatMap.Normalize`
  renders a captured chord into the file's own vocabulary (`pageup` → `page-up`,
  `-` → `minus`, `escape` → `esc`, `f1`..`f24`) and refuses the rest by name
  (`super+m`); a chord already in the file's spelling is checked and kept. The
  pane mirrors it for display (`formatChord(chord, vocab)`) and for the
  browser-reserved warning (`reservedChordOf`).
- **A multi-line TOML value is refused by name.** `tomlValueSpan` is line-based,
  so an array the user broke across lines has a one-line span: the splice is
  attempted in memory, undone, and reported as "written over several lines …
  edit it there" rather than reaching the CLI as a broken file. The YAML block
  case keeps working through the remove-and-reinsert path.

Verified: the engine's own tests (create the context table, keep other tables and
the settings keys intact, join an existing table, reset takes the emptied table,
the chord rendering and its refusals), the endpoint test (146 rows + 12 contexts
through the envelope, a write landing as `ctrl-alt-m` in the TOML, a named
refusal for `super+m`), and `qa-cli-settings.mjs`, whose guest block now loops
over Omp and Codex and proves each one's captured chord reaches its own file in
its own spelling.

**Antigravity shipped too (2026-09-21, `feat/agy-keymap`)**, and with it P3 is
complete: 36 actions in ten id namespaces, the vendor's override file written
through the same engine, labels twenty parts vendor / sixteen parts id-derived
(the two doc pages that match the installed build name twenty actions; the rest
they leave unnamed, and the catalog says which is which). Its pickup was
**measured in a live session**: the TUI keeps the map it loaded at start, so the
row says restart. **OpenCode shipped (2026-09-21, `feat/keymaps-p4`)**: 162 actions from the
loader's own Definitions table, written into the user `tui.json`'s `keybinds`
object (never the legacy `opencode.json` section, which the CLI migrates), the
vendor's "none" reading as unbound, user scope only — a project `tui.json` wins,
and the pane says which file it writes. Pickup: restart, from the loader's
source (the config is snapshotted at TUI start; the reload RPC does not cover
keybinds).

**Claude Code is the last row, and its gate was settled live before any writing:**
2.1.278 logs `KeybindingSetup initialized with 229 bindings` and `Watching for
changes to /home/goat/.claude/keybindings.json` — the feature is not gated off
on this build, and the pickup is hot-reload, observed. Its shape is inverted
(`bindings: [{context, bindings: {chord: action|null}}]`), which needs the
inverted adapter: splice a chord into the context block for the row's action,
drop entries whose value is that action, `null` as the unbind verb, and a
first-write that matches the vendor's own `/keybindings` template. 22 contexts,
~115 actions from the vendor's docs page. That adapter is the final slice.

## P2's inputs (measured 2026-09-21)

**Omp, read out of the installed bundle** (`@oh-my-pi/pi-coding-agent` 18.2.8,
which pins `@oh-my-pi/pi-tui` 18.2.8 — the inlined keybinding module, whose
published sources match the bundle string-for-string):

- **One file, machine-level**, in the agent dir (`~/.omp/agent`, `PI_CONFIG_DIR`,
  or the active profile's dir): `keybindings.yml`, else `keybindings.yaml`, else
  legacy `keybindings.json`. A legacy JSON file is read and then written back as
  YAML into `keybindings.yml`, so a JSON file never stays the live one. **The
  adapter edits whichever file exists, in that precedence order, and creates
  `keybindings.yml` when none does** — writing `.yml` beside an existing `.json`
  would shadow the user's own values.
- **Flat, no contexts**: `Record<action, chord | [chord] | undefined>`; chords
  are lowercase `mod+base`, modifiers ordered `ctrl, shift, alt, super`; reading
  is case-insensitive and folds `esc`→`escape`, `return`→`enter`; `[]` unbinds
  (the CLI prints `Disabled`); an absent key means "use the default".
- **70 action ids, not 116**: 32 `tui.*` + 38 `app.*`, every one carrying a
  `description` (which is the pane's label). The 116 in the first draft of this
  plan was a grep artifact — a regex over the bundle for dotted identifiers also
  matches settings keys and event names; the registry said 116 for a day and is
  corrected. The id namespace (`tui.editor`, `tui.input`, `tui.select`, `app`) is
  the only grouping the CLI has.
- **A platform default exists**: `app.clipboard.pasteImage` is `ctrl+v` on Linux,
  `ctrl+v`/`alt+v` on Windows, `ctrl+v`/`super+v` on macOS. The catalog's `Alt`
  map takes the CLI's own platform names (`linux`/`win32`/`darwin`), and WSL
  reports `linux` to the CLI.
- **Pickup: restart** (measured in P2b, 2026-09-21). `KeybindingsManager.reload()`
  exists and re-reads the files, but nothing in the bundle calls it and nothing
  watches the file — the manager reads them once, when it is created.
- **Where the tables live**: the bundle's module is one region of `dist/cli.js`
  (the minified bundle is ~26 MB, so cite the region, not a line per fact), and
  the same text is published at `@oh-my-pi/pi-tui@18.2.8/src/keybindings.ts` (the
  32-row TUI table, the manager) and `.../src/app-keybindings.ts` (the 38-row
  table, the path resolver, the parser, the display formatter). A drift probe for
  Omp reads those.

**What the first adapter is**: a flat-document reader/writer (read a decoded
document into `action → chords`, splice one action's list literal, insert when
absent, refuse when the key exists in a shape the format layer will not rewrite,
atomic write, revision/409) over the exported format layer; the Omp catalog; the
`omp` registry row moving from `planned` to `shipped`; the pane taking the CLI
from the route; and the harness rows for a guest pane. Then Hermes' three scalar
keys, which need no key-map engine at all.

**The other four** (`claude-code`, `codex`, `agy`, `opencode`) keep their
registry rows and their `planned` answer until each adapter lands, with the
shapes already named in §4: inverted contexts for Claude Code, nested tables for
Codex, `tui.json` with `<leader>` sequences for OpenCode, and Antigravity's flat
`id → [chord]` with its own key names.

**Claude Code's adapter owns one thing the others do not**: its file keys
*chords* to actions, so "add a key to this row" is a different edit — a
`bindings[]` block for that context gains or loses a chord key — which is why its
shape is declared apart rather than squeezed into the flat writer.

## 7. Verification

1. `make ci-scoped` while iterating, `make close` at the end, `make ci` on
   `main`.
2. Go: `pikeys` (the new row and the platform alternates), `clikeys` per
   adapter — golden read, surgical write (a seeded file's comments, key order
   and unknown keys survive), `[]` unbind, unknown action refused, vendor-
   reserved chord refused by name, stale revision 409, unparseable file never
   overwritten, `TestJSListMatchesTheKeyboardCatalog`, `keys-drift` on the
   installed CLIs.
3. Browser: extend `scripts/qa-cli-settings.mjs` (do not add a fourth script)
   — facets filter, `⌨` filter-by-key, capture commit + Esc cancel, conflict
   line, reserved-chord warning, Reset, `?tab=keys` redirect, both apps.
4. Screenshots read from `var/screenshots/keyboard-pane/`: Pi populated,
   filtered-empty, capture open, shared-key filter, unparseable file, a guest
   editor with contexts, Grok's one-liner; 1440 and 1024 desktop, 390 mobile;
   dark and light. `__picodeOverlayAudit()` ok in every capture; the visual
   card answered in the session note.
5. Measure in the DOM: rows ≤32px, no ragged right column, every control in
   a row at `--ctl-h`, pane height at 89 rows ≤ 3300px, no double scrollbar.

## 8. Out of scope

- PiCode's own chords (`AppKeys.jsx`, Preferences → Shortcuts) — already has
  its own store, its own reserved-chord guard and its own page.
- A chord-sequence recorder (refused in §2), an in-pane JSON editor, and
  launching a terminal from the pane.
- Hermes' full map, Omp's named profiles, and Antigravity's IDE keybindings
  (that is `keybindings.json` for the editor, not the CLI).
- Making the browser-reserved keys reachable (Keyboard Lock needs
  page-requested fullscreen; refused 2026-09-11, unchanged here).

## 9. Open questions (owner)

1. **Order**: P0 first (the pane the owner is looking at, and the visual
   language every CLI inherits) or the engine first? Recommended: P0 first.
2. **Grok and Muse**: one line + one action (recommended — no vendor data we
   would have to maintain) or a read-only reference table of their built-in
   maps?
3. **Sequences**: Claude and OpenCode carry two-key sequences. Show them
   read-only in v1 (recommended) or build the recorder now?
4. **`Reset all changed`**: one new field on `PUT /api/pi-keys` for the whole
   map (VS Code has it) or leave 89 rows as the only way back? Recommended:
   ship it behind a confirm, since it is the only bulk undo.

## What P0 shipped, and where it differs from this plan (2026-09-21)

The owner approved P0 alone, and answered the four questions as recommended:
the pane first; Grok and Muse get one line + one action; sequences stay
read-only; `Reset all` ships behind a confirm.

**Shipped** (branch `feat/keyboard-ui`, corrected in `feat/keyboard-row`): the
row anatomy (32px grid — 48px with a note — a keycap at its own `--kbd-h`
instead of `--ctl-h`, the row's own actions in a fixed column, revealed on
hover/`:focus-within` and always on a touch surface); the toolbar (filter,
**Find by key**, counting facets `Changed`/`Shared`/`Off`, the row count,
**Reset all**); the changed bar; the capture strip with the existing chips kept
in view and `aria-live` on the result; the reserved-chord warning; the platform
alternates; `Reset all` over a new `{resetAll: true}`; and the two catalog
fixes — `app.thinking.save` and nine `Alt` rows read out of pi's own docs —
held by `TestAltDeclaresEachPlatformItCovers` and
`TestAlternatesDifferFromTheBaseDefault`. The map is 90 actions in 12 groups
now, not 89.

**The row is four columns, not three** (owner's screenshot, 2026-09-21, then
re-measured): `label (bounded 9-18rem) | keycaps | note (flexible) | actions`,
each cell placed by `grid-column` rather than auto-flowed. Two defects the first
build shipped, both caught only by looking:

- **A leaked flex shorthand.** The base `.key-label` rule (still AppKeys') says
  `flex: 0 1 12rem`, a *width* basis where the label sat in a row. Inside the
  new `.key-name` column the same basis is a *height*: the label, its cell and
  with it every row measured 192-216px. The pane now resets it
  (`flex: none; padding-top: 0`).
- **Auto-flow shifting the actions.** With the note conditional, a row without
  one moved its actions into the flexible column, and a row with two notes
  pushed them onto the next line — rows 32 → 229px. Every cell is placed now.

Density is asserted from that day on (`qa-cli-settings.mjs`): no label cell over
one line, the action and its keycap on one line, the keycaps within 48px of the
label, and 90 rows under 96px each / 4 000px total on desktop (128/6 200 on the
phone). Measured: 32-90px rows, 3 199px of list, 12px between the label and the
first keycap; the phone 44-106px and 5 058px.

**The toolbar is one line, and only the filter shrinks.** Wrapping it put the
row count and `Reset all` on a near-empty second line the moment a facet gained
a count; `nowrap` alone then squeezed the facet labels at 1024px and on the
phone, which is worse. Everything but the filter is `flex: none` and the facet
group scrolls if the line still runs out; the phone wraps on purpose (the count
is dropped there — its chips carry it). Measured: 54px at 1365 and 1024 with all
three chips complete, 90px and three lines at 390px, no page overflow at either.

Differences from the plan above, and why:

- **The toolbar sticks; the group headers do not.** The bar pins to the top of
  the CLI page's own scroller (`#agent-clis-view`, `overflow-y: auto`), so the
  filter and the facets stay reachable 90 rows down — verified in a scrolled
  capture (`top: 0`, audit ok). The first attempt was reverted on a bad
  measurement: a probe scrolled `window`, which this app never scrolls, and the
  bar "travelled with the content", so the rule looked inert. It was not; the
  note is kept here so nobody re-derives the same wrong reason. Group headers
  would need a hard-coded offset equal to this bar's height, which changes when
  the bar wraps — refused for *that*, not for the scroller.
- **The pane's rules are scoped to `.key-pane`.** AppKeys (Preferences →
  Keyboard — PiCode's own chords) shares every one of these class names and keeps
  the older, more compact language; rewriting the base block would have
  redesigned a surface this plan does not own, so the new rules are prefixed
  under `.key-pane` and the base block stands exactly as it was. Two languages
  for one name is a cost, paid knowingly: unifying them is a later task, and it
  needs AppKeys' store (localStorage) to stop being the exception.
- **The facet is `Shared`, never `Conflicts`** (the plan argues why; this is the
  as-built name), and a row pi ships unbound says `Unbound` while a row the
  reader turned off says `Off`.
- **The capture commits on the keypress**, as it did before: the server refuses
  a chord it cannot use, so a rejection writes nothing. No Save button, no
  countdown (both refused in §2).
- **The mobile keycap is 28px, not 24px** — a touch row, with the row's actions
  always visible (`@media (hover: none)` covers the desktop's touch cases too).
- **The harness rows landed in `scripts/qa-cli-settings.mjs`'s keyboard block**
  as planned, and the script gained a `catch` that records the rows that passed
  when a later row fails — a failing run used to leave no record at all. The
  `-keyboard-reset` screenshot is not audited: a bottom-anchored toast whose
  exit animation is frozen by a throttled page reads as a clipped overlay that
  no reader ever sees, and the audited states (`-keyboard`, `-keyboard-off`,
  `-keyboard-clear`, `-keyboard-reset-all`) are the ones that matter.
- **Four stale rows elsewhere in that script were repaired** (they died in the
  native-settings landing, not here): `layerOf()` reads `[data-layer]` on the
  body wrapper, `#g-compact` is `#g-compactionEnabled`, the "unsupported CLI"
  row waits on the pane because every managed CLI now has an editor, and the
  audited captures wait for a bottom-anchored overlay to settle. The post-loop
  untrusted/trust/free-agent matrix still stops on a `ready()` precondition the
  blocked project layer does not meet; it is named as a debt in
  `docs/handoff/open/agent-clis-native.md` rather than re-pointed blind.

P1–P5 are unchanged; the envelope (§4) is still the next step.
