# 2026-09-25 — omp-sessions-workspace: Omp Sessions view lists workspace Omp agents' conversations

Owner approved option 1: when the Omp Sessions view is scoped to a workspace, it also lists that workspace's Omp
agents' transcripts (the same union pi has under ADR-0040). Pi does not share the "Continue in…" gap: its per-agent
dirs nest under `~/.pi/agent/sessions`, so listing and reader already see them (checked live on production with agent
docs-de9ea9: handoff preview OK, session present in the machine-wide list of 272).
Shipped (8f08e8734): `clisession.OmpWorkspace(cwd, agentDirs...)` = omp cwd bucket + top-level `*.jsonl` of each agent
dir (nested artifacts folder skipped); agent-dir rows resume with `--resume <exact path>` (omp looks ids up only in
`~/.omp`). Server `?workspace=` for omp uses `workspaceOmpAgentDirs` (live Omp agents of that workspace only; removed
agents are left to the agent history). `ompSessionUseBy`: a row an Omp agent's terminal is pinned to carries `inUseBy`;
SessionsView (desktop and mobile) shows "in use · <agent>" and "Open agent" instead of "Resume as agent" (a second Omp
on the same file would interleave writers). Both views request Omp with `?workspace=`.
Cost: the owner's only live Omp agent (13.6 MB transcript) lists in ~90 ms uncached; 196 of 197 files in
omp-sessions belong to removed agents and are not read.
Verified: `TestCLISessionsOmpWorkspaceUnionsAgentDirs` (decision table: workspace agent row by path, bucket row by id,
other workspace's agent excluded, `?cwd` bucket only, nested folder skipped, inUseBy after pin); 30× loop green. One
`make ci-scoped` run failed in internal/server without the test name captured; two reruns and the loop were green.
Scratch `ompws`; blind spot: the pin was set by writing the scratch DB directly (no route sets it).
visual-review: PASS (omp-ws-inuse, omp-ws-menu, omp-ws-empty, mobile top/row/menu; overlay audit ok on both menus; 5/5)
Not done / debts: `open/sessions.md` Omp listing line paid ([x]). Not deployed.
Merge: fast-forward ready.

## Debts

- [x] internal/server: one unexplained `make ci-scoped` failure on this branch (test name not captured). Likely cause, found 2026-09-25: /tmp is a 16 GB tmpfs at 93%; 1 of 6 sharded reruns failed in 6 unrelated tests with `database or disk is full` on SQLite migrations. /tmp cleanup is owned by another session.
