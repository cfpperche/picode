# Skills for every agent CLI

Decision: ADR-0196. Plan: `docs/plans/skills.md`. Study: `docs/benchmarks/2026-09-23-skills-marketplace.md`.

## Next

- Slice 1 `feat/skills-inventory`: a read-only Skills tab for the nine CLIs (declarations, reader, digest, shadowing, provenance from the locks, trust state), `GET /api/skills/report`, live parity with `muse skills list --json`, `hermes skills list` and `grok inspect --json`; the Packages menu copy corrected.
- Slice 2 `feat/skills-install`: install, remove, update and check for machine and workspace, writing `.agents/skills`, a `.claude/skills` link and the `skills` CLI's locks; gated by hash vectors from the real `npx skills`.

## Debts

- [ ] The user menu says "Packages — Skills, extensions, updates" (`web/browser/src/lib/userMenuModel.js:18`, `web/mobile/src/lib/moreMenuModel.js:19`) while the pane lists plugins for eight CLIs; corrected in slice 1.
- [ ] Unmeasured vendor facts, each a gate of the slice that uses it: Muse's project folder; whether Omp and Grok read `~/.agents/skills`; whether Claude Code's `--plugin-dir` needs `.claude-plugin/plugin.json`; Codex per-launch skills through `-c` or `-p`; a Grok per-launch mechanism.
