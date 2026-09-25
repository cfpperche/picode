# 2026-09-24 — feat/version-handshake: ADR-0216 client version handshake

Shipped (b647995c7): out-of-process API clients (`picode-mcp`, `picode-browser-host`,
`picode-cli`) send `X-PiCode-Client: <kind>/<build>; protocol=<n>`; the daemon answers
`/api/*` and `/ws/*` with 426 below `version.MinClientProtocol`, naming what to restart
(`mcptool` passes the sentence to the model verbatim). `/api/version` publishes `protocol`
and `minClientProtocol`. The Windows shell announces `window.__PICODE_SHELL__ =
{version, protocol}` (`SHELL_PROTOCOL=1`); `web/browser/src/lib/shellVersion.js`'s
`shellSupports(n)` lets a feature hide instead of failing, but no feature calls it yet.

Verified: `TestClientGate` decision table, `TestHTTPDaemonSendsTheClientHandshake`,
version parse tests, 4 JS tests, `cargo xwin check`, a scratch instance (`/api/version`
shows `protocol: 1`; a protocol-0 MCP client got 426 with "Restart the agent to load
the new tools"; current client got 200). `make close` PASS. One flake seen once —
`TestCLIRestartPreparationFailureAndWorkspaceCleanup` failed in a `make close` shard
("no server running on /tmp/picode-tmuxtest-…", tmux test server gone), passed 5/5
isolated and on rerun; unrelated to this diff.

Not done: no "update PiCode Desktop" notice — no shell update path exists yet (install
reuses the shell exe on disk); debt recorded in `docs/handoff/open/windows-wsl.md`. The
shell announcement is untested in the real Tauri shell (needs `make desktop-restart`,
owner's call). The communication CLI (`/mcp/communication`) is not gated. Not deployed.

Context: item 3 of 4 in the desktop↔daemon API hardening plan (1 waiting screen and
2 local API gate/ADR-0215 landed; 4 transport is next).
