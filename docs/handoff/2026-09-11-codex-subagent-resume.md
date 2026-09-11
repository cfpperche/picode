# 2026-09-11 — feat/codex-subagent-resume

Incident: "Resume last session" worked for pi/claude terminals but `codex resume`
exited 1 with "cannot resume an unloaded multi-agent v2 sub-agent through its
parent". Evidence: `comm-d76de7` had pinned sub-agent `01a08c6e-…` (rollout
`thread_source: "subagent"`, parent `01a08246-…`, agent "Gibbs"); the events
table shows aux threads overwriting the parent pin all day; the refusal string
is in the codex 0.154 binary by design.

Fix (two fronts, both covered by tests):
- `hookMapPy` drops codex child-thread markers (`subagent_start`/`subagent_stop`
  events, `parent_thread_id`/`parentThreadId`, `thread_source=subagent`).
- `clisession.CodexSource` skips sub-agent rollouts, so catalog, `Latest`
  (runtime-end pin) and handoff never offer a thread codex refuses to resume.

Decision table (hook identity): parent turn → report state; child markers
(event/parent key/thread_source) → drop; non-subagent `thread_source` (cli) →
report. Rows are in `TestNativeHookIgnoresChildAndDelayedCompletion` and
`TestCodexListSkipsSubagentRollouts`.

Open: the codex hook payload for a live sub-agent was never captured (field
names inferred from the binary's hook-event roster); the file-based front is
exact. Data, not code: a terminal whose pin is already a sub-agent keeps it
until its next native session — repair once via `codex resume 01a08246-…`.
