# 2026-09-23 — feat/agents-md-study: AGENTS.md across the nine CLIs

One commit, docs only. The owner asked whether every PiCode CLI supports AGENTS.md and what PiCode could add to manage it. `docs/benchmarks/2026-09-23-agents-md.md` answers: yes, all nine read it (Claude Code since 2.1.277) — but no two read it the same way.

**How it was measured.** Loaders were read in source where each CLI ships it; `grok inspect --json` and Hermes's own manifest ran on sentinel fixtures; cheap model probes (Claude Code on Haiku twice, Omp three times, the last to check the upstream reproduction) settled what only a run could, and the probe transcripts and fixtures were deleted afterwards. Muse Code and Antigravity are docs-only, not run.

**What the measurements found.** Precedence inverts: Claude reads `CLAUDE.md` only when both files exist, most others read `AGENTS.md`, and Grok reads both. A `CLAUDE.local.md` switches `AGENTS.md` off for Claude alone. Grok in an untrusted folder loads zero project instructions (measured: 0 files vs. 6 trusted). Omp inside a PiCode worktree reads main's `AGENTS.md`, the branch's copy, and any `AGENTS.md` above the repository — Pi, Claude Code, Grok and Hermes read only the worktree's own. This repository's own `AGENTS.md` (329 lines, 21,149 bytes) is truncated by Hermes at a 64k-token window. Of the 5 local repositories with both files, 3 point Claude at `AGENTS.md` in prose.

**What went to the owner.** The study offers options, not decisions: a read-only Instructions matrix and findings first (no ADR), settings-pane rows, repository writes behind one ADR, exit-record revisions to measure the effect of an `AGENTS.md` change, and an Omp upstream issue on the worktree shadowing. Nothing was built; the owner's calls live in `docs/handoff/open/agents-md.md`.

Verified: `make close` green (scoped: metadata); `main` merged in afterwards and re-closed. visual-review: n/a — no UI. Nothing deployed.

**Landing failed once.** `make ci` on `main` at 65ba8571 failed `docs-check`: its link regex read Antigravity's include syntax, quoted in inline code in the study, as a broken relative link. `ci-scoped` never ran `docs-check` because it classifies `docs/` as metadata. `feat/agents-md-link` rewords the sentence; the scoping gap is a debt in `docs/handoff/open/process.md`.

**Corrected after landing.** The study first said Omp reads both files. Omp's context-file capability keeps one file per folder depth, won by source priority (`.omp/` 100 … `AGENTS.md` = `CLAUDE.md` 10), and a probe showed the tie going to `AGENTS.md`. `feat/agents-md-omp-fix` corrects the study, and the open topic records the owner's decisions of the same day, including upstream issue can1357/oh-my-pi#13010.
