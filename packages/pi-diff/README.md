# pi-diff

Side diff panel for the [pi](https://github.com/badlogic/pi-mono) TUI:
`/diff` opens a right-hand overlay with every file changed against `HEAD`
and the hunks of the file the agent touched last, refreshed after each
edit — the panel Claude Code shows, for pi (ADR-0077).

- **`/diff`** toggles the panel. It never takes keyboard focus: you keep
  typing in the editor while it stays open. It hides itself on terminals
  narrower than 100 columns and comes back when there is room.
- **The list** is every tracked change (`git diff HEAD`, staged or not)
  plus untracked files, one row each with `+added -removed`, `+n new` or
  `bin`. Renames show the new name and say where they came from.
- **The hunks** are the focused file's, numbered like the editor: new
  line numbers for kept and added lines, old ones for removed lines.
  Focus follows the agent — the last `edit`, `write` or `multiedit` —
  and you can override it: `/diff <path>` (an exact path, or a unique
  tail like `hero.tsx`), `/diff next`, `/diff prev`, `/diff top`.
- **Scroll** the hunks with `alt+n` / `alt+u` (three lines) and
  `alt+pageDown` / `alt+pageUp` (a page). `alt+↑/↓` and `alt+j/k` are
  pi's own shortcuts, so the panel does not take them.
- **The footer** always carries the total — `diff +324 -259 · /diff` —
  even with the panel closed, and `diff clean` when there is nothing.

The panel refreshes when a mutating tool (`edit`, `write`, `multiedit`,
`bash`, `powershell`) finishes and at the end of every turn. Changes made
outside the session (another agent, your editor) show up at the next
refresh or the next `/diff`.

## How it draws

The overlay goes straight through the TUI (`tui.showOverlay`, non-capturing),
not through `ctx.ui.custom()`. A custom component counts as a UI prompt in
pi's lifecycle events, so a panel that stayed open all session would tell
every host watching those events that pi is "waiting for user" — PiCode's
guest-TUI sensors included (ADR-0056). The widget slot `pi-diff` is the
handle pi owns for us: registering it shows the overlay, clearing it hides
the overlay.

## Where it runs

| | What you get |
|---|---|
| **Pi TUI** (terminal) | The panel, the commands, the shortcuts and the footer total |
| **`pi --mode rpc`** (PiCode chat) | The footer total only; `/diff` says the panel needs the terminal |
| **PiCode core** | Nothing — this package talks to git, never to PiCode |

## Install

```sh
pi install /path/to/picode/packages/pi-diff
```

or, for one project, add it to `.pi/settings.json` under `packages`.
Everything runs inside each pi session like any installed extension;
uninstalling restores pi's stock behavior exactly.

## Limits

- Diffs longer than 4000 lines are cut with a note; untracked files over
  1 MB are listed without a line count.
- Binary files show `bin` and no hunks.
- The comparison is always the working tree against `HEAD`. A repository
  with no commit yet lists everything as new.

## Development

```sh
node --test test/*.test.ts
```

`src/logic.ts` is pure (git output parsing, focus rules, the layout) and
fully covered by `test/`; `extensions/diff.ts` is the I/O glue (git,
overlay, events). Run it against a scratch repository with
`pi --no-extensions -e packages/pi-diff/extensions/diff.ts`.

MIT — see [LICENSE](LICENSE).
