# AGENTS.md — one file, nine readers

Study: `docs/benchmarks/2026-09-23-agents-md.md`. The owner's decisions of
2026-09-23 are below; nothing is built yet.

## Decided (owner, 2026-09-23)

- Read-only first: the Instructions matrix and its findings (study options A–D), with no ADR. Repository writes (option E) wait for one ADR, after the read-only view has been used.
- The matrix lives in a workspace tab beside Files and Git, one per repository.
- Trust is explained, never written: PiCode says why a CLI ignores the folder and opens that CLI's own prompt (study §9).
- The Omp nested-worktree bug is filed upstream: [can1357/oh-my-pi#13010](https://github.com/can1357/oh-my-pi/issues/13010).

## Next

- Nothing queued. Built 2026-09-23/24: the Instructions page (`#/instructions/<workspace>`), Settings rows, observed reads, ADR-0204 fixes, Draft with an agent, instruction revisions on exits, and every CLI's rule measured (Muse, Omp, Antigravity with its exact-folder trust). What is left waits on Omp upstream (#13010, under Debts).

## Debts

- [ ] Omp 18.2.11 in a PiCode worktree (`<repo>/.worktrees/<name>`) reads the main checkout's `AGENTS.md` and any `AGENTS.md` between the repository and home as well as the worktree's own (measured); Pi, Claude Code, Grok and Hermes read the worktree's only. Filed upstream as can1357/oh-my-pi#13010.
- [x] This repository's `AGENTS.md` (329 lines, 21,149 bytes) is truncated by Hermes on models with a small window (20,000-character floor; measured at 64k) and uses 88% of Antigravity's 24,000-byte per-file cap. Paid 2026-09-24: trimmed to 214 lines, 13,053 bytes (under both limits).
- [x] Muse Code 1.3.0 and Antigravity CLI 1.2.9 rules come from their docs (Antigravity's from docs embedded in the binary), not from a run; one sentinel run each before the resolver declares them. Paid 2026-09-24 (study, "Measured 2026-09-24"): Muse confirmed; Antigravity 1.2.10 loaded none at a session's start, so its cells are now unknown.
- [x] The phone app has no Instructions view; the tab is desktop only. — paid in part by `feat/instr-mobile`: the New agent sheet shows the line; the table stays desktop-only by design (a wide-screen view).
- [x] Omp's `.omp/`, `.claude/`, `.gemini/` and Copilot providers are declared from reading its bundle (nearest folder, start folder only); only the plain AGENTS.md/CLAUDE.md walk and the per-depth tie were run. Paid 2026-09-24: every provider run; one fix, `.agent/` wins the tie over `.agents/`.
- [x] Antigravity 1.2.10: when (if ever) `GEMINI.md`/`AGENTS.md` reach the model — measuring it needs a run that may use a tool, i.e. an allow-rule in `~/.gemini/config` or an interactive run by the owner. Paid 2026-09-24: at a session's start in an exactly trusted folder, interactive only; PiCode reads the trust list.
