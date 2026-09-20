# Agent TUI terminal unification

The Agent CLIs terminal is the implementation base, not a visual template.
Adaptation: paseo's daemon/PTY model and Cursor's scoped controls from
[the benchmark study](../benchmarks/2026-08-24-adopt-t3code-paseo-cursor.md).
Independent app chrome (ADR-0072) stays; transport/input behavior is headless
shared code. Managed Pi and vendor runtime protocols are not replaced.

## Decision and acceptance table

| Conditions | Action | Automated evidence |
|---|---|---|
| Pi/Claude/Codex/other CLI with bound terminal | Use canonical runtime, shared renderer and terminal actions | agentTerminalMatrix; qa-agent-tui |
| Managed Pi with a terminal binding | Chat; no terminal writer attached | agentTerminalMatrix |
| Stopped bound agent | Same stopped/recovery terminal view; no send/keyboard controls | agentTerminalMatrix; qa-agent-tui |
| Explicit Pi Chat view while interactive | Respect view; icon switch returns to same xterm | agentTerminalMatrix; qa-agent-tui |
| Non-Pi requests Chat | Keep TUI; no unsupported managed composer | agentTerminalMatrix |
| Bound record missing or mismatched | No guessed legacy session; use embedded matching record if available | agentTerminalMatrix |
| Live legacy Pi, including failed restart that allocated binding | Same UI/engine, legacy agent address until explicit restart | agentTerminalMatrix |
| Agent CLI terminal link has an owning agent | Resolve owning view, reuse xterm | qa-agent-tui |
| One-finger drag in normal/alternate screen | Shared touch-to-wheel; xterm or TUI owns scrolling | termTouch; termWheel; qa-agent-tui |
| Pinch, canceled gesture, disposed pane | No stale wheel events/listeners | termTouch |
| Socket drops / route switches | Reconnect/reparent same entry, preserve local scrollback | termSocket; qa-agent-tui |
| Session address changes under same ID | Dispose prior attach before connecting replacement | agent-terminal-unification source guard test; not exercised end-to-end |
| Stop/migration of a previously legacy agent | Dispose both possible runtime keys | agentTerminalMatrix; agentConfig tests |
| Attach request fails | Keep message and attachments; show error | qa-agent-tui |
| Terminal open fails | Visible error and Retry, no alternate renderer | qa-agent-tui |
| Canonical pane lives in an agent tab/Canvas | Runtime for input; actual owner/tab for lifecycle/navigation | browser pane/menu tests and scratch QA |

The scratch fixture uses real tmux and WebSocket paths with a harmless
terminal executable, not authenticated vendor completions. Synthetic touch
events verify delivery into that process; they are not physical iPhone
Safari/PWA/IME acceptance. Native Windows shell behavior is not established
by running its browser bundle in Chromium.
