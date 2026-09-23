# 2026-09-22 — feat/cli-door-hardening

Fixes from the adversarial review of ADR-0184 (three reviewers: server/store,
web, completeness). The launch editor was a second door: `PUT
/api/terminals/{id}/launch` now answers 409 for a shell or a sign-in, and the
UI route shows a one-line blocked state with Make agent when a CLI is running.
Sign-ins: the launch script ends with the CLI (no hidden shell; measured 54 s
to the "closed" strip at a 1-min tick), the limit is 15 min without tmux
activity, a 30 s grace and a locked verdict stop the reaper racing creation,
the start stamp is kept so Check now after coming back cannot refile the old
account, the dialog keeps Esc/outside clicks, phones get the key bar, and the
dialog fits short windows. Migration 067 rewritten in place (it ran on the
owner's instance with nothing to migrate): JOIN on workspaces (an orphan
terminal no longer aborts boot), Pi session → `session_path`, deliveries and
queue rows follow to the agent, GLOB for the sign-in name (also in 068). Also:
createCLIAgent cleanup, `DELETE /api/managed-clis` stops the CLI, sign-ins off
two more server lists, OAuth timeout yields to an in-flight exchange, missing
folders refused for launch agents, audits name the bound agent, fixture seeds a
real sign-in, plain Pi New agent = the palette's Pi agent, stopped terminals
offer Start. ADR-0188 records the owner's call: a terminal identity keeps the
browser session drive.

visual-review: PASS (scratch clidoor5, desktop 1440/1280x633 + 390px: dialog
keys and sizes, sign-in ends with its CLI, blocked launch editor + Make agent,
Pi New agent lands on chat, bad folder refused, stopped pane Start).
