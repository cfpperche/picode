# 2026-09-25 — agent-skill-debts: four slice-4 debts paid

(a) `sweepAgentSkillCache` (agent_skill_cache.go): keeps digests named by a live agent
or a not-forgotten exit (Bring back needs them) and copies younger than an hour; runs
at start, after removeForAgent / updateForAgent, and after ForgetAgentExit.
(b) Check / Update for agent skills (skills.go): `checkAgentSkills` previews each
source once and compares by name + digest; `updateForAgent` caches the newer copy and
swaps the entry; the web's Check sends `agent` (both apps).
(c) `clilaunch.Plan.AgentInjection` from `agentSkillsPlan`: Launch summary "Agent's own
skills" plus argv and folders in the details (both apps).
(d) `store.AgentSkill.Missing`, read-time only (cleared before storing and in exits);
desktop row line (opens the agent chip), phone Work row text; `agentSkillsMissing`.
Verified: Go tests for each (sweep's five cases; check current→behind→updated→current→
unreachable; plan for Claude/Omp/Codex; missing mark); cliSkills.test.js; scratch QA:
row warning, Vega's update to v2 (new digest), Launch summary, phone row.
Hermes/OpenCode agent scope stays a debt: neither CLI takes a folder per launch.
