# Agent exits and Outcomes

Decision: ADR-0194. Study: `docs/benchmarks/2026-09-23-agent-exit-feedback.md` (options A–E). Option A — the exit record, the question at removal, the Outcomes page — landed from `feat/agent-exits`.

## Next

- Option B: computed insights over the catalog, each with a minimum sample and the exit ids it rests on; each reason opens its existing door (CLI Settings, Model roles, Connectors, Snippets, Memory, checklist `always`, delivery review); one-click apply only for configuration PiCode owns; the insight shows in the Create dialog; before/after with n. Needs its own tables (insights, applied-at) — ADR-0194 names the shape, not the schema.
- Options C and D wait on a CLI-neutral unattended runner (Automations `start` is Pi-only) or a picode MCP verb for structured proposals; D also needs a security-model ADR (a reviewer reads every agent's transcript and writes repository files).

## Debts

- [ ] The phone asks the question but has no Outcomes screen; the catalog is desktop only.
- [x] Cost per agent — paid 2026-09-23 (`feat/exit-cost`): priced at removal from the session files; residual: a CLI agent counts only its last session, and Grok/Hermes/OpenCode/Antigravity are not measured.
- [ ] Asking only at removal leaves out good agents that are never removed (accepted in ADR-0194; B should read living agents' observed signals too).
- [ ] The Pi TUI turn count follows the watcher's scraped busy line, so a flicker between tools can count twice; guest CLIs and managed Pi count from explicit edges.
- [ ] No response rate exists yet: `asked` vs answered-in-the-dialog is recorded from the first removal on; read it before tuning when to ask.
