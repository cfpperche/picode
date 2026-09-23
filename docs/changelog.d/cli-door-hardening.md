### Fixed

- A shell could still become a CLI terminal with no agent through its launch
  settings; that is refused now — use **Make agent**.
- A sign-in terminal closes when its login exits (it no longer drops back to a
  hidden shell), times out after 15 minutes without activity instead of 15
  minutes after it opened, and **Check now** after coming back to the card no
  longer files the account that was already saved.
- Esc and clicks inside the sign-in window go to the login instead of closing
  it; on phones the window has the terminal key bar.
- **New agent** with a folder that does not exist says so instead of creating
  it; a Pi **New agent** from Agent CLIs is the same Pi agent the palette makes.
- Upgrading no longer fails to start when a terminal belongs to a workspace
  that is gone, a Pi terminal keeps its conversation as an agent, and
  deliveries a terminal registered stay reachable by its agent.
- A stopped agent terminal offers **Start** instead of sending you to Agent CLIs.
