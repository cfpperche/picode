# Fork agent…

A new agent of the same CLI on a copy of a CLI agent's conversation, with a
task and attachments, in the source's folder or a new worktree on its own
branch. Design: `docs/architecture/cli-session-handoff.md` ("Fork"). First
slice from `feat/agent-fork`: Claude Code, Codex, Grok, OpenCode, Omp.

## Next

- Hermes and Antigravity fork only inside their TUI (`/branch`, `/fork`). Either PiCode's same-CLI copy through their Writers (the Continue in… path, a translated copy) or, for Antigravity, measure whether `agy --conversation <id> -i "/fork"` runs the slash command at start.

## Debts

- [ ] Omp's fork still carries its task as one launch argument (line breaks become spaces; a Restart before its first turn re-sends it): the prompt door has no screen reader for Omp (`doorReaderCLI`), and a blind paste into a TUI still opening could be lost. Measuring Omp's prompt for the door reader would move it to the door.

- [x] Muse Code forks natively through `muse serve` (`session/fork`), opens the copy with `muse resume <id>` and gets its task through the prompt door (branch `feat/muse-fork`, 2026-09-24).

- [x] Pi agents are not offered Fork: a Pi agent owns its session file (`--session` is reserved, `SessionPath`), so `pi --fork` needs the new file assigned to the new agent, not a free launch. **Paid 2026-09-24 (`feat/pi-fork`):** pi's own `--fork` runs once into the new agent's folder before its first start and the file becomes its `SessionPath`; the task goes through the prompt door; verified live with the real pi on a scratch instance.
- [ ] The phone hides Fork agent… (no sheet yet); `web/mobile/src/components/AgentRow.jsx` filters the row.
- [x] Codex, OpenCode and Omp name the copy themselves, so the fork pins on its first turn report; a Restart before that re-runs the fork recipe and sends the task again. **Paid for Codex and OpenCode 2026-09-25 (`feat/fork-task-door`):** the task goes through the prompt door, so a Restart before the pin re-forks without the task. Omp still sends it again (see the next line).
- [x] The task travels as one launch argument (clilaunch.Validate: one line, ≤ 8192 characters): line breaks become spaces and attachments follow as `@path`. A multi-paragraph task loses its layout; the prompt door after launch would keep it. **Paid 2026-09-25 (`feat/fork-task-door`) for Claude Code, Codex, Grok and OpenCode:** their task goes through the verified prompt door once the TUI is ready.
- [x] Lineage (`session_handoffs.mode = "fork"`) is recorded but not shown in the sidebar; the Sessions list labels every outgoing row "continued in". **Paid 2026-09-23 (`feat/fork-lineage`):** agent rows say "fork of <name>" (desktop sidebar and phone Work list). The source row does not list its forks yet.
- [x] While any agent is working in the repository (the usual case), the worktree command is typed, not run (ADR-0078's interlock): the person presses Enter in the git terminal, and the fork follows in the background. A worktree needs no interlock the way a checkout does (it writes refs and `.git/worktrees`, not the shared index); exempting `create-worktree-branch` from it is the owner's call. **Paid 2026-09-23:** the owner approved it; ADR-0202 lets the exact `git worktree add -b` command past the busy check.
- [x] Codex, Grok, OpenCode and Omp forks are verified against `--help`/source and fake-CLI tests, not a real run. Claude Code is verified live in production (2026-09-23, by the owner): Same folder fork, own pinned session, 22 source messages then the task and its image, lineage row, source untouched. **Paid 2026-09-24:** the owner ran real forks with Omp, Codex, Grok and OpenCode in production and confirmed they work.
