# 2026-09-23 — feat/mgmt-port-range: the shell grants the daemon's exact origin in its IPC ACL

Pays the debt in `docs/handoff/2026-09-23-mgmt-tray-errors.md` (flipped to [x]): the shell's capability files listed only `https://localhost:8445` / `127.0.0.1:8445` under `remote`, so a daemon on another port (8445–8455 range, or Settings/`PICODE_PORT`) got every IPC command refused in the main and Management windows.

Shipped (2649791b):
- `desktop-shell/src/daemon_acl.rs`: on discovery (`poll_loop`, `main_target`, the Management slow path) re-adds `default.json` and `management.json` via Tauri 2 `add_capability(String)`, identifier suffixed `-port-N`, `remote.urls` exactly the daemon origin (localhost, 127.0.0.1, plus a specific bind host). Skipped for https loopback 8445; marked granted only when every file went in, so a failure retries.
- Deliberately not widened to `localhost:*`.
- `docs/architecture/security-model.md` records the accepted costs: Tauri has no revoke (an old port stays trusted until the shell restarts); `server.json` is same-user writable (not an escalation).

Verified: tests (run as a Windows test exe via interop) show the re-aimed output parses as a tauri `CapabilityFile`, permissions and windows unchanged, bind host added. Adversarial review against tauri 2.11.5 source: no blockers; http-on-8445, bind host and retry-after-failure fixed.
Blind spot: not run inside a restarted Windows shell against a daemon on a non-8445 port; on the default port the grant is skipped and behaviour is unchanged.
visual-review: n/a
Merge: merged on `main` as a876efd9, `make ci` green.

## Next up

- Live check: `make desktop-restart` with the daemon on a non-8445 port; the main and Management windows must answer IPC.
