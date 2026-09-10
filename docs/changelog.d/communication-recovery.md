### Fixed

- Recover observed native conversations and activity after a PiCode server restart on Linux/WSL, preserving running terminals, drafts and pending messages (ADR-0112).
- Show activity, connection status and completed communication tests separately on desktop and mobile; keep selected participants visible while reconnecting.
- Renew Pi receiver presence without restarting its terminal, and bind Hermes message commands to the native conversation when a background review changes its environment.
- Deliver pending messages when Grok's empty composer shows an unaccepted native suggestion, preserving typed drafts and native approval guards.
