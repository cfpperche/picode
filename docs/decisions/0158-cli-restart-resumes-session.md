# ADR-0158: Restart an Agent CLI terminal resumes its pinned conversation

- **Status**: accepted
- **Date**: 2026-09-19
- **Boundary**: process — what a confirmed Restart does to the conversation a live Agent CLI terminal was running
- **Amends**: ADR-0069 (Restart launched a fresh conversation), ADR-0084 (resume was only `start` with `resume: true` on a stopped terminal)

## Context

ADR-0069 said Restart starts another process, not an automatic conversation
resume. ADR-0084 added a pin plus an explicit **Resume last session** on a
*stopped* terminal, and rejected auto-resume after a *daemon* restart —
an unattended relaunch can replay prompts into a changed world.

The menu action **Restart terminal** is the other case: the user is present,
confirmed the dialog, and expects the same Agent CLI conversation back.
On 2026-09-19 that action relaunched Claude Code into an empty chat; the
pin was already on the launch record and was ignored. Recovery was Stop,
then Resume last session.

`POST /api/terminals/{id}/launch/start` with `{"resume":true}` already
relaunches with the pin's verified `ResumeArgs`. Restart prepared the
saved settings *without* those args, then killed the pane.

## Decision

A confirmed Restart on an Agent CLI terminal pins the conversation (the
existing last-chance pin), then prepares the next generation with that
pin's resume recipe — the same one-shot `Overrides.Args` replacement
`start?resume=true` uses — *before* ending the live pane. A failed
prepare still leaves the process running. No pin, or an empty recipe,
keeps today's Restart (current settings, fresh conversation). Start
without `resume`, Stop then Start, and daemon/browser reconnect stay as
ADR-0069/0084 specified: never an unattended relaunch.

## Consequences

Easier: bouncing a stuck TUI, applying a new binary, or cycling a pane
no longer costs the conversation; communication enrollment still sees
the exact resume recipe (`peerResumeExact`).

Harder: pending *argument* overrides do not ride along on a pinned
Restart (the recipe replaces `Args` for that launch; env, path,
executable and tools still apply). A `/new` or Stop then Start is the
fresh-conversation path. A pin of a previous session when this run wrote
no transcript (ADR-0084) is reopened, not discarded.

If the recipe is wrong for a CLI, Restart fails closed (400, pane
alive) the same way an invalid next executable already does.

## Decision table

| Conditions | Action / observable result |
|---|---|
| Live Agent CLI, Restart, pin with resume recipe | Prepare with `ResumeArgs`; kill; launch; same conversation |
| Live Agent CLI, Restart, no pin or empty recipe | Prepare current settings; kill; launch; fresh conversation |
| Restart, prepare fails | 400; live pane untouched |
| Restart, folder/executable invalid | 400; live pane untouched (ADR-0069) |
| Start without `resume` | Fresh conversation (ADR-0084) |
| Stopped terminal, `start` `{resume:true}`, pin exists | Unchanged (now shares `launchWithPinnedSession`) |
| Daemon/browser reconnect | Reconcile only; never relaunch (ADR-0069) |
| Enrolled mailbox | Exact recipe → credential still attaches |

Coverage: `TestLaunchWithPinnedSession`, `TestCLITerminalRestartResumesPinnedSession`
(real tmux; no-pin keeps defaults, pin replaces argv, process replaced, pin kept),
existing `TestCLITerminalResumeDecisionTable` and `TestCLITerminalLifecycleDecisionTable`.

## Alternatives considered

- **Merge resume flags into the saved argument array**: rejected — Codex
  `resume <id>` is a subcommand, and `peerResumeExact` only attaches a
  conversation credential to the exact recipe. Same replace contract as
  ADR-0084.
- **New menu item "Restart and resume" beside Restart**: rejected — the
  user already confirmed Restart on a live Agent CLI; a second item is
  the Stop then Resume path they had to invent.
- **Auto-relaunch after a daemon restart**: still rejected (ADR-0084).
  This ADR is the attended menu action only.
