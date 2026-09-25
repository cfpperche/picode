# 2026-09-24 — skills-outcomes: outcomes with and without a skill, and Promote (ADR-0196 slice 6)

Shipped: `clilaunch.Snapshot.Skills []SkillUse{name,digest,scope,via}`, written by `prepareCLITerminal` via `launchSkills` (the Skills
report's loaded rows; an isolated Pi/Omp agent gets only its own; "If trusted" rows left out). An exit's `config.loaded` =
`{source: "launch"|"exit", skills}`: the terminal's applied snapshot, else `ExitInput.Skills` at removal (managed Pi). Older exits
have none and count on neither side. `store.AgentExitSkillStats` + `GET /api/agent-exits/skills` (the exits filters): per skill,
with vs recorded-without outcome counts and measured cost, `versions` = distinct digests, recorded/unrecorded counts. Outcomes (both
apps): "Skills: resolved with and without", share = resolved / (resolved+partial+unresolved) as in the headline; a side under 5
answered attempts is tagged "few runs"; a coverage line; an exit's detail lists its skills. Skills tab: an agent's own row says
"Trying in <agent> only…"; **Promote to <workspace>** (`POST /api/skills/promote {agent,name}` → `Manager.Promote`) stages the
agent's cached copy (never a fresh download), installs it as a workspace install (links; the lock names the original source so
Check/Update work; same 409s with Replace/accept retries), then drops it from the agent's list. A free agent cannot promote.
Fixed (found by the visual review, pre-existing since slices 1–2): `skills.Digest` did not follow a linked root, so every workspace
skill Claude Code sees through the `.claude/skills` link read as "edited"; now `EvalSymlinks` first (`TestDigestFollowsALinkedRoot`).
Verified: store table test (with / without / unrecorded); server tests (launch snapshot records own + folder skills, isolated only
own; stats route; promote route); JS helper tests. Scratch: 7 exits (4 with `tried`: 3 resolved, 1 partial; 3 without: 1 resolved,
2 unresolved) → 75% vs 33%, "few runs" on both sides; Promote moved the skill into the workspace and off the agent's list.
Blind spot: counts a skill loaded in a run, not one the agent invoked; no real multi-week data behind the comparison.
visual-review: PASS after one FAIL round (mobile header gutter, card margin, "edited" after promote, per-side few runs, path jargon
in the promote question).
Gotcha: the scratch's workspace was the worktree itself, so Promote wrote `.agents/`, `skills-lock.json` and a `.claude/skills`
link into it; removed before commit. Point a scratch workspace outside the tree.
Not done / debts: `claude plugin eval --ablation` not built (spends tokens); not deployed; three new debts and Next (slice 7) in
`docs/handoff/open/skills.md`.
Merge: main moved (14 behind); merge main, rerun `make close`, then fast-forward.
