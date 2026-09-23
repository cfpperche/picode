### Changed

- **Fork agent…** into a new worktree no longer waits for you to press Enter when other agents are working in the repository: creating a worktree touches none of their files, so PiCode runs that one command right away. Every other git command still waits while agents work.
