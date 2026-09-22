---
layout: home

hero:
  name: PiCode
  text: Agent Development Environment for coding-agent CLIs
  tagline: Control your coding agents from the browser — Pi, Claude Code, Codex, Grok, Hermes, OpenCode, Muse Code, Antigravity or Omp. Create them, watch them work, steer them when they drift. You stay in command from the moment of creation.
  actions:
    - theme: brand
      text: Get started
      link: /guide/getting-started
    - theme: alt
      text: Command reference
      link: /commands
    - theme: alt
      text: Run it on a server
      link: /guide/remote-server

features:
  - title: Real CLIs, not a re-implementation
    details: Every agent is the real CLI process, and its own TUI stays one tab away — a door, not a cage. None of the CLIs is required; PiCode itself needs only tmux.
  - title: A terminal that lives in the browser
    details: Each agent gets a web terminal backed by tmux, so work survives reloads, laptops and browser crashes.
  - title: Sessions stay with the CLI
    details: History stays in each CLI's own session files. Nothing is copied into a private database — your CLI's tools keep working.
  - title: Move a conversation between agents
    details: Start in Claude Code and continue in Codex, Grok, OpenCode, Hermes or Pi. PiCode translates the session into the other CLI's own format and opens it there.
  - title: Nine CLIs, one fleet
    details: Several agents per workspace, several workspaces per machine, whichever CLI each one runs — all in one sidebar with live state.
  - title: Automations on a schedule
    details: Message any agent on a timer or a webhook while its terminal is open, or start a fresh Pi agent each run — with templates, cost caps and a full activity log.
  - title: Built for the phone too
    details: Approve, reply and supervise from your pocket — push notifications included, no app store needed.
---
