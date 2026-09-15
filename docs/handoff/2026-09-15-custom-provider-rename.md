# 2026-09-15 — feat/custom-provider-rename: Custom endpoint → Custom provider

Base ef42785f, head 6e6a93e2: 93991ab7 rename + 3eefa6ef prune + main merge
(custom-endpoint-v2.md conflict resolved taking main's whole-section deletion).

Shipped (owner chose labels+docs): UI strings in both app twins — titles,
buttons, menus, toasts, picker entry+hint, aria-label, docs anchor
#custom-provider — plus providers.md, configure.md, cli-providers.md,
routes.md, and changelog fragment. Kept: routes, API paths, component/file
names, backend copy, comments.

Setup guide button dead in desktop shell diagnosed (no on_new_window handler,
affects all external links); owner deferred the fix, debt parked in
open/desktop-v2.md.

Verified: make web + ci-scoped PASS; QA scratch confirmed title, button,
picker, and guide anchor; overlayAudit ok.
visual-review: PASS (2/2 rename shots read in subagent; card 5/5).
Merge: fast-forward ready.
