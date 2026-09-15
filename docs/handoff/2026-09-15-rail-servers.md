# 2026-09-15 — feat/rail-servers: Servers reachable without an anchor

Owner asked whether the gap I reported after the deploy needed fixing: yes,
and it was one condition. The rail's tab row rendered `owner || panel ===
"servers"`, so with nothing selected (empty workspace, deleted workspace, a
fresh window) the rail showed only "Open an agent or terminal to inspect its
files." — no row, no way to click the one panel that is about the machine
rather than about the selection. The row now always renders: no owner gives
Files + Servers (Files keeps that message, so nothing else changed), an owner
gives the usual Changes/Files/PR/Servers. Verified in a scratch of this branch
(:8473): at `#/tree/w/does-not-exist` (the "That workspace is gone." state) the
row reads Files | Servers, Servers lists the dev server started in a terminal,
and Files still renders the tree and the message; with the terminal selected
the row is Changes/Files/PR/Servers as before and the tree still works.
Screenshot: `var/screenshots/rail-noanchor-servers.png`.
Docs: `docs/architecture/routes.md` (the rail clause) and
`docs/architecture/devservers.md` (why the tab does not follow the anchor),
plus the changelog fragment.
Not verified: the ≤767px shell and the narrow-window rail (the row is
horizontal and its tabs already shrink there; no anchor-specific layout).
visual-review: PASS (5/5)
