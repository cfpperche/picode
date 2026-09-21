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
| **P2 — the engine + Omp + Hermes** | `internal/clikeys` (declaration, adapters, refusals, `/api/cli-keys`), primitives exported from `clisettings`, `omp` full editor (same shape as pi), `hermes` three rows | `internal/clikeys/*`, `internal/clisettings/*` (export only), `internal/server/cli_native.go`, `web/shared/domain/cliLaunch.js` (`cliPanes`), both `CliSettings.jsx`, new `KeyboardPane.jsx` |
| **P3 — Codex + Antigravity** | Nested TOML and flat JSON adapters; catalogs from the vendor schema and from the shipped file; restart/live notes measured | `internal/clikeys/codex.go`, `agy.go`, their golden files |
| **P4 — Claude Code + OpenCode** | Inverted-context adapter (context column + context facet), `tui.json` with read-only `<leader>` sequences and the project layer | `internal/clikeys/claude.go`, `opencode.go` |
| **P5 — the honest non-editors** | Grok and Muse: one line + one action; the placeholder sentence `"…are in development — coming soon."` disappears from the codebase | both `CliSettings.jsx`, `docs-site/guide/settings.md` |

Each phase ends with the rite; P0 ships alone and is the one the owner can
judge by eye.

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
