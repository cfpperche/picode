# AGENTS.md — one file, nine readers

Study: `docs/benchmarks/2026-09-23-agents-md.md`. Nothing is built yet; this
topic holds the owner's calls the study asks for (§10) and what it measured.

## Next

- Owner call: build the read-only Instructions matrix and findings first (options A–D, no ADR), and where does it live — a workspace tab beside Files and Git (the study's recommendation), an Inspector section, or a tenth CLI pane?
- Owner call: repository writes (option E: `@AGENTS.md` bridge, personal file + `.gitignore`, the vendor's `/init` through the prompt door) need one ADR — now, or after the read-only view has been used?
- Owner call: file the Omp nested-worktree issue upstream (option G), citing Pi's `findShadowedContextFile`.

## Debts

- [ ] Omp 18.2.11 in a PiCode worktree (`<repo>/.worktrees/<name>`) reads the main checkout's `AGENTS.md` and any `AGENTS.md` between the repository and home as well as the worktree's own (measured); Pi, Claude Code, Grok and Hermes read the worktree's only.
- [ ] This repository's `AGENTS.md` (329 lines, 21,149 bytes) is truncated by Hermes on models with a small window (20,000-character floor; measured at 64k) and uses 88% of Antigravity's 24,000-byte per-file cap.
- [ ] Muse Code 1.3.0 and Antigravity CLI 1.2.9 rules come from their docs (Antigravity's from docs embedded in the binary), not from a run; one sentinel run each before the resolver declares them.
