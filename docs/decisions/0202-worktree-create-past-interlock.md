# ADR-0202: Creating a worktree runs past the busy-repository interlock

- **Status**: accepted (the owner approved it on 2026-09-23: "aprovado, esse comando está liberado")
- **Date**: 2026-09-23
- **Boundary**: process — which git command PiCode may submit in the user's shell while other agents work in the repository
- **Amends**: ADR-0078 (the run door's interlock), for one command only

## Context

ADR-0078's run door presses Enter on a git command only when nobody else is
writing the repository; otherwise the command is typed and waits for the
person's Enter. Fork agent… (`docs/architecture/cli-session-handoff.md`,
"Fork") creates its worktree through that door, and in a busy repository —
the usual state of `~/picode`, where several agents work at once — every
fork into a new worktree stopped at that Enter.

The interlock protects agents from commands that change the files or the
index they are working on (checkout, reset, merge, stage, commit).
`git worktree add -b <slug> <root>/.worktrees/<slug> <ref>` does neither: it
creates a new folder, a new branch ref and the administrative entry under
`.git/worktrees`, and checks the new folder out with its own index.

## Decision

The run door lets exactly the command `create-worktree-branch` composes
past the busy check: `gitcmd.WorktreeCreate` recognizes `git worktree add
-b <slug> <path> <ref>` with nothing before, after or between, a slug that
is one path segment, a path that is `<root>/.worktrees/<slug>` (or the
relative form), and a ref that is a ref. The server also requires `<root>`
to be the repository the terminal is in. Every other check stays: the
pane must be at a shell prompt, the root must match, and every other
command still meets the interlock.

## Consequences

- Fork agent… into a new worktree runs without a manual Enter while
  other agents work.
- The exemption is recognized from the text, so any caller that types this
  exact command gets it — the same as it would by pressing Enter; the
  interlock was always advisory (ADR-0078).
- If two forks pick the same slug at the same moment, the second `git
  worktree add` fails in the visible terminal; the dialog's slug already
  steps past names the graph knows.
- Git's own ref lock serializes concurrent branch creation, so an agent
  creating a branch at the same instant sees git's usual lock error, not
  corruption.

## Alternatives considered

- **Run `git worktree add` in the service process.** Refused by ADR-0096
  (git runs only in the user's shell, with their credentials and hooks).
- **A per-request `force` flag from the dialog.** Lets any command past
  the interlock, which is broader than the one the owner approved.
- **Keep the manual Enter.** The measured cost: nearly every fork in
  `~/picode` stopped at it.
