# Pi interactive runtime validation

## Debts

- [ ] Complete an authenticated Pi turn on a scratch instance and verify the final idle state against the native activity events. Scratch working/needs-you states were exercised, but `invalid_grant` prevented a successful model response; no successful LLM completion is claimed. ADR-0162's activity row is covered by runtime/state tests, with live provider completion still unverified.
- [ ] Exercise ADR-0162's stop/restart, failed shutdown and TUI/RPC exclusivity rows on hosted macOS. Linux isolated-tmux matrix tests exercise these lifecycle boundaries; Darwin arm64 and Windows amd64 cross-builds passed, but cross-compilation does not establish hosted runtime behavior.
