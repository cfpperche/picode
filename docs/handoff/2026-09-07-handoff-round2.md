# 2026-09-07 — feat/handoff-round2: every Agent CLI gives and receives a handoff

Numbering: browser capture (09-05) keeps ADR-0082 and gains the index row
it never had, absent-checklist becomes 0092; session handoff (09-06) keeps
0088, install-a-missing-CLI (09-07) becomes 0093. The llama screenshots
note cited 0082 for the manager; it is 0080.

Handoff (ADR-0094): a writer may publish through the target's own
importer, so `opencode import` and `hermes sessions import --from claude`
receive the conversation and PiCode never writes those SQLite stores;
`WriteRequest.Run` executes the CLI in the session's folder, bounded. Grok
gained a create-only native writer after the spike ADR-0088 left open
passed: `summary.json` and `chat_history.jsonl` are the whole minimum. All
six CLIs read and write; only Hermes has no brief, having no way to start
from a prompt. Imported rows (`source = claude-code`) now list.

Verified live, each answering something only the handed-off conversation
knew: Grok 1.0.13 (PLATYPUS, 17 files), Hermes 0.21.0 (OKAPI, 42),
OpenCode 1.18.29 (named a file from the source). `make ci-scoped` green.

Three smoke findings, now covered: a written session must name a model the
target can serve (Grok switched away from a foreign id, Claude Code could
not restore one, OpenCode called it invalid), so the source's rides in the
handoff note; OpenCode's importer requires `state.metadata` and a model on
the session and every message; stamps must move forward, so the note leads
the conversation in time. Also killed 39 tmux panes leaked by this
feature's own failing test runs; the tests now clean up before asserting.

Not done: lineage polish; codex scan cache.
