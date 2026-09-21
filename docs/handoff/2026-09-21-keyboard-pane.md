# 2026-09-21 — feat/keyboard-pane: the Keyboard pane plan (redesign, then guest keymaps)

Shipped: `docs/plans/keyboard-pane.md` (proposal — nothing implemented) and
`docs/benchmarks/2026-09-21-keyboard-pane.md` (the study behind it). Planned
from measurement, not from memory: the pane holds **89 actions / 12 groups**
at `--ctl-h` (36px) chips with an Add button laid out after the chips (so its
column is ragged), **52 of 89 actions share a chord** with another (pi's
contexts overlap, which is why the plan calls the facet *Shared*, not
*Conflicts*), `app.thinking.save` is missing against pi's own 90, **9 defaults
carry a platform alternate** the pane flattens (`tui.editor.undo` is `alt+z`
on WSL; `app.suspend` has no binding on native Windows), and **5 defaults sit
on browser-reserved chords** (`browserChord.js` already answers with the
browser's own wording).
Verified: `make close` → `ci-scoped` PASS; docs-only diff, so no gate reads
the pane and none of the numbers above comes from a running app.
`visual-review: n/a` — this branch changes no surface; the redesign is judged
in P0, and this method's blind spot is exactly that: the plan is a reading of
the shipped code and the installed CLIs, not of a rendered pane.
Supersedes, pending implementation: `CliSettings.jsx:14-15` ("no guest CLI
exposes a key map PiCode can write") is false for `claude-code`, `codex`,
`opencode`, `agy` and `omp` at the versions on this machine — the topic file
keeps its line until the engine ships, and the plan says so in its header.
Also unverified and named as such: whether `omp`, `agy`, `opencode` and
`hermes` pick a keymap edit up live or need a restart (four UNCONFIRMED rows
in §4 that P2–P4 measure before writing the pane's reload sentence).
Merge: fast-forward ready.

## Next up

- Answer the four owner questions in `docs/plans/keyboard-pane.md` §9 (P0 before the engine, Grok/Muse treatment, sequences read-only, `reset all`), then start P0.
