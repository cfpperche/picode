# 2026-09-17 — agent-history-access

v2b from the browser scope, shipped as ADR-0146 (the boundary is the security
model: a new class of the human's data becomes agent-reachable).

## Decided

One verb, `history`, read-tier (it drives nothing) with **its own gate**:
`browser.historyAccess`, `never` by default, `allow` when the human says so.
The daemon answers it from the store, so a read works with no browser tab open
and the gate cannot live in the shell's policy map. The answer is url, title
and time, capped — never content, cookies or sessions.

**`ask` is deferred, not faked**: the shell's Ask bar defers a live
`PermissionRequested` on a tab, and a history read has neither a tab nor a
request. A third state that can never fire is the dead control this project
keeps refusing to ship; the two candidate doors (inbox ask/reply, a daemon
prompt surface) are named in the ADR.

## Verified

- Gate table in `browser_tool_history_test.go`: unset → 403 naming the setting,
  allow → rows, query narrows, limit honoured, limit above the cap answered,
  master switch first.
- End to end on a scratch: 403 by default, PUT allow → 200 with the seeded
  visit, back to never. Production was left at `never` on purpose — reading
  the owner's history into a session is not a test I should run.
- `internal/browser`'s closed-catalog guard caught the new verb and was
  updated deliberately; 383 web tests, 12 package tests.
- Visual: the row reads "Agent history access — Never" in Browser permissions.

## Verified live (owner set Allow for it, 2026-09-17)

Against the deployed daemon, reading **counts and field names only** so the
owner's own browsing list never entered the session: `limit 3` → 200 with three
rows (`host, id, title, typed, url, visitedAt`, three distinct hosts); a query
narrowed it to the five QA pages that match; an anonymous caller (no agent, no
term) is answered too — the gate is the setting, not a tier. `summarizeHistory`
was exercised on fabricated rows: `date  title — url` per line plus the count,
and a title equal to its URL is not repeated.

Still the owner's: a CLI agent invoking the verb through the pi-browser tool
itself (the same endpoint, one layer up).
