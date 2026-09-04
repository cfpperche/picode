---
description: Build and install PiCode on Linux or WSL, then create your first agent.
---

# Getting started

Building from source requires [Go 1.26+](https://go.dev),
[Node.js 22](https://nodejs.org) (the version used by CI),
[Pi](https://www.npmjs.com/package/@earendil-works/pi-coding-agent), and
tmux 3.5+. The service install uses Linux or WSL with systemd user services.

```bash
git clone https://github.com/cfpperche/picode.git
cd picode
make build
./bin/picode install    # systemd --user; starts when this Linux session starts
```

Open `https://localhost:8445`. The sidebar has a tab per kind: **Agents** (agents without a project), **Workspaces** (your project folders — each card holds its agents and terminals, and its buttons create them right there), **Terminals** (loose shells) and **Pins**. Add a workspace, add an agent inside it (or a free agent), then click **Run**. Close the browser tab; the agent keeps running.

![The desktop shell: workspaces and agents in the sidebar, what needs you first, and today's numbers on the right](../img/app-fleet.png)

<video controls muted preload="metadata" poster="/picode/video/create-agent-poster.jpg" src="/picode/video/create-agent.mp4" style="width:100%;border-radius:12px"></video>

*Video: creating an agent from the browser — the screens above are the real app.*

`make deploy` rebuilds this repo and restarts the service. `picode update` checks GitHub for a newer release. `picode uninstall` removes the service. `--purge` also deletes `~/.picode`.

```bash
make dev          # run from the repo without installing
```

Green padlock: `make cert` (mkcert). See the [README](https://github.com/cfpperche/picode#quick-start) for TLS and bind details.

To send the tab you are looking at to an agent, load the Chrome extension.
Guide: [Chrome extension](/guide/browser-extension).

When a Pi agent asks through the Inbox while running in a terminal, open the
item and choose **Reply here**. PiCode keeps you on that agent's Terminal tab,
shows **Receiving → Processing → Returning**, then restores the same TUI and
session. **Cancel and return** stops the temporary reply; it never opens Chat
or changes the agent's saved run mode. If automatic return fails, **Return to
terminal** restarts that TUI and stays available to retry if the restart fails.

Personal use is free under PolyForm Noncommercial. Company use needs a [commercial license](/license).
