# 2026-09-20 — agent-tui-complete: unify agent terminal surfaces

Shipped: one shared headless terminal runtime and identity resolver now back
Pi legacy TUI, Agent CLIs, mobile TerminalScreen, browser tabs and Canvas.
Menus, attachments, touch scroll, recovery, reconnect, lifecycle actions and
Pi Chat/Terminal toolbar icons use the same terminal surface and session.
Verified: `make ci-scoped` PASS (hooks, vet, 888 JS tests, package tests,
browser/desktop/mobile/Go builds); scratch QA used real tmux/WebSockets and
the harmless fixture for Pi/Claude/Codex, 10 checks and 15 screenshots PASS.
Adversarial review found and fixed persisted-Chat navigation and Canvas link
callback retention. Visual review PASS at mobile 390/320 and browser.
Blind spots: no physical iPhone/IME, native Windows shell, or authenticated
vendor completion; same-ID session replacement has source/unit coverage only.
Merge: pending fast-forward after main catch-up; deploy intentionally not run.

## Next up

- Run owner-controlled land and then decide when to deploy.

## Debts

- Refresh the browser Canvas Find fixture/search assertion in a follow-up.
