# 2026-09-25 — cli-cost-meters: exits and start runs priced on Grok, OpenCode and Hermes

Follow-up to cli-automation-more, which found OpenCode, Grok and Hermes runs had no meter and so the cost cap could not stop them.
Shipped: `climetrics.MeterSession(cli, path, id, prices)`. Grok reads the session folder (the path is the folder, or `prompt_history.jsonl` plus the id). OpenCode reads `message` rows by session_id from its SQLite store. Hermes reads the session totals row plus the assistant-turn count: a list-price estimate when no cost is recorded, otherwise Unpriced. `sumSession` is factored out of `MeterSessionFile`. `climetrics.Metered` now includes grok, hermes and opencode; JS `METERED_CLIS` matches (`TestMeteredCLIsMatchTheEditor`).
`store.ExitMeter` is now `func(cli, path, id string)`; exits and the start-run cost cap (`cliRunCost`) use `MeterSession`.
Live check (2026-09-25, temporary probe on the owner's real sessions): all three measured ok. Grok showed real costs (e.g. $0.57 on 1.3M tokens). OpenCode and Hermes GLM turns (glm-5.3-flash on the Z.AI plan) are recorded at $0 and the price table has no GLM, so they come out `unpriced`: cost 0 plus an Unpriced count. Such turns never trip a cost cap; the editor hint now says so.
Debts: the new GLM/unpriced debt is in `docs/handoff/open/managed-principals.md`; the old "no measured cost" debt there is paid.
Not done: no live automation start run on the scratch instance after this change, so a cap trip was not seen live. The copy change in the editor hint was not captured visually.
Behavior change: runs made before this change on these CLIs stored cost 0, so the runs table now shows $0.00 for them instead of "—". Run cost is still a plain float, so a run with only unpriced turns shows $0.00 (same as Pi today).
`make close` reported "docs/decisions/README.md: new ADR is not in the index". This is a false positive: ADR-0217 was only edited and is already indexed (line 223).
Not deployed (owner's call). Merge: through `make land` after `make close`; if `main` moved, merge it and rerun (ADR-0124).
