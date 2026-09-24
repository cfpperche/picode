# 2026-09-24 — attach-kind-row

Owner report: the Delivery selector in the desktop attach composer sat in the
header line, far from Send. Moved it into the field row, right before Send,
at `--ctl-h` (AttachComposer.jsx, app.css `.term-attach-row .term-attach-kind`).
The header keeps the hint/attachments and the close; its 28px min-height (a
guard for the chip appearing mid-turn) is gone with the chip.

Verified: visual-review PASS in a subagent on a scratch with an inert Codex
terminal — idle and working rows all 36px, bottom-aligned (0px spread), a
4-line field keeps chip and Send bottom-aligned, popover opens upward inside
the viewport, overlayAudit ok in light and dark. Mobile untouched (native
select row). `make ci-scoped` green.

Trade noted: while working the chip takes 91–151px from the field; at 900px
wide the field keeps ~227px. An icon-only chip on narrow windows is the
option if that proves tight.

Confirmed live by the owner after deploy, 2026-09-24 ("ficou bom").
