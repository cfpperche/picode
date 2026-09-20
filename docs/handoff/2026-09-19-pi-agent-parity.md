# 2026-09-19 — pi-agent-parity: pi agents wear the favicon and Continue in…
Owner showed the mobile/narrow sidebar: the Pi agent had a letter face
and no Continue in…, while the Codex CLI agent had both. ProviderFace now
routes every agent through the CLI favicon chain — pi included (the
explicit-id provider path is untouched) — and the pi branch of
agentRowMenu offers Continue in… from the agent's own session pin via
`agentHandoffTerm(ag)` (the pin rides as a terminal lastSession shim into
the same ADR-0088 flow).
Verified: scratch :8471 — Atlas (pi) menu shows Continue in… with every
target (pi excluded as source), pi favicon on the row, Open chat kept,
overlayAudit ok, screenshot read (pi-agent-parity.png). Node tests 14
pass; ci-scoped PASS. visual-review PASS.

## Next up

- Fatia F: automations and Inspector reach CLI agents through the prompt
  door (ADR-0089/0107).
