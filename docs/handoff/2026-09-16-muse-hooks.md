# 2026-09-16 — muse-hooks: Fatia 6 blocked; session-report 409 fallback; HOME isolation
Owner approved opening Fatia 6 (muse activity) after R3233 spiked WITH a hook surface (777 hook strings vs zero in R3057).
Schema measured live with backup/restore of owner settings: Claude-shaped global hooks block, hook_event_name stdin JSON
(SessionStart/UserPromptSubmit/PreToolUse/PostToolUse/Stop/PermissionRequest), no matcher needed, always exit 0.
VERDICT: BLOCKED, do not ship muse reporting. The CLI scrubs the hook environment to PATH alone (instrumented proof:
hook under CLI sees exactly one var) and managed_hooks_env_vars is enterprise-policy tier, undeclared from userland —
so a settings reporter cannot attribute a report to its terminal and the daemon contract is per-terminal by design.
Six live turns + bisection (marker fires, wrapper content runs manually, CLI invokes wrapper but POSTs nothing
without TERM_ID) prove it. Reverted the install/merge/plan/seed/UI-flip; kept no dead code.
Shipped instead: session reports carrying a session id but no runtime entry now fall back to plain terminal states
instead of 409 (decision table TestSessionReportWithoutRuntimeFallsBack) — this retro-fixes Antigravity daemon-side
activity, which 409ed the same way and read Open forever; verified live on a scratch daemon (terminal.state
working/idle broadcast for a real muse-shaped report).
Test-hygiene incident, owned honestly: server boot seeds Activity on and syncs integration files, and tests without HOME
isolation merged dead /tmp hook paths into the OWNER's real muse+agy settings twice (found by md5 watch, cleaned
byte-identical both times, verified). Root cause was ImportCLIConfigs creating rows + seed flipping them + startup
sync installing; only agy/muse write user HOME (others are dataDir-scoped). Fix: whole server suite runs under
throwaway HOME+XDG via TestMain (guardrail TestSuiteIsHomeIsolated) + cleanupServer isolates XDG_CONFIG_HOME
(shared XDG dir made one install read as foreign in the next). Full server package green, owner files untouched after.
Live-validation leftovers all removed: owner settings restored (muse 5 keys, agy production title), 12 probe muse
sessions deleted (none indexed), trust entry /tmp/agyprobe removed, live daemon+tmux+tmp gone. Ambient PICODE_TUI_*
vars from the agent shell bit three times (mapper test hermeticity, manual runs, contaminated tmux server env) —
pin empty TUI vars in any mapper/hook test. Deploy NOT done (owner's call).
uiux-review: PASS (no JSX changed)
visual-review: PASS (no UI changed; live feed evidence read as JSON, not a screenshot)
