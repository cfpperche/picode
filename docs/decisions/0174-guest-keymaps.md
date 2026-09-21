# ADR-0174: guest key maps — writing other CLIs' key bindings

- **Status**: accepted (owner approved the keyboard plan's P2, 2026-09-21)
- **Date**: 2026-09-21
- **Boundary**: persistence — PiCode writes files inside another vendor's configuration directory, one action at a time, preserving everything it does not own; protocol — a new endpoint carries a value class the settings contract deliberately excludes (a list of chords per action, not a scalar); security model — the write targets the machine's own home directory under the same HOME-swapped rules as ADR-0163, never a workspace file the CLI has not declared.

## Context

The Keyboard pane was Pi's alone. `docs/plans/keyboard-pane.md` measured what
the other eight CLIs actually offer, on the versions installed here and in
their own documentation (2026-09-21):

| CLI | What it reads | Shape | Picks an edit up | PiCode |
|---|---|---|---|---|
| pi | `~/.pi/agent/keybindings.json` | `action → chord \| [chord]` | `/reload` or the next run | shipped (ADR-0101) |
| omp | `<agent dir>/keybindings.{yml,yaml,json}` | `action → chord \| [chord]`, 116 ids | unmeasured | planned |
| claude-code | `~/.claude/keybindings.json` | `bindings[]` of `{context, bindings:{chord: action \| null}}` — **inverted** | hot-reload (v2.1.18+) | planned |
| codex | `~/.codex/config.toml`, `[tui.keymap.<context>.<action>]` | `action → chord \| [chord]`, one table per context, `[]` unbinds | restart (its `/keymap` refreshes live) | planned |
| opencode | `~/.config/opencode/tui.json` (or a project one) | `keybinds: action → chord \| [chord]`, `<leader>` sequences, `"none"` disables | restart (unconfirmed) | planned |
| agy | `~/.gemini/antigravity-cli/keybindings.json` | `action → [chord]`, its own namespaces, `pgdown`-style key names | unmeasured | planned |
| hermes | `~/.hermes/config.yaml` | **partial**: three scalar keys, the rest hardcoded | unmeasured | planned |
| grok, muse | — | a built-in map, or a viewer that cannot write | — | refused |

So six of the nine keep a map a tool may write, in three document shapes, with
per-CLI chord vocabularies (`pageUp` in pi, `page-down` in codex, `pgdown` in
agy) and per-CLI unbind verbs (`[]`, `null`, `"none"`). The registry and the
read path shipped in P1 (`internal/clikeys`, `GET /api/cli-keys`); what this
decision fixes is the **write** contract, and the answer to one question: does a
key map ride the settings driver PiCode already has?

It cannot. `internal/clisettings` is **scalar-only by design** (ADR-0099 §5):
its field kinds are `bool | select | text | number`, and a key map is a list of
chords per action. That shape — plus the inverted contexts of Claude Code and
the nested tables of Codex, which the settings writer explicitly refuses as
"array of tables / elements no declared key can address" — is what makes the
settings writer safe, and stretching it would put the safety of every settings
write behind a value class it was never measured for.

## Decision

**A per-CLI key-map driver behind the same door.** `internal/clikeys` keeps the
registry P1 shipped (shape, pickup, contexts, state, vocabulary, and for every
row the source of those vendor facts) and gains, per CLI, a declaration: the
file(s) per scope, the document shape, the action catalog (id, label, group,
defaults, context, and where each default was read), the unbind verb, and the
reload sentence the pane prints. `GET /api/cli-keys` serves it; `PUT` writes it.

**One format layer, not two.** The reader and writer reuse `internal/clisettings`'
format primitives — decode, span-locate, splice-in-place, insert, remove,
atomic write, revision — by exporting them unchanged. A key-map write is
therefore held to the rules that already govern settings writes: the file is
parsed by the real parser, exactly one action's value is replaced by byte span,
everything the writer does not understand survives byte-for-byte, the result
must parse before it is written, the write is atomic (tmp + rename), the mode is
preserved, and a file that changed underneath the read answers **409** and is
not touched.

**Writes are named, one action at a time.** An action the catalog does not know
is refused by name. A chord the CLI cannot express — a spelling outside that
CLI's vocabulary, a chord the vendor reserves, a plain `Escape` — is refused
with the reason, because a written key the CLI ignores is worse than a refusal.
A row the file carries and the catalog does not know is **shown read-only**:
never hidden, never dropped by a write (the rule settings already keeps for
unknown keys), so a vendor's newer action is visible instead of silently
rewritten.

**Two kinds of refusal, told apart.** A CLI whose map exists but whose adapter
has not shipped is `planned` and answers "PiCode cannot write <CLI>'s key map
yet"; a CLI that does not allow remapping is `refused` and answers "<CLI> does
not allow its keys to be remapped". The pane renders the second as a fact about
the vendor and the first as a fact about us, each with one action.

**Every catalog cites its source, and drift is probed.** A catalog row is a
claim about software PiCode does not own, so each declared default names where
it was read (pi's own `docs/keybindings.md`, omp's installed bundle, agy's
shipped file) and `make keys-drift` re-reads the installed artifact when it is
present, failing on a missing action, an unknown extra, or a changed default.
Without pi installed it says so and exits 0, so no gate depends on a vendor
package being present.

**Pi is untouched.** `/api/pi-keys`, `internal/pikeys` and pi's own catalog stay
what ADR-0101 made them; the envelope wraps that report rather than restating
it. This decision amends only the registry line that said no guest CLI exposes a
key map PiCode can write — which was true when it was written and is false for
five CLIs today.

**Refused in this decision.** PiCode does not normalise one chord vocabulary
into another (a translated key is a key that binds differently than it reads);
it does not write a whole file from the pane (import/export of a key map is a
second writer on one surface, and the file is already openable in Files); it
does not drive the vendor's own editor (`codex /keymap` and `claude
/keybindings` are interactive); and it does not write a map for a CLI whose
format it has not read — a `planned` row is the honest answer, not a guess.

## Consequences

One Keyboard pane serves nine CLIs, with coverage that stays uneven by design:
grok and muse refuse, hermes has three keys and no map, and a CLI stays
`planned` until someone reads its file. Each new CLI costs a declaration, a
catalog with sources, a golden-file test that writes and re-reads, and its row
in `make keys-drift`; a vendor that renames an action or changes a default shows
up as an unknown row or a drift failure, visibly, rather than as a key that
quietly stopped working.

PiCode becomes a second writer of files the CLI itself writes. The
revision/409 contract is load-bearing and the pane must say when an edit lands
(`/reload`, a restart, or never documented), because a user who binds a key and
sees nothing happen will otherwise conclude the pane is broken. A CLI mid-turn
holds its map in memory, so a write is applied at the CLI's own pickup point —
which is why the pickup sentence is part of the declaration and not a friendly
afterthought.

The vocabulary rule has a visible cost: the composer refuses chords it cannot
express in that CLI's spelling, and a user who knows the key works in their
terminal may be told PiCode cannot write it. That refusal is the point — the
alternative is a file that looks right and does nothing.

## Alternatives considered

- **Extend the settings descriptor with a `keys` kind.** Rejected: that
  descriptor is scalar-only by design (ADR-0099 §5) and its writer refuses
  exactly the structures a key map needs (arrays of tables, list values). Every
  settings write would then be validated by a rule written for a different value
  class.
- **A hand-written driver per CLI, sharing nothing (the `connectors` model).**
  Rejected: the three shapes share the same splice, atomic-write, revision and
  refusal needs, and four copies of "refuse to overwrite a file the parser
  rejected" is how one of them quietly stops refusing.
- **Driving the vendor's own editor** (`codex /keymap`, `claude /keybindings`,
  `agy /keybindings`). Rejected: they are interactive TUIs, not scriptable, and
  the files they write are documented — driving a terminal UI to edit a JSON
  file PiCode can write directly adds a process boundary and a class of failure
  (a hung TUI) for nothing.
- **A canonical chord vocabulary, translated per CLI in the UI.** Rejected on
  the evidence above: the vocabularies disagree (`pageUp` / `page-down` /
  `pgdown`), and a translation layer makes the pane's display of a binding
  differ from the file. The CLI's own spelling is what the pane shows and
  writes.
- **Letting the pane edit the whole map (a JSON/YAML box).** Rejected by
  ADR-0099 §5 and the 2026-09-12 study: the file is the escape hatch and it is
  already openable in Files; a second competing writer inside the pane is not.
- **Waiting for every vendor to converge on one format.** Rejected: this is the
  registry ADR-0163 already refused to wait for — nine CLIs, three shapes, and
  the ones that differ are the ones users care about.
