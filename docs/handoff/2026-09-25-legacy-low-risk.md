# 2026-09-25 — feat/legacy-low-risk: the low-risk legacy group retired
Shipped (cd59b1c61, 9367e9e08, c797949f4, 0daeec855, 0dffe09d0, 9e3b692bf), the
group the owner asked for: #/app/inbox (desktop "That app is gone."; mobile
lands on Now), #/app/matrix + x:matrix tab restore, #/preferences/status (4
copies), #/clis/terminals and mobile #/changes/<a|t|w>/<id> retired; orphan
web/browser/src/lib/mobileRoutes.js deleted; QA scripts moved to #/inbox and
#/inspector/w/<id>. readScope keeps only shared scope words: ?layer= and pane
words (user/project/machine/local) are an invalid scope; KINDS stays for writing.
termTheme no longer reads/writes picode-term-theme/picode-term-size keys (the
window event keeps its name); Canvas no longer migrates picode-matrix-last /
picode-matrix-view:<id>. gateway --insecure-listen alias removed. ADR
amendments 2026-09-25 on 0075, 0079, 0101, 0102, 0103, 0118, 0208 (also
covering the earlier branches' retirements).
Verified: `make ci-scoped`, `make close` green. Scratch instance: #/inbox and
#/app/canvas load; #/app/matrix and #/app/inbox show "That app is gone." on
desktop; mobile #/app/inbox → Now.
visual-review: PASS with nits (leglow-app-inbox.png, leglow-mobile-app-inbox.png)
Not done (pre-existing): the shared gone-tab card has a hint but no action
button; the tab strip still labels the previous tab (e.g. "Canvas") under it.
Merge: fast-forward ready.

## Debts

- Remaining legacy items (owner's call): docs/handoff/open/legacy-compat.md
