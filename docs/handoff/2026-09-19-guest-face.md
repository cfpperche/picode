# 2026-09-19 — guest-face: guest agent face wears the CLI favicon
Owner asked why sidebar guest rows showed a letter mark while terminal
rows show the CLI favicon. `ProviderFace` (desktop + mobile) now routes
guests to `GuestCliFace`: `terminalCliFaviconUrls` with the same
on-error chain as `TermFace`, vendor-mark fallback; Pi keeps the
provider face. Verified on scratch :8471: Claude Code guest wears the
Claude favicon, Grok agent the vendor mark, Atlas (Pi) unchanged;
`__picodeOverlayAudit` ok; screenshot read (guest-face-sidebar.png).
ci-scoped PASS. visual-review PASS.
Blind spot: CLI without any favicon asset untested live — falls to the
vendor mark by construction (same chain as TermFace).

## Next up

- Fatia E: rekey Inbox / `picode mcp` / grants onto the agent id.
