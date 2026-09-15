# 2026-09-15 — feat/devserver-preview: dev servers in PiCode (A+C), NOT merged

Owner asked for the dev-server door (A+C) in an isolated worktree, to look at
it before deciding: **the branch is left unmerged on purpose.** Docs here: `docs/architecture/devservers.md` + changelog fragment.
What is here: `GET /api/devservers` (loopback listeners from `/proc/net/tcp`,
port→pane attribution through each terminal's/agent's process tree, one HTTP
`<title>` probe per port cached 2 min, probes in parallel — serial measured
17 s on this host), the rail's **Servers** tab with one Open per row, a
loopback terminal link that opens in PiCode instead of the system browser, the
work-browser tab rendering such a page in a frame when there is no WebView2
(app CSP `frame-src` grew the loopback forms), and per-tab URL persistence.
First version listed every listener (Postgres 5432, PiCode 8445); the rule is
now: a PiCode pane owns the port, or it is a usual dev port that answers.
Two bugs found by QA and fixed here: my own effect read a state above its
declaration (blank shell) — and **the same trap is already on main** in
`WebTab.jsx` from commit `4f1a68a7` (address-bar pref): the work-browser tab
takes the whole window down with a ReferenceError. Fixed in this branch only;
main is still broken for anyone who opens a web tab. The live build (`0b2035e`)
predates it, so nothing is broken in what is deployed today.
Verified: `make ci-scoped` PASS; Go tests (parsers, seams, listing rule);
frontend 792+357+338+9. Scratch `devserver` (:8473): a dev server in a PiCode
terminal listed with title/owner, **Open** renders it inside PiCode, live
reload updates it, the terminal's URL opens in PiCode, reload restores the
tab; audit ok. The terminal door respects `localOpenDest` (decision table in
`lib/openLink.js`, 1 test over 5 rows).
Verified on both shells in a plain browser (`/browser/` = the desktop layout a
browser opens, `/desktop/` = the composition the Windows shell loads: same
components, `web/desktop` is `boot()` over `web/browser`): the Servers tab,
the row, **Open** and the frame all work in each; the native WebView2 path is
the shell's own `btab_navigate`, unchanged by this branch. One visual defect
found and fixed while checking: the row clipped the workspace name at a 300px
rail — the owner line now wraps.
Not verified by me: the physical Ctrl+click (the harness cannot hold a
modifier across a synthetic click — the menu's "Open localhost" row, same
handler, was used) and the real WebView2 path (desktop app only).
visual-review: PASS (5/5)
Two things this branch could not fix and that belong to main: `make close`
stops on the handoff board being over budget (12300 B / 12288; main is already
over — `providers-custom` carries 6 bullets), and main moved after this branch
was cut, so it is not a fast-forward today — merge main in before any merge.
