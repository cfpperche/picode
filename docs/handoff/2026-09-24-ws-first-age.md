# 2026-09-24 — feat/ws-first-age: park-time ranking inside the Working-first view

Owner ask: with "Working first" on, agents that are not working order by how
long they have been parked in their status, shortest park first; with the
view off nothing changes (stored order, as before).

Shipped:
- `bucketAgentsByState(agents, statusOf, stampOf?)`
  (web/shared/domain/agentStatus.js): every bucket whose rows are not
  working sorts by the row's truthful status stamp, newest transition first;
  rows without a stamp keep the stored order after the stamped ones; the
  working bucket keeps the stored order.
- Sidebar passes `statusStampOf` — `agentStatusStamp(statusOf(ag), ag,
  agentTerm(ag, terminals))` — the same stamp the row's age pill renders, so
  bucket order and pill age cannot disagree.
- ADR-0173 amendment paragraph (2026-09-24); changelog fragment
  docs/changelog.d/ws-first-age.md.

Verified: `make ci-scoped` PASS (fmt, vet, hooks, test-js, build,
living-docs); new domain test "bucketAgentsByState sorts the not-working
buckets by park time" (working keeps stored order, unstamped last, equal
stamps stable). Live on qa-scratch `wsfirstage` (:8472): Alpha/Bravo/Charlie
settled ~1-2 min apart plus Delta never started; view OFF kept the stored
order Alpha→Bravo→Charlie; view ON rendered READY Charlie(now)→Bravo→Alpha
and STOPPED Delta last with no age; a second Charlie turn re-settled live to
"Ready · now" and stayed first. Screenshots
var/screenshots/wsfirstage-{1-baseline-off,2-menu-overlay,3-working-first-on}.png,
overlayAudit ok:true. visual-review: PASS (card 5/5), read by a subagent.

Notes: `compacting` rows rank as not-working (transient state, harmless);
the dashboard's fleet ranking is untouched.
