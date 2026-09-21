# Study: keyboard-shortcut editors, and which agent CLIs let a tool write their keymap

- **Date:** 2026-09-21
- **Sources:** four web sweeps run on this date, plus live reads of the
  installed CLIs on the owner's machine — pi 0.86.1
  (`docs/keybindings.md`, 90 actions), omp 18.2.7 (`dist/cli.js` names
  `keybindings.yml`/`.yaml`/`.json`, 116 action ids), codex-cli 0.155.1,
  Claude Code 2.1.278, grok 1.0.40, opencode 1.18.31, and `muse`, `agy`,
  `hermes` (Antigravity's `~/.gemini/antigravity-cli/keybindings.json` is
  present with 37 actions, so that row needs no vendor guess).
- **In-house bars:** [benchmarks.md](../benchmarks.md) (UI/UX section, the
  one-control-height rule, the UI copy rule), the 2026-09-12 measurement of
  this same pane ([2026-09-12-cli-settings-ux.md](2026-09-12-cli-settings-ux.md)),
  ADR-0099, ADR-0101, ADR-0163, and `web/shared/domain/browserChord.js`
  (the reserved set, written "for honest UI copy").
- **Scope:** the Keyboard pane's layout and interaction, and the per-CLI
  keymap facts a guest editor must be built on. Not the writers' format
  details (ADR-0163's splice doctrine already covers those) and not
  PiCode's own chords.
- **Proposal:** [../plans/keyboard-pane.md](../plans/keyboard-pane.md)

## A. What the references do

| # | Source | Screen | Pattern | Why it reads well | Cost here |
|---|---|---|---|---|---|
| 1 | [VS Code](https://code.visualstudio.com/docs/configure/keybindings) — and **Cursor**, whose editor is this one | Keyboard Shortcuts | search + record-keys, table `Command / Keybinding / When / Source`, per-row pencil, right-click `Change · Add · Remove · Reset · Show Same Keybindings` | command left, keys right, context dimmed; the search box *is* the recorder (press a key to find rows) — zero new chrome | med: table + `Input` exist; needs a context line and a menu |
| 2 | VS Code, `Show Same Keybindings` | conflict | filters the list to every command sharing that key; the troubleshooting log shows which rule won | no modal: the filter **is** the explanation, and duplicates stay legitimate | low: filter action + warn count |
| 3 | [JetBrains](https://www.jetbrains.com/help/idea/configuring-keyboard-and-mouse-shortcuts.html) | Keymap | `Add Keyboard Shortcut` dialog, first + optional second stroke, conflict block naming the rivals, offer to remove the other assignment; bold predefined vs blue changed | the theft is explicit and reversible, and the presets teach that defaults are immutable | med: Radix dialog + capture + conflict list |
| 4 | [macOS](https://support.apple.com/guide/mac-help/keyboard-shortcuts-mchlp2262/mac) | Keyboard Shortcuts | categories left, rows right, checkbox + key field; double-click the field to record; **Restore Defaults** bottom-left | the checkbox mutes an action without unbinding it — the only "off, but remember" in the study; grouping navigates 100+ rows | low: a column — refused, pi has no enabled flag |
| 5 | [Raycast](https://manual.raycast.com/command-aliases-and-hotkeys) | hotkey recorder | inline cell: live chips, "press keys" placeholder, conflict in red **with the owner's name**, `Overwrite` or retry | the best capture affordance: no dialog, no lost context, and the collision names who else owns the key | med: cell capture + conflict query; the 1.5 s auto-save is refused |
| 6 | [Obsidian](https://help.obsidian.md/editing/hotkeys) | Hotkeys | search + `All / Assigned / Assigned by me`, `+`/`−` per row, per-hotkey restore | faceted chips beat VS Code's `@source:` type syntax: discoverable, one click, and *Assigned by me* is exactly "changed" | low: two chips |
| 7 | [Zed](https://zed.dev/docs/key-bindings) | Keymap | pencil/double-click/Enter; precedence documented as "deeper context, then user wins"; a pending-keystroke countdown for sequences | the countdown is the best "mid-sequence?" affordance found — the one thing a sequence recorder must have | refused in v1: sequences render read-only |
| 8 | [TanStack hotkeys recorder](https://tanstack.com/hotkeys/latest/docs/framework/react/guides/hotkey-recording) | recorder hook | `onRecord/onCancel(Esc)/onClear(Bs/Del)`, `ignoreInputs`, modifier-only waits, Enter confirms | the trap-avoidance rules, written down: Esc always cancels, blur never commits, one recorder at a time | low: a state machine, not a dependency |
| 9 | [Notion](https://www.notion.com/help/keyboard-shortcuts) · [Slack](https://slack.com/help/articles/201374536-Slack-keyboard-shortcuts) · [Figma](https://help.figma.com/hc/en-us/articles/5665442977431) | reference overlays | strong section headers; flat mono `Ctrl+Key`, no 3D keycaps; platform-aware columns | at 13px a plain mono chip with a hairline border reads as a key; depth and bevels read as a toy | low: a token class |
| 10 | Warp · [Ghostty](https://ghostty.org/docs/config/keybind) · Kitty · WezTerm · tmux · Neovim · Emacs | config-file keymaps | every one of them is a file first (`keybind =`, `bind-key`, `:map`) with a **query verb** (`ghostty +show-keys`, `tmux list-keys`, `:verbose map`, `C-h k`) | the file is the truth and the query is the proof — the reason this pane shows rows the catalog does not know | shapes §4's drift rule |

### Adapt these five

1. **Inline capture cell** (Raycast + TanStack): the row's key area becomes the
   recorder — `Press a key` swaps in beside the existing chips, accent ring
   (`--accent` border + `--accent-soft` wash), 11px hint
   `Esc cancels · plain letters need a modifier`, one recorder at a time,
   `preventDefault + stopPropagation`, `aria-live` on the result, focus back
   to the row. Commit on the keypress; the server refuses a chord it cannot
   use, so nothing is written on a rejection. **No Save button and no
   countdown**: the write is one reversible row, and every other row-level
   control in this app (the settings switches) saves on change.
2. **Modified gutter bar** (VS Code), not bold-for-changed (JetBrains): a 2px
   accent bar at the row's leading edge when the effective keys differ from
   the CLI's default, plus a per-row `Reset to default` revealed on hover /
   `:focus-within`. Bold halates and blue text loses contrast at 13px in dark
   mode; the bar is the same language the guest settings rows already use for
   "set in this file" (`agent-clis.css:220`).
3. **Conflict inline, never a banner** (JetBrains + VS Code): on capture of a
   taken key, one line under the row — `Also on Next prompt, Toggle path.` with
   `Show` (filter the pane to the rivals) and `Keep both`. Persisted
   duplicates become the `Shared` facet, not a warning: pi's contexts
   legitimately overlap, and for CLIs that declare contexts the same facet is
   a real conflict and gets the warn colour. Never a full-width banner (it
   separates the cause from the rivals) and never a blocking modal (it blocks
   a legitimate duplicate).
4. **Faceted chips** (Obsidian) over type-syntax filters (VS Code): `All ·
   Changed · Shared · Off`, single-select, a facet whose count is 0 not drawn.
5. **Sticky group headers + plain mono chips** (Notion + macOS's grouping):
   11px uppercase `--text-secondary` headers with a count, rows on a 4px grid,
   and a keycap as its own token class — mono 12px, 22px tall, 1px
   `--border-strong`, 4px radius, `2px 6px` padding (`app.css:417`, the
   `.hotkey-row kbd` treatment), never `--ctl-h`.

### Refuse these

| Refused | Reason |
|---|---|
| Chord-sequence recording | needs timeouts, prefix resolution and a pending-state UI (Zed's countdown); Pi has no sequences, and the two guests that do (Claude, OpenCode) can show them read-only until someone needs more |
| Raycast's auto-save countdown | a timer commits for the reader; a keypress plus a server-side refusal is both faster and safer |
| Backspace-to-clear inside a capture | pi binds `backspace` (`deleteCharBackward`), so the key being recorded is the key that would clear it. `Unbind` is the explicit row action instead |
| Plain `Escape` as a recordable chord | every recorder in the study treats Esc as exit; `app.interrupt` still *defaults* to `escape`, it just cannot be re-recorded |
| An in-pane JSON editor (VS Code's `{}`, Zed's `keymap.json`) | two writers on one file inside one surface; the file is already openable in Files, and ADR-0163's line about not writing what we do not own still holds |
| A read-only "full key reference" table for Grok and Muse | it would be hand-maintained vendor data that drifts silently — the failure `TestEveryBooleanDeclaresItsDefault` exists to catch for settings |
| Copying Grok's or PiCode's own map into a guest pane's **defaults** | a default is a claim about someone else's software; §4 declares where each one was read from |
| Bold/blue-for-changed, 3D keycaps, no-confirm Restore Defaults, conflict banners | contrast/halation at 13px, visual weight against flat tokens, irreversible bulk action, and a cause separated from its rivals |

## B. The per-CLI landscape (what a guest editor can stand on)

| CLI | Keymap | File / section | Value shape | Pickup | Source |
|---|---|---|---|---|---|
| `pi` | full | `~/.pi/agent/keybindings.json` | `id → string \| string[]`; per-id override | `/reload` or next run | installed `docs/keybindings.md`; `internal/pikeys` |
| `omp` | full | `~/.omp/agent/keybindings.{yml,yaml,json}` | `id → string \| string[]` | UNCONFIRMED | installed bundle (one constant names the three files); 116 ids |
| `agy` (Antigravity CLI) | full | `~/.gemini/antigravity-cli/keybindings.json` | `id → string[]`; `[]` disables; invalid entries fall back per key | UNCONFIRMED | [CLI settings](https://antigravity.google/docs/cli/settings); the shipped file |
| `codex` | full, by context | `~/.codex/config.toml` → `[tui.keymap.<context>.<action>]` | `action → string \| string[]`; `[]` unbinds; one key on two actions in one surface is a config error | restart (its `/keymap` refreshes live) | [config reference](https://developers.openai.com/codex/config-reference), [config basics](https://developers.openai.com/codex/config-basic) |
| `opencode` | full | `~/.config/opencode/tui.json` (+ project `tui.json`, `$OPENCODE_TUI_CONFIG`) | `keybinds: action → string \| string[] \| object`; comma-separated alternatives; `<leader>` sequences with `leader_timeout`; `"none"`/`false` disables | restart (UNCONFIRMED for the keybind path) | [keybinds](https://opencode.ai/docs/keybinds) |
| `claude-code` | full, inverted, by context | `~/.claude/keybindings.json` (global only) | `bindings: [{context, bindings: {chord: action \| null}}]`; chords may be space-separated sequences | **hot-reload** (v2.1.18+) | [keybindings](https://code.claude.com/docs/en/keybindings) |
| `hermes` | partial — 3 keys | `~/.hermes/config.yaml` → `voice.record_key`, `copy_shortcut`, `display.busy_input_mode` | single key-name strings | restart (UNCONFIRMED) | [voice guide](https://hermes-agent.nousresearch.com/docs/guides/use-voice-mode-with-hermes); [issue #4256](https://github.com/NousResearch/hermes-agent/issues/4256) |
| `grok` | **none** | — | — | — | [keyboard shortcuts](https://docs.x.ai/build/keyboard-shortcuts): *"built in and cannot currently be remapped"*; `~/.grok/config.toml` has no keymap section on this machine |
| `muse` | **none** | — | — | — | [interactive mode](https://ai.developer.meta.com/docs/muse-code/interactive): `/keymap` is view-only; `settings.json` carries no keymap key |

Two facts the pane must not flatten:

- **No canonical chord spelling exists.** pi says `pageUp`, Codex
  `page-down`, Antigravity `pgdown` and `ctrl+_`, OpenCode `<leader>n`, Claude
  `ctrl+k ctrl+s`. A per-CLI vocabulary is a declaration, not a formatting
  detail.
- **Vendor action lists grow under us.** pi documents 90 actions to the pane's
  89; nine of pi's defaults carry a platform alternate we currently flatten
  (`tui.editor.undo` is `alt+z` on WSL, `app.suspend` has no binding on native
  Windows); Omp carries 116 ids; Antigravity's catalog is a shipped file whose
  namespaces are its own. Hence the drift probe and the rule that a row the
  catalog does not know is shown, marked and read-only rather than hidden.

## What follows

The proposal — the pane's redesign, the four CLI groups, the engine that
owns the declarations, and the phases — lives in
[../plans/keyboard-pane.md](../plans/keyboard-pane.md).
