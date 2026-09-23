# AGENTS.md — one file, nine readers

Study: `docs/benchmarks/2026-09-23-agents-md.md`. The owner's decisions of
2026-09-23 are below; nothing is built yet.

## Decided (owner, 2026-09-23)

- Read-only first: the Instructions matrix and its findings (study options A–D), with no ADR. Repository writes (option E) wait for one ADR, after the read-only view has been used.
- The matrix lives in a workspace tab beside Files and Git, one per repository.
- Trust is explained, never written: PiCode says why a CLI ignores the folder and opens that CLI's own prompt (study §9).
- The Omp nested-worktree bug is filed upstream: [can1357/oh-my-pi#13010](https://github.com/can1357/oh-my-pi/issues/13010).

## Next

- Slice 1 landed (`feat/agents-md-instructions`): the resolver, the workspace Instructions tab and the New agent line, read-only. Next: the Settings rows (Claude Code's Project instructions mode, Codex's `project_doc_max_bytes` and fallback names, Hermes's `context_file_max_chars`) and the "what this agent read" line from session records (Claude transcript, Codex rollout, Grok `prompt_context.json`).
- Still the owner's: trim, split or keep this repository's `AGENTS.md` (study §10.6). Repository writes (option E) wait for one ADR; exit records carrying instruction revisions (§10.5) come after.

## Debts

- [ ] Omp 18.2.11 in a PiCode worktree (`<repo>/.worktrees/<name>`) reads the main checkout's `AGENTS.md` and any `AGENTS.md` between the repository and home as well as the worktree's own (measured); Pi, Claude Code, Grok and Hermes read the worktree's only. Filed upstream as can1357/oh-my-pi#13010.
- [ ] This repository's `AGENTS.md` (329 lines, 21,149 bytes) is truncated by Hermes on models with a small window (20,000-character floor; measured at 64k) and uses 88% of Antigravity's 24,000-byte per-file cap.
- [ ] Muse Code 1.3.0 and Antigravity CLI 1.2.9 rules come from their docs (Antigravity's from docs embedded in the binary), not from a run; one sentinel run each before the resolver declares them.
- [ ] The phone app has no Instructions view; the tab is desktop only.
- [ ] Omp's `.omp/`, `.claude/`, `.gemini/` and Copilot providers are declared from reading its bundle (nearest folder, start folder only); only the plain AGENTS.md/CLAUDE.md walk and the per-depth tie were run.
