# 2026-09-25 — feat/legacy-sessions-drain: pre-ADR-0162 Pi sessions and the ADR-0139 drain retired
Owner approved after measurement: one machine; default tmux server not
running; production 20 agents, none Pi, none legacyInteractive.
Shipped (6c9839649): internal/tmux/drain.go and WithLegacy/SocketFor removed;
public Manager methods are the primary implementations again; the bridge
attaches with SocketPath(); -S/plain-shape tests moved to
internal/tmux/socket_test.go; requireDrainTmux → requireTmuxBinary.
Shipped (42ca108f2): legacyAgentInteractive and the LegacyInteractive JSON
field, agentSession's old-live picode-<id> branch (unbound agents still resolve
to picode-<id>, a test TUI fixture nothing in production creates), the launch
block on an old pane, the /ws/term?session=picode-<id> rewrite, the web
fallback in agentTerminal.js and legacyInteractive reads in agentStatus.js /
feedReducers.js; TestPiLegacyAndRPC deleted. ADR-0139 and ADR-0162 amended.
docs-site SSH guide names `-S ~/.picode/tmux.sock` (plain `tmux ls` was stale).
Verified: `make ci-scoped` PASS (second run), `make close` green. Scratch:
Pi agent open binds picode-sh-<term>; restart agent, restart/stop/start shell,
all on the instance socket; browser attach typed text reached the pane; API no
longer returns legacyInteractive. Blind spot: no pre-ADR-0162 session existed
to observe, so the removal is proven only by absence in production data.
visual-review: PASS (lsd-shell.png, lsd-agent.png)
Merge: fast-forward ready.

## Debts

- TestOpencodeGUICredentials flake and remaining legacy items: docs/handoff/open/legacy-compat.md
