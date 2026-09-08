# Study: the right-click menu for a terminal pane

- **Date:** 2026-09-07
- **Owner request:** give terminals the PiCode context menu the rest of the
  app already has; keep the modifier bypass for the browser's own menu; move
  the attach bar behind it, with a close button that gives the pane the whole
  editor back. Propose the rest of the rows from the benchmarks.
- **Scope:** the pane of a terminal (shell or Agent CLI) and of an agent's
  TUI. Not the tab strip, not the sidebar rows — those already have menus.

## What is true in the repo today

`App.jsx` carried one generic menu (Copy, Paste, Reload, Toggle theme) and
skipped terminals outright: `if (e.target.closest(".xterm, .term-pane"))
return; // terminals: untouched for now`. The bypass modifier already existed
and is configurable in Settings (`lib/contextMenuPrefs.js`, default Shift).
The attach bar (ADR-0089) was mounted for the whole life of a running CLI
terminal, ~90 px the pane never got back; mobile already opened the same
door on demand as a sheet (`TermAttachSheet.jsx`).

Ctrl+click already opens a path or URL under the cursor
(`web/shared/domain/termLinks.js`) — undiscoverable behind a modifier.
`xtermOptions()` already sets `rightClickSelectsWord`.

## Benchmarks

| Source | Fact | Take / refuse |
|---|---|---|
| VS Code | `terminal.integrated.rightClickBehavior`: `default` / `copyPaste` / `paste` / `selectWord` / `nothing`; menu carries Split, Kill, Move to New Window | A configurable modifier is ours already. Lifecycle rows belong in the pane's menu, not only in the sidebar |
| Windows Terminal | Copy, Paste, Find, Duplicate tab, Split pane, Web search, Close | Confirms pane + tab actions in one menu. Web search deferred |
| Ghostty | Copy, Paste, Select All, Split, Reset | "Select all" earns its row; Reset deferred (tmux owns the screen) |
| iTerm2 | Right-click performs **smart selection** at the cursor and matching rules add actions ("Open URL/Semantic History", "Send Text") | **The pattern we took:** act on the token under the cursor, not only on a prior selection. Our `termLinks` already resolves it |
| Warp | Block menu: copy command / output / both, bookmark, filter, search in block, share | Copying *output* needs shell integration to know where a command began — refused for v1, not faked |
| Cursor | "Add to Chat" / "Debug with AI" over terminal output | **The row that makes this ours:** the selection goes to the CLI already running in that pane, through the ADR-0089 door |
| VS Code + Copilot | `@terminal`, `/explain` over a selected error | Same direction; we keep it inside the pane's own agent |

## Decision

One menu, built from what the pane can do (`lib/termMenu.js`, pure and
tested): clipboard rows always; the CLI door (`Ask <cli> about this`,
`Attach files…`) only while a launched CLI runs; `Open <token>` only when the
cursor sits on one; `Clear` only on a bare shell — a TUI owns its screen;
rename/settings/files/close/remove only for a terminal of its own, never for
an agent's TUI pane.

A selection handed to the CLI takes one of two shapes:

| Selection | What the bar receives |
|---|---|
| one line, ≤ 400 chars | the message input, pre-filled |
| multi-line or longer | staged as `selection.txt` and attached by path |
| empty | no `Ask` row at all |

**The right button belongs to PiCode.** xterm forwards a right press to the
pane as a mouse report and tmux answers with its own menu drawn inside the
terminal — two menus over one click. The press stops at a capture-phase
listener; only `contextmenu` continues, which is where xterm still selects
the word under the cursor and where the bypass modifier hands over to the
browser. That press is also where the selection is read: a guest with mouse
reporting on (every agent TUI) makes xterm drop it before any
`contextmenu` listener runs.

## Refuse

| Temptation | Why not |
|---|---|
| Split pane | We have no panes; the tab is the unit (tab strip, ADR-0030) |
| "Copy last output" | Needs shell integration to know where a command started; a guess would be a lie |
| Find in the pane | Wants `@xterm/addon-search` — a dependency and a search overlay. Phase 2, priced separately |
| `tmux clear-history` | A server route for a courtesy; Ctrl-L is what the user would type |
| Reset terminal | The tmux pane owns the screen; a local reset only desynchronises the view |
