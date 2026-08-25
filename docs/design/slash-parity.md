# Slash parity — TUI `/` vs PiCode composer

> Status: 2026-08-25. Source of the 24: pi `BUILTIN_SLASH_COMMANDS`
> (23) plus the built-in `/llama` extension. PiCode extras are listed
> separately. **Composer `/x` must open PiCode UI**, never type `/x`
> into the terminal. The TUI keeps every command inside the dock
> (philosophy: door, not cage).

## Rule

| Surface | What `/settings` means |
|---|---|
| Pi TUI (dock) | pi's own settings UI (thinking, theme, delivery, transport) |
| PiCode composer | a **PiCode** action — hash route, chip, dialog, page |

Forwarding the keystroke to tmux is a **debt**, not a design. Skills
Skills and prompt templates (`/skill:…`, `/templatename`) appear in the
composer picker (insert, then Send — pi RPC expands them). We do not
reimplement the skill engine.

## Status key

| Status | Meaning |
|---|---|
| **ui** | Composer `/x` opens a PiCode surface |
| **partial** | Listed in composer; action is incomplete or still TUI |
| **proxy** | Composer sends the slash into the terminal |
| **missing** | Not in `web/src/lib/slash.js` |
| **n/a** | Deliberately not mirrored (or PiCode-only) |

Count against the TUI 24 (not extras): **22 ui · 0 partial · 0 proxy · 2 missing**.

## Matrix (TUI 24)

| # | TUI | TUI does | PiCode should | Status | Notes |
|---|---|---|---|---|---|
| 1 | `/settings` | 29 knobs; writes **global** `~/.pi/agent/settings.json` | `#/settings` pi GUI (ADR-0012) | **ui** | S0 shell. Knobs in S1. Product chrome → `#/preferences`. |
| 2 | `/model` | model picker | focus model chip | **ui** | Composer cockpit |
| 3 | `/tree` | session tree / **in-place** leaf jump | session tree dialog | **ui\*** | \*Parity break: see below. |
| 4 | `/thinking` | thinking level | focus thinking chip | **ui** | Composer cockpit |
| 5 | `/scoped-models` | Ctrl+P cycle set | `#/settings` Scoped models | **ui** | Patterns in `enabledModels`. |
| 6 | `/export` | HTML / JSONL export | download JSONL | **ui** | HTML later |
| 7 | `/import` | resume from JSONL | file picker + resume | **ui** | |
| 8 | `/share` | private GitHub gist | share dialog | **missing** | Not the phone QR |
| 9 | `/copy` | last assistant → clipboard | clipboard of last reply | **ui** | Same text as bubble copy |
| 10 | `/name` | session display name | rename dialog | **ui** | |
| 11 | `/session` | file, id, tokens, cost | session facts pop | **ui** | Dialog from status bar + session file |
| 12 | `/changelog` | pi version history | pi changelog viewer | **ui** | Installed pi package |
| 13 | `/hotkeys` | pi shortcuts | PiCode keymap overlay | **ui** | PiCode keys, not TUI |
| 14 | `/fork` | new session from a user turn | tree dialog, user rows | **ui** | RPC `fork` when in chat |
| 15 | `/clone` | duplicate current branch | clone in tree dialog | **ui** | RPC `clone` when in chat |
| 16 | `/trust` | save project trust | trust UI / confirm | **ui** | Writes `trust.json` for the agent cwd |
| 17 | `/login` | provider auth | `#/providers` API key | **ui** | OAuth/subscriptions still TUI until pi RPC login |
| 18 | `/logout` | drop credentials | `#/providers` sign-out | **ui** | Deletes that key from `auth.json` |
| 19 | `/new` | new session | new session (API) | **ui** | Session bar |
| 20 | `/compact` | compact context | compact confirm | **ui** | |
| 21 | `/resume` | pick a session | session picker | **ui** | |
| 22 | `/reload` | reload skills/config | reload + toast | **ui** | Restarts the process; RPC has no reload |
| 23 | `/quit` | quit pi | stop this agent | **ui** | Must not close the browser |
| 24 | `/llama` | llama.cpp models | llama manager or `#/providers` | **missing** | Extension command |

## `/tree` TUI vs GUI (parity break)

TUI `/tree` calls `session.navigateTree`: **same JSONL**, leaf moves, you
continue on that branch. RPC has `get_tree` / `fork` / `clone` but **no**
`navigate_tree`.

| | TUI | PiCode today |
|---|---|---|
| View | tree | tree (cards) |
| Click a prompt | in-place jump | **`fork`** (new session file) |
| `/clone` | new file | new file (RPC) |

**Workaround (kept):** viewer + fork. We will not write a private leaf
into pi's JSONL.

**Upstream:** asked for `navigate_tree` in
[earendil-works/pi#8645](https://github.com/earendil-works/pi/issues/8645)
(read side was #5810; #5119 asked for both and closed completed).

Public page: [commands#tree](https://cfpperche.github.io/picode/commands#tree).

## `/settings` scope (pi 0.84.x)

TUI `/settings` is **not** session and **not** workspace.

| Layer | Path | Who writes |
|---|---|---|
| Global | `~/.pi/agent/settings.json` | TUI `/settings` (all 29 rows) |
| Project | `<cwd>/.pi/settings.json` | `pi config` (Tab) / `pi install -l` — **not** `/settings` |
| Session | JSONL + in-memory | `/model` `/thinking` `/compact` this run |
| PiCode | `#/preferences` | theme + server port (product, not pi) |
| PiCode | `#/settings` | pi GUI for the selected agent (ADR-0012) |

Project settings *override* global when the cwd is trusted. `/settings`
still always `save()`s `globalSettings`. `defaultProjectTrust` is
global-only by spec. Auto-compact in that menu goes through
`setCompactionEnabled` → global `compaction.enabled`.

Many of the 29 are TUI chrome (padding, hardware cursor, OSC progress).
Those stay in the dock. PiCode `/settings` stays the **product** page.
Pi knobs that matter in the GUI (auto-compact, steering, follow-up)
are a later surface, not a dump of the 29.

## PiCode extras (not in the 24)

| Composer | Does | Status |
|---|---|---|
| `/provider` | focus provider chip | **ui** | TUI folds this into `/model` |
| `/skill:name` | insert, pi expands on send | **ui** | Picker from disk; not in the 24 |
| `/templatename` | insert, pi expands on send | **ui** | Same |

## Where it lives

| Thing | Path |
|---|---|
| This matrix | `docs/design/slash-parity.md` |
| Composer list | `web/src/lib/slash.js` |
| Dispatch | `web/src/desktop/App.jsx` `onSlash` |
| Canonical TUI list | pi `dist/core/slash-commands.js` `BUILTIN_SLASH_COMMANDS` |
| Hash routes | `docs/architecture.md` Application routes |

Bump the count in this file when a row changes status. Do not add a
composer entry that only proxies unless the matrix row says **proxy**
and the next step is named.
