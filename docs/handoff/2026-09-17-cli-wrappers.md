# 2026-09-17 — feat/cli-wrappers: PATH wrappers for muse+agy; launch divergence gone
Owner ordered PATH wrappers for muse+agy to kill launch divergence (no more
needsWrapper carve-outs). Same shape as other CLIs: dataDir wrapper with
presence lease + native runtime-start/end, per-run real= bake, maintenance
subcommands exec-skip via MUSE_MAINT/AGY_MAINT lists (hermes pattern).
Lifecycle update jobs bypass wrappers (resolveCLIExecutable skips them), verified.
Reporting unchanged by design: agy keeps its title reporter (now WITH a runtime
behind it, so reports take the native path with precise pins); muse stays
honestly Open (hook env scrubbed, Fatia 6 verdict stands) but launches uniformly.
Seed bumped v2 to v3 so muse joins seeding (empty configs only). Plan Files
carry the wrapper; display dedupes it.
Bugs caught by the tests, honestly told: first draft set picode_tui=0 AFTER
wrapperLifecycle, which owns the default and had already fired runtime-start —
fixed by exec-skip before it; then every maintenance case misread as leased
because the agent shell running the suite exports real PICODE_TUI_RUN_ID and
the test inherited it — harness now scrubs TUI vars (fourth ambient-var bite
this week). Decision table covers 17 lease cases.
Verified: ci-scoped PASS; scratch with real vendor binaries (both terminals
boot through the wrapper — muse shows its trust prompt, agy its OAuth flow;
PUT on clean, wrappers installed, reporter intact); owner files untouched after.
Real agy turn through the wrapper NOT run (no auth in scratch HOME); native-path
proof awaits the first owner turn in production. Deploy NOT done (owner's call).
uiux-review: PASS (no JSX changed)
visual-review: PASS (no UI changed; TUI trust/OAuth screens read live, not recorded)
