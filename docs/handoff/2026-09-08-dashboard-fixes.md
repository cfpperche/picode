# 2026-09-08 — feat/dashboard-fixes: the adversarial pass on the cross-CLI dashboard

Shipped: five wrong numbers on the deployed dashboard, all confirmed against
the real stores before fixing. Claude Code errors 0 -> 409/30d (the failure
rides on the user turn); 45 aborts -> 0 (a normal `stop_sequence` was read as
one) and 4 refusals now counted as their own beat; Codex prompts 620 -> 455
(its AGENTS.md injections, named in `content_item_kinds`, were counted as
prompts — and the `event_msg` twin cannot be the source either, since
`codex-tui` never emits it and counting only it found 136); Claude Code sessions 42 -> 30/week (subagent
transcripts fold into the parent by `sessionId`); Hermes errors claimed and
never populated.

The structural fix: coverage is derived from evidence (`guestAcc.evidence`),
not declared per adapter. Three of the five were a hand-written "reported"
beside a counter nothing incremented — the silent zero the ADR forbids, one
layer up. Tests now parse real redacted lines from each CLI
(`internal/climetrics/testdata/`); every hand-built fixture had passed.

Also: `hasUnknownModelCost` marks Claude Code's total partial; `hermesTime`
no longer reads an ISO string as epoch seconds; the server cache key carries
the claimed workspaces so a new workspace refreshes a scoped window; the
cache memory estimate corrected to 60–80 MB. Five design risks recorded as
Still open in ADR-0097 (singleflight, symlinked scope, unpriced ranking,
Codex duplicate events, pre-metadata Codex items).

Verified: `make ci-scoped`, real-store probe (per-CLI turns), JS 478 tests.
Merge: `git merge --ff-only feat/dashboard-fixes && make ci` from main.
