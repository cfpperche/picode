# 2026-09-21 — feat/agy-keymap: Antigravity's key map, and P3 is complete

Shipped: **Antigravity's Keyboard tab**. 36 actions in the ten id namespaces its
own `keybindings.json` carries, chords kept in the file's spelling
(`ctrl+l`, `pgdown`, `esc`, `ctrl+_`), and PiCode writing that file through the
same flat engine as Omp — an *override* file, so removing a row hands the action
back to the binding built into the binary, which is exactly what the engine's
reset does. Labels: twenty from the two vendor doc pages that match the installed
build (`/docs/cli/using`, `/docs/cli/vim-editor-mode`), sixteen derived from the
id where those pages leave the action unnamed — the catalog header says which is
which.
**Pickup: measured, not assumed.** With the file remapped under a running `agy`
(`cli.cycle_mode` from `shift+tab` to `ctrl+n`, written through the real engine),
the TUI kept the map it loaded at start — the old key still cycled the mode, the
new one did nothing — so the row says restart. The user's keybindings file was
backed up and restored byte-identical (md5 verified).
Verified: `make ci-scoped` PASS; the engine's tests (36 rows, ten contiguous
groups, every action bound, the vendor's un-capturable chords kept, a write
rendering the pane's `escape` as the file's `esc`, reset handing the row back);
`qa-cli-settings.mjs`, whose guest block now covers Omp, Codex and Antigravity on
both apps — each captured chord reaching its own file in its own spelling, and
each Reset all handing it back. visual-review: PASS (pristine base, changed row,
and Reset-all dialog read on both apps; the pristine frame shows the honest empty
state, "not created yet").
Also: `desktop-shell/tools/mkicon.go` was unformatted at main's tip (fmt-check
red for every branch); a blank line after the build tag.
The live checks that started this turn are recorded in
`docs/handoff/open/agent-clis-native.md`: Omp applies a PiCode-written map
(observed), Codex reads and validates it (observed — and that check found the
three-row gap, fixed in `feat/codex-fallback-rows`), and one codex check remains
open (pressing a rebound key in a live TUI needs the real HOME).
Merge: fast-forward ready.

## Next up

- P4: Claude Code's inverted contexts (chord-keyed rows, hot-reload) and OpenCode's leader sequences with a project layer — the last two, and the only ones needing new pane work.
