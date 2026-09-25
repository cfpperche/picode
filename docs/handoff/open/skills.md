# Skills for every agent CLI

Decision: ADR-0196. Plan: `docs/plans/skills.md`. Study: `docs/benchmarks/2026-09-23-skills-marketplace.md`. Slice 1 (read-only Skills tab) landed from `feat/skills-inventory`; slice 2 (install, remove, update, check) from `feat/skills-install`; slice 3 (each CLI's own per-skill switch) from `feat/skills-toggles`; slice 4 (an agent's own skills at launch, Pi, Omp and Claude Code) from `feat/skills-agent`; architecture in `docs/architecture/skills.md`.

## Next

- Slice 7 `feat/skills-doctor`, if the debts below still matter: locally modified skills, broken links, diverged copies.

## Debts

- [x] The user menu said "Packages — Skills, extensions, updates" while the pane lists plugins for eight CLIs. Paid by `feat/skills-inventory`: Packages reads "Plugins, extensions, updates", and search offers **Skills**.
- [x] Muse's project folders, Omp's and Grok's user folders were unmeasured. Measured 2026-09-23: Muse reads `.agents` then `.claude` in trusted workspaces, above the user folders; Omp reads `.agent(s)/skills` walk-up and in the home; Grok reads `~/.agents/skills` above `~/.claude/skills` and no project skills in an untrusted folder.
- [ ] Grok hides vendor-shipped Claude skills (`docx`, `pdf`, `pptx`, `xlsx`, `skill-creator` from `~/.claude/skills/synced`) behind its bundled copies; the reader lists them as loaded. `grok inspect --json` is the vendor's own answer and could replace the folder read for Grok.
- [ ] Hermes hides a skill whose `requires_toolsets` or `environments` are not active (measured: `sdlc-review`); the reader lists it as loaded.
- [ ] Bundled and plugin skills (Muse's `bundled:`, Grok's `~/.grok/bundled`, Claude plugin skills) are not rows: the report covers folders a user or installer writes.
- [ ] Trust state is read only for Pi (`trust.json`); Grok, Hermes and Muse rows say "If trusted". Grok's `projectTrusted` (from `grok inspect`) and Hermes' `skills.trusted_project_dirs` are readable next.
- [ ] `internal/slashres` (the Pi composer's `/skill:` picker) still has its own scanner; the plan wanted it on the new reader. It also lists loose `.md` files at a skills root, which the reader does not.
- [x] Unmeasured: whether Claude Code's `--plugin-dir` needs `.claude-plugin/plugin.json`. Measured 2026-09-24 by `feat/skills-agent` on 2.1.281: no; `claude --plugin-dir <dir> plugin details <name>` lists the skill.
- [ ] Codex per-launch: measured 2026-09-24, `-c 'skills.config=[…]'` filters installed skills per run but adds no folder, so Codex still has no agent scope. A Grok per-launch mechanism is unmeasured.
- [ ] Hermes and OpenCode have no agent scope: Hermes `--skills` preloads a skill that is already installed and OpenCode's `permission.skill` filters, so neither adds a folder for one agent. An agent list of names they already have (preload/allow) would be a different feature; not built.
- [x] The launch plan (Launch pane) does not list an agent's own skills; the argv in the launch snapshot does (`--skill`, `--config`, `--plugin-dir`). **Paid 2026-09-25 (`feat/agent-skill-debts`):** the plan's `agentInjection` names them and the flag each CLI takes (Launch summary "Agent's own skills", details list the argv and folders).
- [x] A skill missing from the cache is named on the Skills tab's agent chip, not on the agent card. **Paid 2026-09-25:** `AgentSkill.Missing` is read at every agent read (never stored); the desktop row shows "<name> skill missing" (opens the agent chip), the phone's Work row shows it too.
- [x] The agent skill cache (`<data>/skills/cache`) is never swept; a restored agent relies on that. A sweep would keep every digest named by an agent or an exit. **Paid 2026-09-25:** `sweepAgentSkillCache` keeps every digest a live agent or a not-forgotten exit names, and copies younger than an hour; runs at start, after an agent skill's removal or update, and after an exit is forgotten.
- [x] Update and Check do not cover agent skills: a newer copy means adding the skill again from its source. **Paid 2026-09-25:** Check reads each agent skill's source again (current / behind / missing / unreachable); Update caches the newer copy and swaps the agent's entry (a critical finding needs force).
- [ ] Global lock entries PiCode writes have an empty `skillFolderHash` (the skills CLI's GitHub tree SHA); that CLI will offer an update for them until it rewrites the entry.
- [ ] Update and Check reach only sources PiCode can fetch (github, well-known, local); a Vercel entry of another `sourceType` reads as unreachable with the reason.
- [ ] Remove does not yet name the agents that lose the skill (no agent scope until slice 4).
- [ ] A GitHub source downloads the whole repository tarball (up to 50 MiB); a sparse fetch of the skill's folder would be lighter.
- [ ] Switches (slice 3): OpenCode's `deny` dropping a skill is read from its bundle, not from a command (no OpenCode command prints the filtered list).
- [ ] Switches: Claude Code's `user-invocable-only` and `name-only` values read as on; the pane offers only on/off.
- [ ] Switches: Hermes' per-platform `skills.platform_disabled` lists are not read or written.
- [ ] Switches: Codex and Muse keys use the path the reader found. If either vendor canonicalises a skill that sits behind a symlink (as slice 2's links do), PiCode's entry may not take effect while reading back as off. Unmeasured.
- [ ] Switches: `path.Match` stands in for the vendors' globs. OpenCode reads `[abc]` literally, and Omp may accept `{a,b}` and `**`, so such a pattern may read differently from how the CLI applies it.
- [ ] Switches: Claude Code's managed-settings layer, which outranks the others, is not read.
- [ ] Switches: an `opencode.jsonc` with trailing commas (which OpenCode accepts) is unreadable to `stripJSONC`, so its rows lose the switch.
- [ ] Switches: a file with no final newline gains one on a round trip, and a CRLF file gains a blank line when a user-written Codex element is removed.
- [ ] Switches: under Grok's name-keyed switch, a shadowed copy of a disabled skill still reads "Shadowed" although both copies are off (QA 2026-09-24).
- [ ] `#/clis/antigravity/skills` (the id is `agy`) shows Pi's header above an "unknown cli" error; the header and the pane disagree (QA 2026-09-24).
- [ ] The Marketplace shows no third-party audits: skills.sh's `/api/search` carries none, so a skills.sh card links to its page. A skills.sh audits endpoint, if one is published, would put them on the card.
- [ ] Installing a skills.sh card fetches the whole repository (the search names the repository, not the folder); a card with 20 siblings reads all 20 to preview one.
- [ ] A source is re-read only when someone searches and it is a day old; nothing refreshes the catalog while no pane is open.
- [ ] Not built from slice 6: `claude plugin eval --ablation with-without` runs (they spend real tokens; a job with `--max-cost-usd` when wanted).
- [ ] The Outcomes skill comparison counts a skill loaded in a run, not a skill the agent actually invoked; no CLI's session record was read for invocations.
- [ ] A managed Pi agent's skill set is read at its removal, not at its start (no launch snapshot): a folder changed in between is recorded as it stands at removal.
