# 2026-09-24 — exit-instr

Exit records now freeze the instruction files the agent read (ADR-0194
amendment 2026-09-24; the owner's item 1 of the AGENTS.md follow-ups).

- `cliinstructions.AgentRevisions`: the CLI's own record (Claude Code,
  Codex, Grok → `observed`), else the rules for its CLI and folder
  (`declared`); path, bytes, 12-hex SHA-256; never the text.
- Stored in `config.instructions` (no migration); every exit path passes the
  `ExitInput.Instructions` hook: agent delete, terminal removal, workspace removal.
- Outcomes ▸ exit ▸ How it was set up shows the line (desktop; the phone
  has no exit detail).

## Debts

- The hash is taken at removal, not at session start: an edit made while the agent ran is attributed to it.
