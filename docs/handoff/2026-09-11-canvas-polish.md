# 2026-09-11 — four owner decisions on the Canvas and What's New (`feat/canvas-polish`)

**Shipped.**

1. **The switcher stops lying.** One canvas draws its *name* (`.cv-switch-one`,
   a `<span>`: no chevron, no tab stop, no hover); two or more draw the
   `<select>` they always did. **New canvas** stays in the `⋯` menu. Both wear
   `.cv-switch` — same height, same 10 px leading padding, same 104 px floor —
   so the row does not move: measured live, cluster `240.81 x 36` and switcher
   `104 x 36` at `(257, 52)`, byte-identical before the second canvas, after it,
   and after deleting it again. The name stays reachable without a tab stop: it
   is the group's content, and the `⋯` button announces "More actions for Ops".
2. **An empty canvas keeps its plane** and floats one line + **Add panel** over
   the middle of it (`.cv-blank`). The page-level blankslate is gone: it cost
   the reader their chosen ground and made the first panel a React Flow
   remount. Pointer-transparent apart from the card, `z-index: 5` under the
   clusters' 6, gone the frame a panel lands (the optimistic one included).
   **The minimap goes with the panels** — an empty map was a box of nothing.
3. **No React Flow badge**: `proOptions={{ hideAttribution: true }}`, its CSS
   removed with it. `@xyflow/react` is MIT, unchanged and still vendored;
   `NOTICE` has never listed dependencies, so nothing was added there. Recorded
   in `docs/architecture/canvas.md`, "The library's mark", which reverses
   2026-09-10's reading that this was a licence question.
4. **What's New opens on a fresh install** (ADR-0063, dated amendment). The
   product-state gate is out of `shouldAutoOpen` and both shells; every other
   defer condition and the `picode-whats-new-seen` acknowledgement are
   untouched. Runbook step 5 loses the seed-a-terminal recipe and now says an
   empty instance showing nothing is a regression.

**Verified** on a scratch built with `-X …version.Stamped=release` (the source
build reports `release:false`, so a real stamped binary was the only honest
way) against a wiped `PICODE_DATA`: `/api/version` `release:true`, zero
workspaces / agents / terminals, cold load → the surface opened; **Got it** →
`picode-whats-new-seen = "0.2.0"`; reload → stayed closed. Deferral, same
instance: an Inbox `question` item raised the badge and the surface waited
across a reload (`seen` still null); with the palette open the badge was
cleared and it still waited; closing the palette opened it. Unit-level, the
same inputs through main's module and this branch's: fresh `false → true`,
seeded `true → true`, blocked/seen/source-build all unchanged — one row moved.

**visual-review: PASS** — `canvas-label-{dark,light}`, `canvas-two-select-
{dark,light}`, `canvas-empty-{dark,light}`, `canvas-empty-grid-dark`,
`canvas-empty-cross-light`, `canvas-panel-zoom1-dark`, `canvas-zoomout-dark`
(20 %, no badge), `whatsnew-fresh-{dark,light}`; `overlayAudit ok:true` on the
`⋯` menu with the cluster row `[36,36,36]` at `[52,52,52]`. **uiux-review:
PASS** — empty state is one line + one action, no disabled-with-tooltip, no
homemade primitive (a label is not a control).

**Debt.** The `<select>`'s own popup is the OS's, so the open two-canvas list
is DOM evidence (`["Ops","Review"]`), not pixels. A canvas name longer than the
240 px box still ellipsizes in both shapes, and the two shapes differ by the
chevron's 16 px there — only the short-name case is pixel-identical. No mobile
check: the Canvas is desktop-only; the mobile shell got the What's New change
and its unit rows only.

**Merge**: `git merge --ff-only feat/canvas-polish`.
