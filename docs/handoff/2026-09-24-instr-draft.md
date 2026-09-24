# 2026-09-24 — instr-draft

Instructions' empty state gains **Draft with an agent** (the owner's item 2
of the AGENTS.md follow-ups: the CLI's `/init` as a draft, outside ADR-0204).

- `POST /api/workspaces/{id}/agents` takes `prompt` (`startPromptLaunch`,
  `internal/server/agents_prompt.go`): terminal CLIs start with their
  configured args + the brief handoff's `PromptArgs`; managed Pi gets a
  queued task; Hermes (no Prompter) is 409 and left out of the dialog.
- QA on scratch with a fake Codex: the terminal received the prompt.
  The managed-Pi path is covered by the Go test only (no live model run).

## Debts

- A relaunch before the CLI's session is pinned sends the draft prompt again (same as a brief handoff).
