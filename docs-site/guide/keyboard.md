# Keyboard and browser shortcuts

PiCode runs in a browser tab, and the browser keeps some key combinations
for itself: `Ctrl+T` opens a new tab, `Ctrl+W` closes it, and no web page
can change that in a normal window. Agent CLIs (Codex, Claude Code, Grok,
OpenCode, Hermes, pi) use those same keys — Codex, for example, opens its
MCP details with `Ctrl+T`. This page shows what works where, and how to
give every key to your agent.

## What the browser takes in a normal window

In a normal (non-fullscreen) window these combinations never reach PiCode —
the browser acts on them before the page sees anything. On Windows and
Linux the browser keys use `Ctrl`; on macOS the same slots use `Cmd`
(`Cmd+T`, `Cmd+W`, …) and PiCode's `Ctrl` chords reach the page as usual:

| Keys | Browser action |
|---|---|
| `Ctrl+T` / `Ctrl+Shift+T` | New tab / reopen closed tab |
| `Ctrl+W` / `Ctrl+Shift+W` | Close tab / close window |
| `Ctrl+N` / `Ctrl+Shift+N` | New window / incognito window |
| `Ctrl+Tab` / `Ctrl+Shift+Tab` | Next / previous tab |
| `Ctrl+1` … `Ctrl+9` | Jump to tab |
| `Ctrl+PgUp` / `Ctrl+PgDn` | Previous / next tab |

Everything else — `Ctrl+C`, `Ctrl+F`, `Ctrl+P`, `Alt+←`, `F3`, … — reaches
the page, and inside a terminal pane it goes straight to the CLI running
there. Copy, interrupt, line-editing keys and function keys already work.

## Fullscreen mode gives every key to the page

Enter **Fullscreen** (`Ctrl+Shift+Enter`, the right-click menu, or the
command palette) and PiCode asks the browser to hand over its reserved
keys. The terminal keeps all of them:

- `Ctrl+T` opens the Codex MCP details instead of a new tab.
- `Ctrl+W` reaches the CLI instead of closing the tab.
- `Escape` keeps doing its job inside the terminal (vim, pager, TUI menus).

Two ways out remain, on purpose:

| Gesture | What it does |
|---|---|
| `Escape` (outside a terminal) | Leaves fullscreen mode |
| Press and **hold** `Escape` (~2 s) | The browser's own exit — always works |

Leaving the mode returns the keys to the browser.

## Browser support

Handing over reserved keys needs the Keyboard Lock API while the page is
fullscreen. Chrome, Edge and Opera support it. Firefox and Safari do not:
fullscreen mode still hides the chrome, but the browser keeps its
shortcuts until those browsers ship the API. Surfaces that cannot take
the lock (for example pages embedded outside a top-level tab) degrade the
same way, and on macOS capture of the `Cmd` keys follows Chrome's own
implementation.

## Rebinding PiCode's own shortcuts

PiCode's own chords stay out of the browser's reserved set so they work
everywhere: `Ctrl+K` for the command palette, `` Ctrl+` `` for a new
terminal, `Alt+[` / `Alt+]` to cycle tabs. Rebind them in **Settings →
Keyboard**, and the agent's own key map in **Settings → Keys**.

| Keys | Action |
|---|---|
| `Ctrl+K` | Command palette |
| `` Ctrl+` `` | New terminal |
| `Ctrl+.` | Toggle inspector |
| `Ctrl+Shift+F` | Find in terminal |
| `Ctrl+Shift+Enter` | Fullscreen mode |
| `Alt+[` / `Alt+]` | Previous / next tab |
| `Alt+W` | Close tab |

In a terminal pane: `Ctrl+C` copies a selection or interrupts, `Ctrl+V`
pastes, `Ctrl+Shift+C` / `Ctrl+Shift+V` always copy / paste, and
`Shift+Esc` hands the keyboard back to the app from a Canvas panel.
