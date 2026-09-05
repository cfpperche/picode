# Diff panel for pi

See what the agent changed without leaving its terminal: `/diff` opens a
side panel with every file changed against `HEAD` and the hunks of the file
it touched last.

`pi-diff` is an **optional pi package — an extension, not part of PiCode
core**. It talks to git inside the pi session and never to PiCode. Without
it nothing changes; with it, every pi terminal tab (and a plain `pi` outside
PiCode) has the panel.

Install `packages/pi-diff` from the PiCode repository. Guide for install
targets: [Packages](/guide/packages).

```sh
pi install /path/to/picode/packages/pi-diff
```

## Using it

| You type | What happens |
|---|---|
| `/diff` | Toggle the panel. It stays out of the way: the editor keeps focus |
| `/diff hero.tsx` | Show that file's hunks (an exact path or a unique tail) |
| `/diff next` / `/diff prev` | Move the focus through the list |
| `/diff top` | Back to the first hunk |
| `alt+n` / `alt+u` | Scroll the hunks three lines |
| `alt+pageDown` / `alt+pageUp` | Scroll a page |

The list has one row per tracked change (staged or not) and per untracked
file, with `+added -removed`, `+n new` or `bin`. The focus follows the
agent's last `edit`, `write` or `multiedit`. Hunks are numbered like the
editor: new line numbers for kept and added lines, old ones for removed.

The panel refreshes when a mutating tool finishes and at the end of every
turn. The footer always shows the total, `diff +324 -259 · /diff`, even
with the panel closed.

## Where it runs

| | What you get |
|---|---|
| **Pi TUI** (terminal, PiCode terminal tabs included) | The panel, the commands, the shortcuts, the footer total |
| **PiCode chat** (`--mode rpc`) | The footer total only; use the Files and Changes pane for the diff |
| **Narrow terminals** (under 100 columns) | The panel hides itself and comes back when there is room |

## Limits

- Always the working tree against `HEAD`; a repository with no commit
  lists everything as new.
- Diffs over 4000 lines are cut with a note; untracked files over 1 MB
  have no line count; binary files show `bin`.
- Edits made outside the session show at the next refresh or the next
  `/diff`.
