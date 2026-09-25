# 2026-09-25 — fork-no-task

The owner simplified Fork agent…: a fork opens on its copy of the
conversation and waits; the person gives it its first task in its own
session. The dialog (desktop and phone) keeps Name and Where (new worktree
or same folder).

- Removed end to end: the `prompt`/`files`/`paths` fields of
  `POST /api/agents/{id}/fork-agent` (now `{name, workPath}`; anything else
  is a 400), `deliverForkTask` and its Inbox fallback, `forkPrompt`, the
  attachment staging, `Fork.TaskAfterLaunch` and the prompt parameter of
  `clisession.Forker.ForkArgs`.
- The Pi and Omp narrow readers (ADR-0217) stay: the attach bar and
  messages still use them.
- The task-delivery debts in `docs/handoff/open/agent-fork.md` are moot;
  the topic's intro says so. Hermes/Antigravity fork is still the next item.
- Docs: `docs/architecture/cli-session-handoff.md` (Fork),
  `docs-site/guide/agent-clis.md`.
- Owner still owes a live fork after deploy (any CLI): it should open
  waiting, with no task sent.
