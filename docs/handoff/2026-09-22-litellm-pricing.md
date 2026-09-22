# 2026-09-22 — feat/litellm-pricing: estimate unpriced turns at LiteLLM list price

Owner approved LiteLLM's price table on 2026-09-22: ADR-0185 (accepted) supersedes the ADR-0097 study's price-table refusal.
Shipped: `internal/pricing` (Parse/Lookup/Cost; bare-name alias only when direct-provider rows agree, nested rows like `azure/eu/` don't vote, family names never priced, 1h cache-write rate). Loader keeps `var/litellm-prices.json` from boot, fetches now + daily off the request path; `PICODE_PRICE_TABLE_URL=off` opts out.
Estimates cover only turns the CLI left unpriced: every Codex turn, and Claude Code sessions with no snapshot in any file. `estimated` sits beside `cost` on current/prior/byCli/byModel. Cost state is "estimated" when all turns are, "partial" when some stay unpriced. Coverage notes give amount and turns, and the table version joins the stats fingerprint.
UI: Spend line "$X estimated at list price", By CLI "estimated" / "$X est." and "~$X", model rows "~$X", "~" in the coverage matrix, ranked amount column 76px. New public page `docs-site/guide/dashboard.md` (estimates + opt-out).
Measured on the owner's machine over 30 days: Codex $1,542 estimated over 10,984 turns (1 unpriced), Claude Code $115 over 21 snapshot-less sessions, Spend $6.9k → $8.6k ($1.66k estimated). Over 400 days: Codex $4,522.
Adversarial review of e60a1d95 found six bugs, all fixed in 4d1cc1c2. Regional-alias conflicts left 9,005 turns unpriced. Codex was labeled estimated while it had unpriced turns. The hasUnknownModelCost floor was lost. 1h cache writes used the 5m rate (-12%). The cache grew per table version. $0-listed models were counted unpriced.
Verified: go tests (pricing, climetrics, server session stats) and test-js green. Live fetch checked on a scratch instance, which wrote `var/litellm-prices.json` at boot. Blind spot: the dashboard was reviewed with stubbed stats only, not the real data.
visual-review: PASS (round 2 on scratch with stubbed stats: r2-1280-bycli, r2-1280-coverage, r2-900-bycli, r2-1280-long; overlayAudit ok; card 5/5).
Gates: `make ci-scoped` had one failure not caused by this branch. TestOmpSigninStartsBrowserOauth failed because the production daemon held the fixed Omp OAuth callback port 127.0.0.1:53692. Earlier, internal/server tests failed when the session's tmux guard (`~/.picode/bin/tmux`) refused kill-session. The same failures happen on main.
Merge: fast-forward ready. Not deployed.

## Next up

- After deploy, check the live dashboard: the Codex row shows "~$" estimated, the Spend line shows the estimate, and the coverage matrix shows "~". **Done 2026-09-22:** deployed as 0.5.0+386f07c; the owner confirmed the dashboard live.

## Debts

- Resumed Claude Code sessions: turns after the last snapshot are neither priced nor estimated (docs/handoff/open/dashboard.md).
- `costSplit` (Efficiency's cache spend) excludes estimates (docs/handoff/open/dashboard.md).
- Long-context premium is not priced (docs/handoff/open/dashboard.md).
- Per-model price overrides have no surface (docs/handoff/open/dashboard.md).
