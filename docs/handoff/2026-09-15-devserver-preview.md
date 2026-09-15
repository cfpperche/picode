# 2026-09-15 — feat/devserver-preview: dev servers in PiCode (A+C) — MERGED

Owner approved it after the live review; **merged into `main` 2026-09-15**
(merge `30813c01`), after merging main into the branch twice (69+ commits; one
real conflict the auto-merge produced: both sides had already moved
`showFullUrl` above the meta effect, so the merge kept two identical
declarations — dropped in `449d1f7b`).
Docs: `docs/architecture/devservers.md` + changelog fragment.
What landed: `GET /api/devservers` (loopback listeners from `/proc/net/tcp`,
port→pane attribution through each terminal's/agent's process tree, one HTTP
`<title>` probe per port cached 2 min, probes in parallel — serial measured
17 s on this host), the rail's **Servers** tab with one Open per row, a
loopback terminal link that opens in PiCode instead of the system browser
(honouring `localOpenDest`; table + test in `web/browser/src/lib/openLink.js`),
the work-browser tab rendering such a page in a frame when there is no WebView2
(app CSP `frame-src` grew the loopback forms), per-tab URL persistence, and
the strip's browser tab finally gets a name. The listing rule stays narrow: a
PiCode pane owns the port, or it is a usual dev port that answers (the first
version listed Postgres and PiCode itself).
Verified on the branch and again on merged `main` in a plain browser (scratch
`dev` :8473, workspace "Dev server demo" over `var/demo/devserver-site`):
panel row with title/owner/workspace, **Open** renders the page in a frame
with live reload, the terminal's URL opens in PiCode (`#/web/2`), audit ok.
Screenshots: `devserver-main-{panel,open,termlink}.png`.
Not verified: the physical Ctrl+click (harness cannot hold a modifier — the
menu's "Open localhost" row, same handler, was used) and the real WebView2
path (needs the desktop app). The earlier note's two "belongs to main" items
are both closed: the board renders inside the cap again (ADR-0140 expires old
debts) and the blank-window crash was fixed and merged in `86ad102a`.
visual-review: PASS (5/5)
