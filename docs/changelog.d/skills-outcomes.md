### Added
- **Did the skill help?** Every agent start now records which skills it loaded, and **Outcomes** has a **Skills** section: for each skill, how often runs with it were resolved against runs without it, with sides of fewer than five answered runs marked as *few runs*. An exit's detail lists its skills.
- **Promote to the workspace.** A skill you are trying in one agent says so in the Skills tab; when it works, **Promote to** puts exactly the copy the agent used into the workspace's `.agents/skills` for every agent CLI there, and takes it off the agent's own list.

### Fixed
- **Skills linked for Claude Code no longer read as "edited".** A workspace skill that Claude Code sees through a link was fingerprinted as the link itself, so its row said it had changed since it was installed.
