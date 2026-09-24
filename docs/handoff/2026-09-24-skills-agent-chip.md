# 2026-09-24 — skills-agent-chip: the agent chip for terminal-backed agents

The owner found no agent chip in Claude Code's Skills tab with a Claude agent selected
(after the slice 4 deploy). Cause: the desktop passed the Agent CLIs page only
`agent?.id`, which is set for Pi agents; a Claude Code or Omp agent opens as its
terminal's tab, whose owner the App already resolves as `ctxAgent`. The Skills pane
now falls back to `ctxAgent` (`skillsAgentId`); Settings, Packages and Connectors
keep their context unchanged (their agent layers were not re-checked for guests).
The phone's Skills shortcut in More now carries the last agent.
Verified: scratch instance, Claude agent's terminal tab selected, `#/clis/claude-code/skills`
shows Global / QA / Vega; `make ci-scoped` green.
Unchecked: Packages and Connectors read the same `agent?.id`, so an Omp agent selected
from the sidebar likely misses its agent chip there too; not verified or changed here.
