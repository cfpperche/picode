# 2026-09-14 — onboarding-opencode-zai: OpenCode on the Z.AI coding plan

Shipped: OpenCode's provider resolved per the owner's screenshot — `model=zai-coding-plan/glm-5.3-flash` in `opencode.jsonc` (the Cloudflare default returned Forbidden). Verified on a scratch: model line shows the Z.AI plan, first prompt turns, the onboarding flow connects, and the Run-test probe leaves with the correct check id.
Verified: pane captures and `var/qa/oc-zai/summary.json`; scratch stopped, terminals removed, orphan killed by exact pid.
Findings: at 80 cols the deep worktree path wraps and OpenCode's editor guard reads it as a draft (widen the pane); both scratch tmux sessions were killed externally mid-check (second occurrence today) — recorded in the plan, not attributable from logs.
visual-review: n/a (acceptance run)
Not done / debts: check correlation for OpenCode→grok pending a run without external session kills.
Merge: fast-forward ready after this close.
