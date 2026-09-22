# 2026-09-22 — feat/cli-door-adopt

ADR-0184 slice 4, the last one. A PiCode shell running a catalog CLI shows
"<CLI> is running in this shell · **Make agent**" (`adoptOffer`, both apps'
`TermSurface`). `POST /api/terminals/{id}/adopt` binds the shell to a new
agent in its workspace (free → free agent in the pane's live folder), sets the
launch to that CLI with PiCode's tools for its next start, and refuses (409)
nothing detected / already an agent / a sign-in / another CLI's launch. Only on
a click; CLIs outside PiCode are untouched.

Detection is the existing runtime presence (ADR-0062); measured on a scratch:
1.1–2.3 s from typing `claude` to the bar (3 s reconcile tick is the worst
case), 0.3–1.4 s for it to go on Ctrl-C. A CLI that execs into a differently
named binary is invisible to the tmux fallback (pre-existing; the PiCode PATH
wrapper announces the real ones).

visual-review: PASS (scratch clidoor4, desktop + 390px: idle shell no bar,
bar while claude runs, Make agent → toast, agent in /api/workspaces, second
adopt 409, launched agent shows no bar; overlayAudit ok).

With this, every user-facing CLI launch is an agent; the only launch terminal
without one is a server-owned sign-in (topic `one-cli-door.md` holds the one
open debt).
