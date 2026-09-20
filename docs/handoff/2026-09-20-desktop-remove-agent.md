# 2026-09-20 — desktop-remove-agent: removing a CLI agent takes its terminal
Desktop parity for the mobile fix (4e1477f6): removeAgent now deletes
the CLI agent's bound terminal after the agent DELETE, drops the
terminal's xterm attach and its "t:" tab, scoped to non-Pi agents —
exactly what agentRowMenu's copy promises ("Delete this agent and its
terminal"; Pi's copy says only the agent goes). Verified on the scratch
end-to-end: seeded a codex agent bound to a terminal via the workspace
API, removed it through the sidebar menu, and the API confirmed both
the agent and the terminal are gone (Atlas and the seeded shell
remain); overlay audit ok:true.

## Next up

- Server adoption migration for launch_cli terminals without an agent
  (the three guest-era orphans on the owner's fleet remain).
