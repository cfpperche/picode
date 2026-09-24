# 2026-09-24 — skills-agent: an agent's own skills at launch (ADR-0196 slice 4)

Shipped: `agents.skills` (migration 073), `Store.SetAgentSkills` announcing `agent.updated`;
content cached at `<data>/skills/cache/<digest>/<name>` by `Manager.Install` with
`scope=agent` (no lock, no link). `POST`/`DELETE /api/skills` take `scope=agent` + `agent`;
the report with `?agent=` lists them with scope `agent`, root "This agent only". Launch,
measured 2026-09-24 against each CLI's own command list: Pi `--skill <folder>` (a folder copy
of the same name wins; `--skill` survives `--no-skills`); Omp `--config
<data>/skills/agents/<id>/omp.yml` with `skills.customDirectories` (agent copy wins; isolation
turns folder sources off in the overlay, since `--no-skills` also drops custom folders:
14 skills → 1); Claude Code `--plugin-dir <run>/picode-agent` (no manifest, measured on
2.1.281; skills are `/picode-agent:<name>`). Hermes and OpenCode scoped out (no way to add a
folder for one agent); ADR-0196 carries a dated note correcting its launch row. Every agent
terminal's fingerprint folds in its own scope (`agentLaunchFingerprint`, before injection), so
"Launch changes pending" follows a skill change, guests too; exit config carries the skills
and a restore puts them back. UI (both apps): agent chip for Pi, Omp and Claude Code with Add
skill ("Add to <agent>"), Remove on agent rows, "Missing" status, one-line note on CLIs
without an agent scope.
Verified: `make close` green; store, install, launch and route tests; `cliSkills.test.js`.
visual-review: PASS, card 5/5 (two FAIL rounds: duplicate missing message, remove confirmation
below the fold, empty pane hiding the agent chip, Codex note missing when empty, "0 of 0").
Blind spot: flags measured by running each CLI directly; no agent was started through PiCode with its own skill.
Not done / debts: slice 3 (toggles) next; debts in `docs/handoff/open/skills.md`.
Merge: fast-forward ready.
