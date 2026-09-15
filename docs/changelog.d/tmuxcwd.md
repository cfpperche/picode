### Fixed

- A new terminal's folder is right from the first response. The creation answer read the pane's directory live, and that read races the pane's own process: tmux returns the *server's* directory — the daemon's own working folder — until the shell has spawned (measured: 12 of 30 creations in a loop), so the UI and the Inspector could see the wrong folder for a moment. The creation answer now names the folder the session was created in; every later poll still reads the live path, where a `cd` shows up.
