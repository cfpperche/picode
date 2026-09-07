# Cross-CLI session handoff work plan (ADR-0088)

- Date: 2026-09-06
- Owner approved: the handoff direction (not a live switch), the create-only
  exception to the never-write rule, milestone 1 = Claude Code ⇄ Codex ⇄ pi
  native with Grok and Hermes via brief, and a deterministic brief (no model).
- Status: phases 0–2 implemented in `feat/session-handoff` and smoke-tested
  on real binaries (Claude Code 2.1.263, Codex 0.153.4, Grok 1.0.13, pi).
  Phases 3–4 planned.

## Outcome

Any session listed under Agent CLIs → Sessions can continue in another CLI.
The Sessions row menu offers "Continue in <CLI>…" for every target the
server advertises (`GET /api/clis` → `sessions: {list, read, write,
prompt}`); a dialog previews what travels and what is left behind; the
handoff writes a new native session (Claude Code, Codex, pi as an adopted
agent) or starts the target from a brief (Grok, and any CLI with a
Prompter). Lineage shows on both rows. Adding a CLI = implementing
`Reader` / `Writer` / `Prompter` in its own `internal/clisession` file.

## Phase 0 — capabilities plumbing (done)

`clisession.CapabilitiesOf`, `cliView.Sessions`, the web picker derives
from the advertised list (`SESSION_CLIS` removed).

## Phase 1 — timeline, readers, brief (done)

`internal/transcript` (Window, Repair, Prepare, HandoffNote, Brief);
readers for Claude Code, Codex, Grok (per-session directory; the listing
now carries title, model, size), pi (leaf ancestry) and Hermes (SQLite);
`session_handoffs` + `session.handoff`; preview/handoff endpoints; menu,
dialog, lineage badges.

## Phase 2 — native writers (done)

Claude Code, Codex and pi writers with round-trip verification and the
`tools: text` option; the installed target version comes from the setup
check, run on demand. Smoke on real binaries: `codex resume <id>`,
`claude --resume <uuid>`, a pi agent chat — record per-version results in
`docs/handoff.md`.

## Phase 3 — Grok native spike, then lineage polish

Write a minimal `<cwd>/<uuid>/{summary.json, chat_history.jsonl}` from a
three-turn timeline into a scratch HOME and run `grok --resume <id>`. If
Grok also needs `updates.jsonl`, keep Grok brief-only and record it.
Note: Grok Build has its own `/resume-claude` import; comparing it with
PiCode's brief is part of the spike.

## Phase 4 — Hermes

Hermes is a source today. As a target it has no interactive launch with an
initial prompt (brief needs `Tmux.PasteText` after the wrapper's
runtime-start) and no import (native means inserting into a SQLite file
Hermes may hold open) — both are owner decisions.
