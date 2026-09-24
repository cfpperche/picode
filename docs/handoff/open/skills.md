# Skills for every agent CLI

Decision: ADR-0196. Plan: `docs/plans/skills.md`. Study: `docs/benchmarks/2026-09-23-skills-marketplace.md`. Slice 1 (read-only Skills tab) landed from `feat/skills-inventory`; slice 2 (install, remove, update, check) from `feat/skills-install`; slice 4 (an agent's own skills at launch, Pi, Omp and Claude Code) from `feat/skills-agent`; architecture in `docs/architecture/skills.md`.

## Next

- Slice 3 `feat/skills-toggles`: each CLI's own switch (Claude `skillOverrides`, Codex `[[skills.config]]`, OpenCode `permission.skill`, Omp `skills.ignoredSkills`, Grok `[skills] disabled`, Hermes `skills.disabled`, `muse skills disable`).

## Debts

- [x] The user menu said "Packages — Skills, extensions, updates" while the pane lists plugins for eight CLIs. Paid by `feat/skills-inventory`: Packages reads "Plugins, extensions, updates", and search offers **Skills**.
- [x] Muse's project folders, Omp's and Grok's user folders were unmeasured. Measured 2026-09-23: Muse reads `.agents` then `.claude` in trusted workspaces, above the user folders; Omp reads `.agent(s)/skills` walk-up and in the home; Grok reads `~/.agents/skills` above `~/.claude/skills` and no project skills in an untrusted folder.
- [ ] Grok hides vendor-shipped Claude skills (`docx`, `pdf`, `pptx`, `xlsx`, `skill-creator` from `~/.claude/skills/synced`) behind its bundled copies; the reader lists them as loaded. `grok inspect --json` is the vendor's own answer and could replace the folder read for Grok.
- [ ] Hermes hides a skill whose `requires_toolsets` or `environments` are not active (measured: `sdlc-review`); the reader lists it as loaded.
- [ ] Bundled and plugin skills (Muse's `bundled:`, Grok's `~/.grok/bundled`, Claude plugin skills) are not rows: the report covers folders a user or installer writes.
- [ ] Trust state is read only for Pi (`trust.json`); Grok, Hermes and Muse rows say "If trusted". Grok's `projectTrusted` (from `grok inspect`) and Hermes' `skills.trusted_project_dirs` are readable next.
- [ ] `internal/slashres` (the Pi composer's `/skill:` picker) still has its own scanner; the plan wanted it on the new reader. It also lists loose `.md` files at a skills root, which the reader does not.
- [x] Unmeasured: whether Claude Code's `--plugin-dir` needs `.claude-plugin/plugin.json`. Measured 2026-09-24 by `feat/skills-agent` on 2.1.281: no; `claude --plugin-dir <dir> plugin details <name>` lists the skill.
- [ ] Unmeasured: Codex per-launch skills through `-c` or `-p`; a Grok per-launch mechanism. Until measured, both tabs say they cannot take a skill for one agent.
- [ ] Hermes and OpenCode have no agent scope: Hermes `--skills` preloads a skill that is already installed and OpenCode's `permission.skill` filters, so neither adds a folder for one agent. An agent list of names they already have (preload/allow) would be a different feature; not built.
- [ ] The launch plan (Launch pane) does not list an agent's own skills; the argv in the launch snapshot does (`--skill`, `--config`, `--plugin-dir`).
- [ ] A skill missing from the cache is named on the Skills tab's agent chip, not on the agent card.
- [ ] The agent skill cache (`<data>/skills/cache`) is never swept; a restored agent relies on that. A sweep would keep every digest named by an agent or an exit.
- [ ] Update and Check do not cover agent skills: a newer copy means adding the skill again from its source.
- [ ] Global lock entries PiCode writes have an empty `skillFolderHash` (the skills CLI's GitHub tree SHA); that CLI will offer an update for them until it rewrites the entry.
- [ ] Update and Check reach only sources PiCode can fetch (github, well-known, local); a Vercel entry of another `sourceType` reads as unreachable with the reason.
- [ ] Remove does not yet name the agents that lose the skill (no agent scope until slice 4).
- [ ] A GitHub source downloads the whole repository tarball (up to 50 MiB); a sparse fetch of the skill's folder would be lighter.
