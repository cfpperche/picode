# 2026-09-11 — feat/clis-terminals-section

The Agent CLIs view has one terminal list instead of two: the general
**Terminals** tab left the tab strip, and a CLI's page keeps its
**Terminals** section, so a terminal reads beside the CLI that launches it.

`#/clis/terminals` still resolves: `cliLocation` maps it to the CLIs view
and the view rewrites the hash to `#/clis`, so old links and the sidebar's
"Agent CLI terminal" entry land on a live surface with no blank frame.
Editor exits (save, gone terminal) go to `#/clis/<cli>`; `TermSurface` and
`PeerConnectionDetails` link to `#/clis`.

**Files** — `CliTabs.jsx` + `AgentClis.jsx` (both shells: tab gone,
redirect, `all` list mode pruned, editor targets); `cliLaunch.js` (+ test,
the old address); `TermSurface.jsx`, `PeerConnectionDetails.jsx` (links);
`docs/architecture/routes.md` §`#/clis`; one changelog fragment.

**Gates** — `make ci-scoped` PASS (fmt, vet, hooks, 643 JS tests, build).
Visual on a scratch instance (:8474): tab strip without Terminals, section
row, `•••` menu (audit ok, rows 36px/36px), Grok empty state, "That
terminal is gone.", mobile 390px with the same six tabs — screenshots in
`var/screenshots/`, read in a subagent: visual-card 5/5, PASS.

**Debt / notes** — the machine-wide cross-CLI list is removed on request:
a terminal is found through its CLI; the sidebar's Terminals tab is
untouched. Pre-existing, not introduced here: a long terminal path
ellipsizes on the phone.
