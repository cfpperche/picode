# Terminal activity: hooks and the process tree

## Next

- Re-measure pane titles and fix `docs/architecture/direct-session-communication.md` (the "titles carry no status" paragraph). On 2026-09-24 the titles carried a spinner in Claude (`✳` idle, `◐◑` working), Codex and Grok (braille prefix), so that paragraph is stale. ADR-0212 kept the title out of the status on purpose.

## Debts

- [ ] Muse and Antigravity have no runtime lease, so a `!` command in either still shows Open (ADR-0212). The pane walk would need a root without a lease.
- [ ] Omp runs shell builtins (`!sleep 30`) inside its own process, so the tree walk cannot see them. Omp's `user_bash` extension event could report them.
- [ ] The 280px sidebar hides the spinner and the age on working pills (`web/browser/src/styles/app.css` container rule, from eee247e5e). That applies to "Running" as much as to "Working". Decide whether the busy pill keeps its spinner.
