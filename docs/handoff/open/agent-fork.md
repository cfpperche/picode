# Fork agent…

A new agent of the same CLI on a copy of a CLI agent's conversation, with a
task and attachments, in the source's folder or a new worktree on its own
branch. Design: `docs/architecture/cli-session-handoff.md` ("Fork"). First
slice from `feat/agent-fork`: Claude Code, Codex, Grok, OpenCode, Omp.

## Next

- Muse Code: native fork through `muse serve` (`session/fork`, optional `cutPoint.lastTurnId`, MSP schema 1.3.0), then `muse resume <new-id>` in the new terminal. The method is documented in the schema; the handshake has not been exercised live.
- Hermes and Antigravity fork only inside their TUI (`/branch`, `/fork`). Either PiCode's same-CLI copy through their Writers (the Continue in… path, a translated copy) or, for Antigravity, measure whether `agy --conversation <id> -i "/fork"` runs the slash command at start.

## Debts

- [ ] Pi agents are not offered Fork: a Pi agent owns its session file (`--session` is reserved, `SessionPath`), so `pi --fork` needs the new file assigned to the new agent, not a free launch.
- [ ] The phone hides Fork agent… (no sheet yet); `web/mobile/src/components/AgentRow.jsx` filters the row.
- [ ] Codex, OpenCode and Omp name the copy themselves, so the fork pins on its first turn report; a Restart before that re-runs the fork recipe and sends the task again.
- [ ] The task travels as one launch argument (clilaunch.Validate: one line, ≤ 8192 characters): line breaks become spaces and attachments follow as `@path`. A multi-paragraph task loses its layout; the prompt door after launch would keep it.
- [x] Lineage (`session_handoffs.mode = "fork"`) is recorded but not shown in the sidebar; the Sessions list labels every outgoing row "continued in". **Paid 2026-09-23 (`feat/fork-lineage`):** agent rows say "fork of <name>" (desktop sidebar and phone Work list). The source row does not list its forks yet.
- [x] While any agent is working in the repository (the usual case), the worktree command is typed, not run (ADR-0078's interlock): the person presses Enter in the git terminal, and the fork follows in the background. A worktree needs no interlock the way a checkout does (it writes refs and `.git/worktrees`, not the shared index); exempting `create-worktree-branch` from it is the owner's call. **Paid 2026-09-23:** the owner approved it; ADR-0202 lets the exact `git worktree add -b` command past the busy check.
- [x] Codex, Grok, OpenCode and Omp forks are verified against `--help`/source and fake-CLI tests, not a real run. Claude Code is verified live in production (2026-09-23, by the owner): Same folder fork, own pinned session, 22 source messages then the task and its image, lineage row, source untouched. **Paid 2026-09-24:** the owner ran real forks with Omp, Codex, Grok and OpenCode in production and confirmed they work.
