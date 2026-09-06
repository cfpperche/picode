# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

**Agent contract:** every commit with a user-visible change MUST add an entry
to the `[Unreleased]` section. The repository's official language is English
(see `AGENTS.md`); changelog entries included.

## [Unreleased]

### Added

- **Inspector: ask a running agent to do a Git action** (ADR-0078). For every
  agent running in the rail's repository — managed or in its own terminal —
  the Git menu now lists an "Ask &lt;name&gt;" submenu with the same actions.
  Asking sends the agent a plain-language message through the channel that
  already carries its prompts: a queued turn for a managed agent (delivered
  at once, or as a follow-up once its current turn ends) or the Inbox
  reply's own door into a TUI (the receiver extension, or a bracketed
  paste). The agent decides how and runs it in its own turn; PiCode never
  runs git here. The commit form's message becomes optional when asking —
  left empty, the agent writes one from the changes. A stopped agent, or a
  terminal hosting a coding CLI, is not offered.

- **Inspector: run Git actions when nobody is working** (ADR-0078). The Git
  menu gains a per-viewer checkbox, "Run when no agent is working here". With
  it on, PiCode types the command into your terminal and presses Enter itself,
  but only when no agent in that repository is mid-turn, no automation is
  running there, and no other terminal there is working or holding a program;
  otherwise the command is prepared as before and a note says who is busy.
  Git still runs in your own shell with your credentials and hooks. Typing
  now also refuses a terminal that moved away from the folder or whose pane
  is not at a shell prompt, and takes a fresh terminal instead. Right-hand
  toasts step left of the rail while it is open, so a note never covers its
  buttons.

- **Session forensics** (ADR-0085): the daemon now records which tmux
  sessions were alive at a graceful shutdown and reports at boot exactly
  which ones did not survive (log + `var/restart-report-*.json`), stopped
  CLI terminals say "PiCode restarted while this terminal was running"
  when that applies, launch scripts ignore SIGHUP like interactive shells
  always did (explicit Stop escalates to SIGTERM so stopping still
  stops), and every `picode deploy` appends who/what/where to
  `var/deploy-log.jsonl`. The next session-loss incident self-reports.
- **CLI terminal session recovery** (ADR-0084): every CLI terminal now pins
  the native conversation it is running (claude, codex, grok, pi) and a
  stopped terminal offers "Resume last session" — one click relaunches the
  CLI with its verified resume arguments. A deploy, crash or daemon restart
  no longer costs the conversation; the pin is stored (`terminal.last_session`
  event) so it survives the process. Plain start is unchanged; nothing
  auto-restarts. Desktop and mobile terminal surfaces both carry the button.
- **Contextual llama.cpp model guidance**: GGUF choices show file size,
  estimated runtime memory and a hardware-oriented starting recommendation.

- **Checklist disclosure on sidebar cards** (`docs/plans/sidebar-checklist-
  expand.md`). Clicking a card's plan line expands the full list in place:
  ☑ on finished steps, the braille spinner on the one being executed, ☐ on
  the rest. The items are already in client state and arrive over the feed,
  so opening costs no fetch and updates render live while open. Works on
  agent and terminal cards; the line is a real button (`aria-expanded`,
  Enter opens, Escape closes) and never navigates. No server change.

- **llama.cpp operation activity** (ADR-0083): load, unload and download now
  run as persistent jobs. Activity shows per-file progress, reconnect outcomes
  and supported download cancellation on desktop/mobile. Duplicate requests
  are deduplicated; conflicting model operations are refused. Unknown results
  remain visible rather than being reported as success.

### Changed

- **Sidebar text column corrected for the smaller identity mark.** The
  runtime-favicon resize shrank agent/terminal identity marks from 24px to
  16px without updating the folder/branch line's (and, transitively, the
  checklist line's) left inset, so every row's `.ws-context` sub-line sat
  8px right of its own title in production. Found while re-verifying the
  checklist chevron fix below against the deployed favicon change; the
  inset is now 23px (16px mark + 7px row gap) everywhere it is used.

- **Checklist chevron moved to its own gutter** (owner refinement, follow-up
  to the compact-view alignment below). The disclosure's chevron sat in the
  same 31px column as the title/folder text, but a rendered icon's ink
  rarely touches its own bounding box the way plain text does — the owner
  spotted the chevron reading as offset from the title above it even though
  the boxes lined up. The chevron now sits in a dedicated gutter *before*
  that column (the tree-view convention: a leading disclosure mark, then
  the label), so the **text** of the checklist line and its expanded steps
  lands on the exact same column as the title, subtitle and folder/branch
  line, independent of the chevron icon's own inset. No JSX change.

- **Checklist compact-view alignment** (owner refinement, `docs/plans/
  sidebar-checklist-expand.md`). The counter drops its parentheses (`5/8`,
  not `(5/8)`) and moves to a fixed column at the row's end, the Linear/
  GitHub sub-issue idiom, instead of prefixing the step text. The line and
  its expanded list now sit in the same 31px text column as the card's
  folder/branch line below it — previously flush left, out of the card's
  grid. The disclosure line's focus ring matches the row's own selection
  style (`box-shadow`) instead of a text-field-like inset outline. A
  finished plan (every step completed) dims to the same weight as a
  completed step, so it stops reading as open activity. The plan line
  moves above the folder/branch line on agent and terminal cards, matching
  identity → activity → location. Mobile's sub-line drops its parens too
  (`5/8 · text`). No server or domain change.

- **Identity favicons match the workspace favicon**. Agent faces (sidebar
  rows, collapsed strips, editor tabs, inspector) and terminal CLI badges
  now render at the workspace favicon's 16px box with its 3px radius —
  natural image shape, no forced circular crop. Letter/glyph fallbacks keep
  the white plate for sidebar contrast; image faces stay full-bleed. Row
  indent (`ws-meta`) re-aligned to the smaller identity mark.

- **Checklist line refinements on cards** (ADR-0082). The `(x/n)` counter
  no longer wears the accent color — the operator line is one muted line.
  An **absent** checklist (the task required a plan and none was written)
  now renders as silence: no line, no "No checklist", on agent cards,
  terminal cards, the terminal pane strip and mobile rows. The data plane
  is unchanged — the absent marker is still published and stored.

## [0.1.0] - 2026-08-23

### Added

- **Project bootstrap**: public repository, MIT license, CI (GitHub Actions
  with gofmt/vet/test/build across linux/macos/windows), Makefile.
- **Living documentation system**: `docs/` with architecture, philosophy,
  engineering + UI/UX benchmarks, handoff (`docs/handoff.md`) and ADRs
  (`docs/decisions/`) — with an explicit contract that documentation evolves
  with the code (see `AGENTS.md`).
- **Pi agent harness**: root `AGENTS.md` (operating contract), quality skills
  in `.pi/skills/` (`quality-gate`, `uiux-review`, `handoff-update`) and
  project settings in `.pi/`.
- **Go server skeleton**: `picode` binary with UI embedded via `go:embed`,
  `/api/health` and `/api/version` endpoints, dark-first placeholder page
  with a live health check.
- **Initial decision records (ADRs)**: browser app served by a single Go
  binary (0001), dual-channel tmux+RPC agent control (0002), dependence on
  user-installed `pi` (0003).

[Unreleased]: https://github.com/cfpperche/picode/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/cfpperche/picode/releases/tag/v0.1.0
