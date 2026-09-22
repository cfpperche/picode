---
description: Install PiCode from a GitHub release on Linux or WSL, add the CLIs you use, then create your first agent.
---

# Getting started

Linux or WSL with systemd user services. You do not clone the repo.

::: tip If you only remember one thing
Download the release, run `picode install`, open `https://localhost:8445`.
Clone the repo only when you are changing PiCode.
:::

**Windows:** [PiCode Desktop](/guide/windows-desktop). **A machine you reach from elsewhere:** [On a server](/guide/remote-server).

## 1. Install tmux

PiCode keeps every agent and terminal alive in tmux, so work survives a closed browser tab. Install tmux 3.5+ from your distro's package. Nothing else is required: the coding-agent CLIs come in step 3.

## 2. Install PiCode

```bash
V=$(curl -fsSL https://api.github.com/repos/cfpperche/picode/releases/latest | sed -n 's/.*"tag_name": *"v\{0,1\}\([^"]*\)".*/\1/p')
curl -fLO https://github.com/cfpperche/picode/releases/download/v$V/picode-linux-amd64
curl -fLO https://github.com/cfpperche/picode/releases/download/v$V/SHA256SUMS
sha256sum -c --ignore-missing SHA256SUMS
chmod +x picode-linux-amd64
./picode-linux-amd64 install
```

`install` copies the binary to `~/.local/bin/picode`, writes the systemd user unit, and starts it. Put `~/.local/bin` on your `PATH` if `picode` is not found after this.

## 3. Install the CLIs you use

PiCode runs the real CLIs — Pi, Claude Code, Codex, Grok, Hermes Agent, OpenCode, Muse Code, Antigravity and Omp — and works with whichever of them you install; none is required. Open `https://localhost:8445`, then **Agent CLIs**: the page shows which CLIs are on this machine, offers **Install** for Pi, Codex and Claude Code, and links the vendor's own command for the others. For example:

```bash
npm install -g @earendil-works/pi-coding-agent   # Pi
npm install -g @anthropic-ai/claude-code         # Claude Code
```

Two kinds of thing run in PiCode. An **agent** is a workspace or free instance of one CLI: PiCode owns its launch, follows its activity and collects its questions in the Inbox. A **terminal** is a loose shell. Pi is the one CLI that also has a managed mode with structured chat; every other CLI runs its own TUI in the terminal PiCode gives it. Details: [Agent CLIs](/guide/agent-clis).

## 4. Create an agent

The sidebar has a tab per kind: **Agents** (agents without a project), **Workspaces** (your project folders — each card holds its agents and terminals, and its buttons create them right there), **Terminals** (loose shells) and **Pins**. Add a workspace, add an agent inside it and pick which CLI runs there (or add a free agent), then click **Run**. Close the browser tab; the agent keeps running.

![The desktop shell: workspaces and agents in the sidebar, what needs you first, and today's numbers on the right](../img/app-fleet.png)

<video controls muted preload="metadata" poster="/picode/video/create-agent-poster.jpg" src="/picode/video/create-agent.mp4" style="width:100%;border-radius:12px"></video>

*Video: creating an agent from the browser — the screens above are the real app.*

The first visit may warn about the certificate. `picode provision` installs a locally trusted one when `mkcert` is present (it may ask for root once).

## After that

| Command | What it does |
|---|---|
| `picode update` | Download and verify a newer GitHub release, then restart |
| `picode uninstall` | Remove the service; `--purge` also deletes `~/.picode` |
| `picode pair` | Print a one-time link for another browser or phone |

Phone: [On your phone](/guide/mobile). Send the tab you are looking at to an agent: [Chrome extension](/guide/browser-extension). Changing PiCode itself: [From source](/guide/from-source).

Personal use is free under PolyForm Noncommercial. Company use needs a [commercial license](/license).
