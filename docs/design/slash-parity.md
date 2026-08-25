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
and prompt templates (`/skill:…`, `/templatename`) stay pi's; we do
not reimplement those.

## Status key

| Status | Meaning |
|---|---|
| **ui** | Composer `/x` opens a PiCode surface |
| **partial** | Listed in composer; action is incomplete or still TUI |
| **proxy** | Composer sends the slash into the terminal |
| **missing** | Not in `web/src/lib/slash.js` |
| **n/a** | Deliberately not mirrored (or PiCode-only) |

Count against the TUI 24 (not extras): **7 ui · 1 partial · 4 proxy · 12 missing**.

## Matrix (TUI 24)

| # | TUI | TUI does | PiCode should | Status | Notes |
|---|---|---|---|---|---|
| 1 | `/settings` | 29 knobs; writes **global** `~/.pi/agent/settings.json` | `#/settings` pi GUI (ADR-0012) | **missing** | Plan: [pi-settings.md](pi-settings.md). Product chrome → `#/preferences`. |
| 2 | `/model` | model picker | focus model chip | **ui** | Composer cockpit |
| 3 | `/tree` | session tree / branch jump | session tree UI | **proxy** | UI not built |
| 4 | `/thinking` | thinking level | focus thinking chip | **ui** | Composer cockpit |
| 5 | `/scoped-models` | Ctrl+P cycle set | scoped-models UI | **missing** | |
| 6 | `/export` | HTML / JSONL export | download / save dialog | **missing** | |
| 7 | `/import` | resume from JSONL | file picker + resume | **missing** | |
| 8 | `/share` | private GitHub gist | share dialog | **missing** | Not the phone QR |
| 9 | `/copy` | last assistant → clipboard | clipboard of last reply | **missing** | Copy on bubbles exists; slash does not |
| 10 | `/name` | session display name | rename dialog | **ui** | |
| 11 | `/session` | file, id, tokens, cost | session facts pop | **proxy** | Status bar has some of this |
| 12 | `/changelog` | pi version history | pi changelog viewer | **missing** | Not PiCode CHANGELOG |
| 13 | `/hotkeys` | pi shortcuts | PiCode keymap overlay | **missing** | |
| 14 | `/fork` | new session from a user turn | fork from a bubble | **missing** | Timeline work |
| 15 | `/clone` | duplicate current branch | clone current session | **missing** | Timeline work |
| 16 | `/trust` | save project trust | trust UI / confirm | **missing** | |
| 17 | `/login` | provider auth | `#/providers` + TUI OAuth | **partial** | Still needs the dock for OAuth |
| 18 | `/logout` | drop credentials | `#/providers` sign-out | **proxy** | |
| 19 | `/new` | new session | new session (API) | **ui** | Session bar |
| 20 | `/compact` | compact context | compact confirm | **ui** | |
| 21 | `/resume` | pick a session | session picker | **ui** | |
| 22 | `/reload` | reload skills/config | reload + toast | **proxy** | |
| 23 | `/quit` | quit pi | stop this agent | **missing** | Must not close the browser |
| 24 | `/llama` | llama.cpp models | llama manager or `#/providers` | **missing** | Extension command |

## `/settings` scope (pi 0.84.x)

TUI `/settings` is **not** session and **not** workspace.

| Layer | Path | Who writes |
|---|---|---|
| Global | `~/.pi/agent/settings.json` | TUI `/settings` (all 29 rows) |
| Project | `<cwd>/.pi/settings.json` | `pi config` (Tab) / `pi install -l` — **not** `/settings` |
| Session | JSONL + in-memory | `/model` `/thinking` `/compact` this run |
| PiCode | `#/settings` | our theme + server port (product, not pi) |

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
