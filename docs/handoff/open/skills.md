# Skills for every agent CLI

Decision: ADR-0196. Plan: `docs/plans/skills.md`. Study: `docs/benchmarks/2026-09-23-skills-marketplace.md`. Slice 1 (read-only Skills tab) landed from `feat/skills-inventory`; slice 2 (install, remove, update, check) from `feat/skills-install`; architecture in `docs/architecture/skills.md`.

## Next

- Slice 4 `feat/skills-agent`: the This agent scope — migration 070, launch injection per CLI (Pi `--skill`, Omp `--skills=`, Claude `--plugin-dir`, Hermes `--skills=`, OpenCode `permission.skill`), the restart fingerprint for guests.
- Slice 3 `feat/skills-toggles`: each CLI's own switch (Claude `skillOverrides`, Codex `[[skills.config]]`, OpenCode `permission.skill`, Omp `skills.ignoredSkills`, Grok `[skills] disabled`, Hermes `skills.disabled`, `muse skills disable`).

## Debts

- [x] The user menu said "Packages — Skills, extensions, updates" while the pane lists plugins for eight CLIs. Paid by `feat/skills-inventory`: Packages reads "Plugins, extensions, updates", and search offers **Skills**.
- [x] Muse's project folders, Omp's and Grok's user folders were unmeasured. Measured 2026-09-23: Muse reads `.agents` then `.claude` in trusted workspaces, above the user folders; Omp reads `.agent(s)/skills` walk-up and in the home; Grok reads `~/.agents/skills` above `~/.claude/skills` and no project skills in an untrusted folder.
- [ ] Grok hides vendor-shipped Claude skills (`docx`, `pdf`, `pptx`, `xlsx`, `skill-creator` from `~/.claude/skills/synced`) behind its bundled copies; the reader lists them as loaded. `grok inspect --json` is the vendor's own answer and could replace the folder read for Grok.
- [ ] Hermes hides a skill whose `requires_toolsets` or `environments` are not active (measured: `sdlc-review`); the reader lists it as loaded.
- [ ] Bundled and plugin skills (Muse's `bundled:`, Grok's `~/.grok/bundled`, Claude plugin skills) are not rows: the report covers folders a user or installer writes.
- [ ] Trust state is read only for Pi (`trust.json`); Grok, Hermes and Muse rows say "If trusted". Grok's `projectTrusted` (from `grok inspect`) and Hermes' `skills.trusted_project_dirs` are readable next.
- [ ] `internal/slashres` (the Pi composer's `/skill:` picker) still has its own scanner; the plan wanted it on the new reader. It also lists loose `.md` files at a skills root, which the reader does not.
- [ ] Unmeasured, each a gate of the slice that uses it: whether Claude Code's `--plugin-dir` needs `.claude-plugin/plugin.json`; Codex per-launch skills through `-c` or `-p`; a Grok per-launch mechanism.
- [ ] Global lock entries PiCode writes have an empty `skillFolderHash` (the skills CLI's GitHub tree SHA); that CLI will offer an update for them until it rewrites the entry.
- [ ] Update and Check reach only sources PiCode can fetch (github, well-known, local); a Vercel entry of another `sourceType` reads as unreachable with the reason.
- [ ] Remove does not yet name the agents that lose the skill (no agent scope until slice 4).
- [ ] A GitHub source downloads the whole repository tarball (up to 50 MiB); a sparse fetch of the skill's folder would be lighter.
