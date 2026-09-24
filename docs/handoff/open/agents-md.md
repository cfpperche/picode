# AGENTS.md — one file, nine readers

Study: `docs/benchmarks/2026-09-23-agents-md.md`. The owner's decisions of
2026-09-23 are below; nothing is built yet.

## Decided (owner, 2026-09-23)

- Read-only first: the Instructions matrix and its findings (study options A–D), with no ADR. Repository writes (option E) wait for one ADR, after the read-only view has been used.
- The matrix lives in a workspace tab beside Files and Git, one per repository.
- Trust is explained, never written: PiCode says why a CLI ignores the folder and opens that CLI's own prompt (study §9).
- The Omp nested-worktree bug is filed upstream: [can1357/oh-my-pi#13010](https://github.com/can1357/oh-my-pi/issues/13010).

## Next

- Settings rows, observed reads and ADR-0204's first fixes (bridge, personal file) landed 2026-09-23.
- Slice 1 landed (`feat/agents-md-instructions`): the resolver, the workspace Instructions tab and the New agent line, read-only. Next: the Settings rows (Claude Code's Project instructions mode, Codex's `project_doc_max_bytes` and fallback names, Hermes's `context_file_max_chars`) and the "what this agent read" line from session records (Claude transcript, Codex rollout, Grok `prompt_context.json`).
- Done 2026-09-23: this repository's `AGENTS.md` trimmed to 214 lines (detail in `docs/agents/`), and the New agent line on the phone. Open: exit records carrying instruction revisions (§10.5), and running a CLI's own `/init` as a draft (left out of ADR-0204).

## Debts

- [ ] Omp 18.2.11 in a PiCode worktree (`<repo>/.worktrees/<name>`) reads the main checkout's `AGENTS.md` and any `AGENTS.md` between the repository and home as well as the worktree's own (measured); Pi, Claude Code, Grok and Hermes read the worktree's only. Filed upstream as can1357/oh-my-pi#13010.
- [x] This repository's `AGENTS.md` (329 lines, 21,149 bytes) is truncated by Hermes on models with a small window (20,000-character floor; measured at 64k) and uses 88% of Antigravity's 24,000-byte per-file cap. Paid 2026-09-24: trimmed to 214 lines, 13,053 bytes (under both limits).
- [ ] Muse Code 1.3.0 and Antigravity CLI 1.2.9 rules come from their docs (Antigravity's from docs embedded in the binary), not from a run; one sentinel run each before the resolver declares them.
- [x] The phone app has no Instructions view; the tab is desktop only. — paid in part by `feat/instr-mobile`: the New agent sheet shows the line; the table stays desktop-only by design (a wide-screen view).
- [ ] Omp's `.omp/`, `.claude/`, `.gemini/` and Copilot providers are declared from reading its bundle (nearest folder, start folder only); only the plain AGENTS.md/CLAUDE.md walk and the per-depth tie were run.
