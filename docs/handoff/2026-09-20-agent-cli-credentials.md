# 2026-09-20 — feat/agent-cli-credentials: credential vault study and plan

Shipped: `docs/benchmarks/2026-09-20-agent-cli-credentials.md` — probed where the
nine catalog CLIs (pi, claude-code, codex, opencode, grok, hermes, muse, agy, omp)
keep credentials, plus pi/omp internals, WSL2 keyring reality, BYOK GUI benchmarks
and the "one rotating refresh token, one consumer" rule.
`docs/plans/agent-cli-credentials.md` — step 1 (encrypted vault at
`<DataDir>/credentials.json` + `credentials.key`, one vault for pi and the guests,
providers pane for all nine, import of native logins, migration from
`~/.picode/accounts.json`) and step 2 (binding a credential on
`clilaunch.Config`/`Overrides`, env-var or per-account-dir injection, guided vendor
login, harvest on stop); two ADRs to seed (`credentials-vault`,
`credential-injection`) and six owner questions gating step 1.
`docs/benchmarks/README.md` gained its index row and studies bullet.
Verified: docs-only; `make close` green (FULL= GO= WEB= PACKAGES= DOCS= METADATA=1).
Blind spot: the probed store rows come from this WSL2 machine, and vendor claims
the study could not confirm are marked inference in it.
visual-review: n/a (no UI change).
Not done / debts: no changelog fragment — nothing user-visible shipped. No code;
implementation waits on the six owner questions, and durable items live in
`docs/handoff/open/agent-cli-credentials.md`.
Merge: fast-forward ready.
