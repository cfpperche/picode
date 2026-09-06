# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

**Agent contract:** every commit with a user-visible change MUST add an entry
to the `[Unreleased]` section. The repository's official language is English
(see `AGENTS.md`); changelog entries included.

## [Unreleased]

### Added

- **llama.cpp service preparation (internal only)**: execution previews now
  pin executable content with SHA-256, reject invalid file paths/modes and
  emit explicit GPU, Jinja, autoload and bounded CPU-thread settings. Service
  lifecycle controls are still under development.

- **llama.cpp service ownership design** (ADR-0084): the upcoming lifecycle
  controls are scoped to explicitly owned local profiles; external routers
  remain read-only for service and cache management.

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
