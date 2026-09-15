# 2026-09-15 — ask-prompt-spec

Owner said go on the Ask prompt (last piece of Browser permissions v1).

## Done

- Read what exists: the shell installs `attach_permission_handler` already
  (btab.rs) — policy map kind → bool, answers Allow/Deny, reports the outcome
  on `btab://permission`. The missing half is the deferral and the prompt.
- Wrote the whole sketch into `docs/handoff/open/work-browser-tabs.md`: the
  tri-state map, the thread_local pending map with fully qualified COM types,
  the **sync** answer command (sync on purpose: it must touch a thread_local),
  the ACL's three edits, and the verification recipe (a local page calling
  `getUserMedia` in a web tab).
- Named the debt that comes with it: a held request with no answer hangs the
  site, so the slice needs a denying timeout or that risk stated.

## Why not the code today

The slice is ~100 lines of COM across shell + ACL + UI, and **none of it is
verifiable from a web scratch** — the deferral path only runs in the desktop
shell against a page that asks for a device. The session reached this point
with its context spent; writing an unverifiable shell path in that state is
how a hung permission request ships. A fresh session starts from the sketch.

Docs-only: `ci-scoped` PASS, no deploy.
