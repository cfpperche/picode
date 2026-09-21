---
description: What the browser keeps, what PiCode can bind, and what to do about the overlap.
---

# Keyboard and browser shortcuts

PiCode runs in a browser tab, and the browser keeps some key combinations
for itself: `Ctrl+T` opens a new tab, `Ctrl+W` closes it, and no web page
can change that. Agent CLIs (Codex, Claude Code, Grok, OpenCode, Hermes,
pi) use those same keys — Codex, for example, opens its MCP details with
`Ctrl+T`.

- **Where:** this page is the map. Remap PiCode chords in **Agent CLIs → Keyboard**.
- **Not this:** PiCode cannot steal `Ctrl+T` or `Ctrl+W` from the browser. Those never reach the page.

## What the browser keeps for itself

These combinations never reach PiCode — the browser acts on them before
the page sees anything. On Windows and Linux the browser keys use `Ctrl`;
on macOS the same slots use `Cmd` (`Cmd+T`, `Cmd+W`, …) and PiCode's
`Ctrl` chords reach the page as usual:

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

The agent's own map (**Agent CLIs → that CLI → Keyboard**) marks every row
that uses a chord from the reserved list above, so a binding that can never
fire in a browser tab is visible rather than mysterious — five of pi's own
defaults are in that list today.

Each CLI's Keyboard tab says what *its* map is, because they are not the same
thing. Pi is editable there. A CLI whose own editor PiCode has not built yet
says so and links the vendor's documentation. A CLI that keeps no key map file
at all — Hermes Agent, whose three editable keys are rows in its **Settings**
tab — says that and sends you to them. A CLI that does not allow remapping (Grok,
Muse Code) says that and links its own key list, which is the only way to see
those keys.

## When a CLI wants a key the browser keeps

There is no way for a web page to take `Ctrl+T` back. Where a CLI's own
chord collides with one of these, give the CLI a different chord if it
allows one — the Keyboard pane is where you do it, and the marked rows say
which ones — or run it in a terminal outside the browser. Everything
outside the reserved list above already reaches the terminal untouched.

## Fullscreen mode hides PiCode, not your window

**Fullscreen** (`Ctrl+Shift+Enter`, the right-click menu, or the command
palette) hides PiCode's own chrome — the sidebar, the tab strip, the
Inspector — and gives the whole page to the tab you are on. It does not
touch the browser window: your other tabs and your address bar stay where
they are.

If you want the screen as well, press `F11`. That is your browser's own
fullscreen, it works exactly as it always has, and the two stack: enter
either one first, leave either one first, nothing else changes. It does
not hand the reserved keys over either — that needs an API the browser
offers only to a page that asked for fullscreen itself, which PiCode
deliberately does not do.

Inside the mode, `Escape` takes one step at a time:

| Gesture | What it does |
|---|---|
| `Escape` with a panel revealed | Closes that panel, stays in the mode |
| `Escape` with nothing revealed | Leaves the mode |
| `Escape` inside a terminal | Goes to the CLI — vim, pagers and TUI menus keep it |

From a terminal, leave with `Ctrl+Shift+Enter`, the right-click menu, or
the exit button at the right end of the revealed tab strip.

## Rebinding PiCode's own shortcuts

PiCode's own chords stay out of the browser's reserved set so they work
everywhere: `Ctrl+K` for the command palette, `` Ctrl+` `` for a new
terminal, `Alt+[` / `Alt+]` to cycle tabs. Rebind them in **Preferences →
Keyboard**, and the agent's own key map in **Agent CLIs → that CLI →
Keyboard**.

| Keys | Action |
|---|---|
| `Ctrl+K` | Command palette |
| `` Ctrl+` `` | New terminal |
| `Ctrl+.` | Toggle inspector |
| `Ctrl+Shift+F` | Find in terminal |
| `Ctrl+Shift+Enter` | Fullscreen mode |
| `Alt+[` / `Alt+]` | Previous / next tab |
| `Alt+W` | Close tab |

In **PiCode Desktop**, `Ctrl+R` and `F5` reload the page on screen: a
work-browser or web-app tab reloads that site, everywhere else reloads
PiCode. Inside a terminal they stay with the CLI — `Ctrl+R` is still
reverse search.

In a terminal pane: `Ctrl+C` copies a selection or interrupts, `Ctrl+V`
pastes, `Ctrl+Shift+C` / `Ctrl+Shift+V` always copy / paste, and
`Shift+Esc` hands the keyboard back to the app from a Canvas panel.
