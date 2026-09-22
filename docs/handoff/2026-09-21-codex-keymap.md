# 2026-09-21 — feat/codex-agy-keymap: Codex's key map is editable (P3, Codex half)

Shipped: **Codex's Keyboard tab**. 146 actions in the twelve contexts Codex puts
them in — global, chat, composer, editor, the four vim modes, pager, list, agents
and approval — every label the vendor's own schema carries, every default chord
from the vendor's own table, read out of the artifacts at the tag the installed
build pins (`codex-cli 0.155.1`). PiCode writes `[tui.keymap.<context>]` tables in
`~/.codex/config.toml` — the same file the Settings tab edits, through the same
document primitives — and leaves every key it does not manage alone. The plan was
wrong about the shape and this is the correction: rows are `action = "chord"`
*inside* the context's table, not a table per action, which would be a TOML type
error.

Two things the engine gained, both because Codex demanded them:

- **`FlatMap.Normalize`**, a per-CLI chord vocabulary for the write. Codex spells
  `ctrl-alt-m` where the pane captures `ctrl+alt+m`, and it validates its whole
  keymap at startup: a wrong spelling is a CLI that does not start. A captured
  chord is rendered into the file's vocabulary (`pageup` → `page-up`, `-` →
  `minus`, `escape` → `esc`, `f1`..`f24`) and anything Codex cannot express is
  refused by name (`super+m`). The pane mirrors it: `formatChord(chord, vocab)`
  keeps the CLI's separator in the display, and `reservedChordOf(chord, vocab)`
  recognises codex's spelling when warning that a browser eats a chord.
- **A multi-line TOML value refused by name.** `tomlValueSpan` is line-based, so
  an array the user broke across lines had a one-line span; the splice is now
  attempted in memory, undone, and reported as "written over several lines … edit
  it there" rather than reaching the CLI broken. YAML's block case keeps working
  through the remove-and-reinsert path.

One real bug of mine the harness caught: `ResetFlat` looked rows up by their id
instead of the declaration's path, so `Reset all` silently left every nested row
in place. Fixed, and the codex test now covers it.
Verified: `make ci-scoped` PASS; the engine's tests (context table created,
sibling contexts and settings keys intact, reset takes an emptied table, the
chord rendering and its refusals at f24/f25 and `super`), the endpoint test (146
rows + 12 contexts, a write landing as `ctrl-alt-m`, a named refusal), and
`qa-cli-settings.mjs`, whose guest block now loops over Omp and Codex on both
apps and proves each captured chord reaches its own file in its own spelling.
The **phone layout needed fixing and the visual review is what found it**: at
390px the desktop row grid squeezed the keycap track until a 75px chip sat in a
3.4px column and drew over the label — invisible with pi's short labels, obvious
with codex's sentences. The rule that governs a phone is
`#m-app .m-more-page .key-row` in `mobile-settings.css` (an id beats the pane's
own class rule, which is why editing `app.css` changed nothing): its keycap track
is now `minmax(min-content, max-content)`, so it cannot collapse under its chips,
and the label takes the flexible track. Measured after: zero overlapping rows,
75px track for a 75px chip. The desktop app gets the same form under 620px.
visual-review: PASS (five captures read after the fix — codex's three on both
apps plus pi's two as the regression check; card 5/5; the remaining notes are
cosmetic: tight leading between two stacked notes, a long label wrapping to four
lines, and the frame-edge clipping every list has).
Not done: **Antigravity**, the other half of the P3 row. Its research is complete
(36 ids, `id -> [chord]`, one override file, `[]` disables a default, removing a
row returns to the vendor's default) and it needs no new engine; what it needs is
its pickup measured and its catalog's labels settled, because the vendor publishes
two documentation generations and only `/docs/cli/using` + `/docs/cli/vim-editor-mode`
match the installed build. That plan is in `docs/plans/keyboard-pane.md` §P3 and
`docs/handoff/open/agent-clis-native.md`.
Merge: fast-forward ready.

## Next up

- Antigravity's key map (the rest of P3): measure the pickup by remapping `cli.cycle_mode` in a live `agy` session and watching the footer, take the catalog from `/docs/cli/using` + `/docs/cli/vim-editor-mode` (the two pages that match the installed build, not the drifted reference), and declare it as a flat map into the engine P2b/P3 left. Research: `docs/plans/keyboard-pane.md` "P3 (Codex) shipped" + scout findings in `docs/handoff/open/agent-clis-native.md`.
