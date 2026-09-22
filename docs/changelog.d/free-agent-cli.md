### Changed
- **New agent picks its CLI.** The sidebar's **New agent** — on the Agents tab, the create action on the Browser and Computer pages, and the mobile Agents section — opens the same CLI picker a workspace uses: Pi, Claude Code, Codex, Grok, Hermes Agent, OpenCode, Muse Code, Antigravity or Omp, with a Name and an optional Folder. A guest CLI agent gets its own terminal on that folder (ADR-0179); `POST /api/agents` accepts `cli`.
- **The picker lists Pi only when it is installed**, like every other CLI.
