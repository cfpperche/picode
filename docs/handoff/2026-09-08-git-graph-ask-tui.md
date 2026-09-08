# 2026-09-08 — feat/git-graph-ask-tui: "Ask Pi" reaches pi as a terminal

The owner runs pi as an Agent CLI terminal for now (the managed chat still
has bugs), and the graph's ask door reached only agents of the store — so on
the owner's instance the form offered Prepare and Run, nothing else. The
owner approved option (c): the door reaches a **pi terminal whose ADR-0060
receiver is alive**, and nothing else. Recorded as an amendment to ADR-0089.

What changed, all reuse: the receiver extension takes `PICODE_TERM_ID` when
it has no agent id and posts to the daemon named by `PICODE_TERM_URL` (the
activity hook's variable — this also keeps a scratch pi from reporting to
production); the pi wrapper injects it beside the activity extension; a
terminal's hello carries its session file, kept in the registry (a terminal,
unlike an agent, has no record of one). `POST /api/terminals/{id}/ask`
delivers through `deliverViaReceiver` under the key `term-<id>` — never a
paste — and records `terminal_ask_delivered/failed` as events, because
`tasks.agent_id` references `agents`. The graph lists pi terminals as
occupants (`kind: "terminal"`, `live` = the runtime's presence), offers
**Open Pi** into the pane and **Ask Pi** in the form. Claude Code, Codex and
the rest: `409 cli`, as before.

Proven live on a scratch with a real pi launched through Agent CLIs: the
payload listed `{"id":"pi-…","name":"Pi","kind":"terminal","live":true}`; the
ask answered `200 {via:"receiver", proof:true}` and the pane's context went
from 0 to 12 tokens — the model's reply then hit the account's usage limit,
which is the account, not the door; the form's select read *Prepare · Run ·
Ask Pi* with the prompt as preview.

visual-review: PASS (gg5-ask-pi-terminal; overlayAudit ok; card 5/5).

Fixture lesson, again: `New()` copies `Deps`, so a test that registers a
`TermRuntime` must construct `TermRuntimes` itself — the same trap the agent
ask fixture documents for `Replies`.

Debts: the recorded cwd is used for a terminal's worktree, not the live pane
cwd (a TUI owns its pane; one tmux call per terminal per load was refused);
no hello arrives until the pi has drawn its first session, so "Ask Pi" is
absent for a few seconds after launch and the request says `no-session`.
