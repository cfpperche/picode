# 2026-09-16 — developer-mode: raw CDP for a full-tier agent (ADR-0144)

Owner asked for the reference's Developer mode block ("Elevated risk —
Enable full CDP access", screenshot 19:20) and agreed with the shape I
proposed: unlock raw CDP *for the agent* through the bridge we already
trust, and keep the loopback port out of the UI (it is reachable by any
local process — a hole no tier and no audit can narrow).

Landed, ADR first (0144, accepted):
- one new verb, `cdp`, whose method comes from the caller; granted only when
  `browser.developerMode` is on **and** the caller's tier is `full`
  (daemon gate, fail closed: an unreadable setting means off);
- the shell re-checks both from its own copy of the switch
  (`btab_set_developer_mode`, pushed by the settings page on load *and* on
  save — the same push for the JavaScript switch, which had the load gap);
- every raw call appends one `browser.cdp` event (allowed, refused or
  failed) with principal + method; the card lists the newest 20 and
  refetches on the feed;
- the card: warning label, off by default, "what keeps working" line, and
  the audit underneath.

Verified: `go test ./internal/browser ./internal/server ./internal/store`,
`node --test web/browser` (365 + the new prefs row), `cargo xwin build`, and
the decision table's four rows as table-driven server tests
(off+full → refuse; on+read/act → refuse; on+full → dispatch with
`raw:true` + one audit row). Not run live: the agent-side flow — that is the
owner's click (Settings ▸ Browser ▸ Developer mode, then any `cdp` call).

## Next up

- Owner: turn Developer mode on, ask an agent at Full tier for a raw method
  (e.g. `Network.getAllCookies`), and watch the Raw calls list below the
  switch.

## Debts

- Nothing prunes `browser.cdp`, so a long-lived Developer mode grows the
  events table (the events table's retention concern, not this feature's).
  Terminal callers are audited by `termId` alone.
