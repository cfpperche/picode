### Fixed

- A git command from the graph or from **Fork agent…**'s new worktree no longer picks a stopped or idle agent's terminal as its shell (the server refused it with "This is an Agent CLI"); it goes to a plain shell in the folder, or a new one.
