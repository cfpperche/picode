# 2026-09-20 — keybar-row-fixes: adversarial pass on the shipped work
Owner-requested self-review of today's mobile work. Four defects found
and fixed:
1. KeyBar momentum inverted (glide(-vel) negated the drag rate twice) —
   a released flick slid the bar back; short drags never glide, which is
   why it passed the first QA.
2. A second finger's touchstart early-returned without preventDefault,
   leaving its default (the IME blur) armed.
3. Remove agent orphaned the bound terminal: the server only nulls
   terminal_id and stopAgentLocked kills the agent-id tmux session, not
   the terminal's — a live CLI survived as an unowned Work card. Mobile
   now deletes the bound terminal after the agent (matching the shared
   menu copy). Desktop removeAgent has the same gap — flagged, not
   changed here.
4. Agent-row status chip lacked the terminal row's 92px ellipsis guard.
Debt noted: the three guest-era orphan terminals on the owner's fleet
(agents/codex/Terminal) still await the adoption migration (Fase 1 of
the Work-tab plan presented 2026-09-20; not yet built).

## Next up

- Server adoption migration for launch_cli terminals without an agent.
- Desktop removeAgent parity: it also leaves the bound terminal orphaned.
