# 2026-09-22 — feat/dashboard-limits: the Limits card reads every plan the Providers roster fetched

Why: t3code's usage page reads plan limits for five vendors; PiCode's Limits card read only Codex rollouts while the Providers roster already fetched Anthropic, xAI, Z.ai and others.

Shipped: `/api/sessions/stats` carries `plans` = every row of the `internal/usage` cache (new `usage.Cached`: a map read, no pi, no vendor call; failures kept; `Entry` gains `accountLabel`). `limitRows` (`web/shared/domain/dashboardStats.js`) merges them with the CLI rollout limits: a fresher `openai-codex` plan replaces Codex's own reading; windows without a percentage (credits) are not bars; a failed plan (`auth_required`/`error`) is one line + a link to the owning CLI's Providers pane (anthropic→claude-code, openai-codex→codex, xai→grok). The coverage footnote hides when a plan is ok; `resetsIn` no longer says "60m"; hover says "read just now". Docs: `docs/architecture/climetrics.md`; fragment `docs/changelog.d/dashboard-limits.md`.

Layout fixes found by visual review: `.dash-grid-2` is `columns: 340px 2` (one column when a card would drop below 340px — ranked labels were cut to fragments at 900px); the `.spend-row` bar keeps ≥ 40px; the provider icon no longer shrinks; the coverage matrix spans the full row (it clipped at 1000–1100px) with a 12px gap.

Verified: go tests, `make test-js`, `make ci-scoped` green. Visual review on a scratch instance with stubbed stats, 7 rounds (final: `r7-1280-gap.png`, `r7-900-gap.png`, `r7-1280-limits.png` in `var/screenshots/`), overlay audit ok, card 5/5. Blind spot: real vendor plans were never seen rendered — the scratch's usage cache is empty, so every plan row came from the stub.
visual-review: PASS (stubbed data)
Not done / debts: in `docs/handoff/open/dashboard.md`.
Merge: fast-forward ready.

## Next up

- Live check after a deploy: production's Limits card should show Codex 5h, xAI weekly, Z.ai 5h/7d and an Anthropic "sign in again" line (the production usage cache held exactly these on 2026-09-22).
