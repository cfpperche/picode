# 2026-09-24 — feat/shell-waiting-screen: desktop shell gets a live waiting screen

Shipped (aa980d1b6): the main window opens at once on `ui/waiting.html` instead of picking a
target once at setup (dead `offline.html`, or a 30 s block behind the readiness gate before the
window existed). The health loop publishes stages — Linux starting, PiCode starting/restarting,
not answering, certificate untrusted, opening — and navigates to `/desktop/` once the daemon
answers. `capabilities/waiting.json` is local-only (the daemon's `/desktop/` cannot call the new
commands); the cert probe is strict (no `-k`); `await_desktop_ready` no longer navigates onto a 404.

Verified: `make desktop-test` (6 new waitstate tests), `cargo xwin check` for the Windows target,
`make ci-scoped` PASS, `make close` PASS. Headless render of all 8 stages, dark/light, 720x480,
read by a subagent: first pass FAIL (failed-trust copy vs primary action, orphan wrap, 9px shift
on opening) — fixed; second pass PASS. visual-review: PASS (v2-min-untrusted-failed.png,
v2-untrusted.png, v2-opening.png, v2-starting.png; card 5/5).

Honesty: the live half has NOT run in the real Tauri shell — eval push, `navigate`, the
local-only capability, the `certutil -user` prompt. Owner check is the new `- [ ]` in
`docs/handoff/open/windows-wsl.md`. Not deployed.

Context: item 1 of 4 hardening the desktop↔daemon HTTP API — 2: security (Sec-Fetch on GETs, no
loopback session minting cross-site, exact Host allowlist, owner approved narrowing); 3: version
handshake; 4: transport (cert SAN 127.0.0.1/::1, shell token, OAuth listener timeout).
