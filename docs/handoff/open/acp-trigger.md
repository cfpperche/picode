# ACP — ADR-0091's trigger, re-measured

Study: `docs/benchmarks/2026-09-22-zed.md` §B. **ADR-0091 stands**; this topic
holds the owner decision it asks for, not a plan.

## Next

- Owner call: ADR-0091 refuses an agent-protocol client "until a single protocol carries first-party, non-adapter support across most of the CLIs PiCode hosts". Re-measured 2026-09-22 — five of nine are first-party native (Grok Build, Hermes, OpenCode, Antigravity, Omp), two more are vendor co-maintained adapters (Claude Code with Anthropic, Codex with OpenAI). Does that meet "most", and does it re-open the decision?
- If it re-opens: Pi is the weakest ACP citizen of the nine (community adapter only; upstream closed PR #836 saying it belongs outside pi-mono, on top of rpc mode). Decide whether Pi stays on RPC while guests speak ACP, or whether that asymmetry argues for waiting.

## Debts

- [ ] Hermes Agent and Omp ship native ACP but are absent from the ACP registry, so a registry-driven integration would miss two CLIs PiCode already manages (`docs/benchmarks/2026-09-22-zed.md` §B).
- [ ] Muse Code has no vendor ACP implementation, only community wrappers — any ACP path leaves it on terminal-only for the foreseeable future.
- [ ] The cost of re-opening is unpriced: a per-agent capability matrix (Zed disclaims history, checkpoints and token usage per agent), vendor-owned auth, and a launch path for each CLI.
