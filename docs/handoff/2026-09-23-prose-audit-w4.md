# 2026-09-23 — prose-audit-w4: wave 4 closes the architecture prose audit
Shipped: wave 4 — the 21 stable architecture files — verified clean.
Path sweep: zero missing across every backtick-cited path; ADR sweep: 74
cited, zero missing; numeric caps verified in code: pins (16 tags, 40
runes, 100 KB body, 24 files, 8 MB image, 16 MB file, 2 MB scene),
automations (30 s watchdog, 2 h timeout), snips (snipDecodeLimit, 100 KB
body), devservers (5 s poll). PI_ROLES_AGENT, PI_COMPACT_AGENT and the
picode-communication MCP server confirmed where the prose names them.
No drift, so no changelog fragment: nothing user-visible changed this
wave. Tracker: docs/plans/docs-prose-audit.md now records all four
waves — ~280 claims verified, 22 drifts fixed (waves 1–3), 0
unverifiable left unresolved.
Verified: `make ci-scoped` PASS (docs gates). Method blind spot: stable
files were swept (paths, ADRs, constants, absolutes) rather than read
paragraph by paragraph; measured-historical values (e.g. climetrics
30-day sums) accepted as records.
visual-review: n/a
Not done / debts: none — the audit plan is complete.
Merge: fast-forward ready.
