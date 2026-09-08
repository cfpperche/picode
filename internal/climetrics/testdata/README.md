# Real-shaped fixtures

Each line here is a real line from a CLI's own store on the machine this was
built on (2026-09-08), with every text field replaced by `REDACTED` and
nothing else changed. They exist because the first round of tests used
hand-built dicts that reproduced what the author *believed* each format was
— an `is_error` inside an assistant message, a Codex prompt with no
`agents_md` twin — and three wrong numbers reached the deployed dashboard
with every test green. A parser test that does not start from the vendor's
actual line is a test of the author's model, not of the format.

- `claude-parent.jsonl` — an assistant turn with `tool_use`, the **user**
  turn that carries its `tool_result` with `is_error: true`, a `refusal`
  stop, and the cumulative `cost-state` snapshot.
- `claude-agent-a1.jsonl` — a subagent transcript (`agent-*.jsonl`) that
  carries the parent's `sessionId`.
- `codex-rollout.jsonl` — `session_meta`, a `response_item` user message
  whose `content_item_kinds` marks it as an AGENTS.md injection, a
  `response_item` user message with kind `user.text` (the person's prompt),
  the `event_msg user_message` twin that only `codex exec` sessions emit — a
  `codex-tui` session never writes one, which is why the event cannot be
  the source — a `function_call`, and a `token_count` with `info`.
