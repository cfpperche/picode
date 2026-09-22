# Performance — the Feel bar is a promise nothing measures

Study: `docs/benchmarks/2026-09-22-zed.md` §C.

## Next

- Owner call on the two cheap gates the Zed study proposes: a gzip bundle budget in `ci-gates`, and one p95 keystroke-to-echo probe through websocket + tmux on a scratch instance. Neither needs an ADR; both need a threshold only the owner can set.

## Debts

- [ ] `docs/benchmarks.md:127` promises "interactions respond in <100ms" and nothing measures it — no gate in `ci-gates` (`hooks-check fmt-check vet test test-js desktop-test build ci-docs vale`) can fail on latency or size.
- [ ] The browser bundle is unbudgeted. Measured 2026-09-22: entry `index-*.js` 3.4 MB raw / 1.0 MB gzip, `subset-shared.chunk-*.js` 1.8 MB / 0.7 MB, whole `browser/` build 15 MB across 138 files. An unbudgeted number only moves one way.
- [ ] No terminal echo latency figure exists for any PiCode instance, so the one number a terminal product cannot fake is unknown.
