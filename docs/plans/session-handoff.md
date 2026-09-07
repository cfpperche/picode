# Cross-CLI session handoff work plan (ADR-0088)

- Date: 2026-09-06
- Owner approved: the handoff direction (not a live switch), the create-only
  exception to the never-write rule, milestone 1 = Claude Code ⇄ Codex ⇄ pi
  native with Grok and Hermes via brief, and a deterministic brief (no model).
- Status: all phases delivered and smoke-tested on real binaries
  (Claude Code 2.1.263, Codex 0.153.4, Grok 1.0.13, OpenCode 1.18.29,
  Hermes Agent 0.21.0, pi).

## Outcome

Any session listed under Agent CLIs → Sessions can continue in another CLI.
The Sessions row menu offers "Continue in <CLI>…" for every target the
server advertises (`GET /api/clis` → `sessions: {list, read, write,
prompt}`); a dialog previews what travels and what is left behind; the
handoff writes a new native session (Claude Code, Codex, Grok, pi as an
adopted agent) or hands it to the target's own importer (OpenCode,
Hermes), and falls back to a brief for any CLI without an import path.
Lineage shows on both rows. Adding a CLI = implementing `Reader` /
`Writer` / `Prompter` in its own `internal/clisession` file.

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

## Phase 3 — Grok native (done)

The spike passed on 2026-09-07: `summary.json` and `chat_history.jsonl`
are the whole minimum. Grok Build 1.0.13 resumed a session written by the
Go writer and answered a question only that conversation could answer.
`updates.jsonl`, `events.jsonl` and `prompt_context.json` are runtime
state the CLI rebuilds, and the folder's shared `prompt_history.jsonl` is
left alone, so Grok's own picker does not list a handed-off session —
`--resume <id>`, which is what PiCode launches, does. Two findings shaped
every writer: Grok refuses a foreign `current_model_id` and switches, and
Go's default HTML escaping turned `<user_query>` into an escape sequence
no vendor writes.

## Phase 4 — OpenCode and Hermes (done)

Both keep sessions in a SQLite store their CLI holds open, and both ship
an importer, so neither needed a store write: `opencode import <file>` and
`hermes sessions import --from claude <path>` (ADR-0094). Proven against
both binaries on 2026-09-07. Hermes assigns its own session id and folds
tool calls into text, so its writer hands over a Claude Code transcript
with tools rendered as turns and verifies arrival rather than block for
block. Hermes still cannot start from a prompt, so it has no brief mode.

## Next

Lineage polish, and the codex scan cache item in `docs/handoff.md`.
