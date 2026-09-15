# 2026-09-15 — launch-spike: Muse/agy launch-parity spike (slice 0)

Owner (2026-09-15) approved covering the agy/muse row-menu gaps (missing Launch settings and Continue in…) by bringing their launch to parity with the other CLIs, and approved starting with a spike.
Shipped: slice 0 delivered `docs/plans/launch-muse-agy.md` — the equivalence table (8 capabilities), the adapter anatomy (Catalog row + `cliIntegrationPlan` branch + Reader/Writer/Prompter), and probe results against Muse Code 1.3.0 / agy 1.2.3. All probes read-only.
Muse verdicts: Prompter GO (positional PROMPT, resume `<uuid>`); Reader GO via official `export --session` (schema v1, non-interactive when piped); activity LIKELY via native plugin hooks (PreToolUse documented, full event list unconfirmed); index `sessions.status` is metadata-only (`valid`/`missing_metadata`) so no polling it as Working; Writer UNKNOWN.
Agy verdicts: Prompter GO (`--prompt-interactive`, `--print --output-format json`, `--conversation`/`--continue`, `--model`/`--effort`/`--mode`); Reader LIKELY (per-conversation sqlite `steps` table, numeric `step_type` codes to map); activity has NO hooks (`settings.json` holds only model+trustedWorkspaces) so only a polling fallback from `presence/*.lock` + summaries `status`/`not_fully_idle` (lock staleness risk noted); Writer UNKNOWN.
Slices 1–5 are defined in the plan with entry/exit criteria; writers are explicitly allowed to fail to brief-only.
visual-review: N/A (docs-only; no UI)
