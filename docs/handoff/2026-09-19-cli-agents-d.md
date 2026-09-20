# 2026-09-19 — cli-agents-d: guest agent rows speak their CLI
Owner hit it live: a Claude Code guest showed subtitle "Pi agent", its menu
had no Launch settings, and the only launch-config screen reachable was
`#/clis/new`. Guest rows now name their CLI (`agentSubtitle`), read status
from the bound terminal (runMode never sees the terminal's tmux session),
and the menu (`agentRowMenu`, shared pure module + node tests) carries
Start/Restart/Stop terminal + Launch settings → `#/clis/terminal/<tid>`;
guests never offer Open chat. Mobile starts/stops guests via the same
launch. Peer contact lists label guest owners by CLI (`principalLabel`).
Verified: scratch :8471 — guest menu stopped/running shapes, Launch
settings opened "Claude Code · Launch settings", start → Open, stop with
confirm → Stopped, Pi menu unchanged, `__picodeOverlayAudit` ok. Node tests
8 pass; ci-scoped PASS. Screenshots read: guest-menu, guest-launch-settings,
guest-started, guest-stopped (var/screenshots/).
Blind spot: mobile flow reasoned, not screenshotted (phone hides Launch
settings by design; lifecycle uses the same endpoints as desktop).

## Next up

- Fatia E: rekey Inbox / `picode mcp` / grants onto the agent id.
