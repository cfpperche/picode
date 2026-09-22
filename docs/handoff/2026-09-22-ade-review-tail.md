# 2026-09-22 — feat/ade-review-tail

Origin: this is the third and last branch that fixes the adversarial review of the ADR-0179 work. It follows feat/guest-automations and feat/npm-user-prefix.

Web: System's Agent CLIs section said "none installed yet" until `/api/clis` answered, because the fetch waited behind the Pi catalog. After a failed fetch it said so forever. Now it shows skeleton rows while loading, then the list, or one line: "Could not read the list of CLIs. Open Agent CLIs". Desktop starts the fetch at boot and tracks `clisState`. Mobile now loads the list itself and refetches on `feed.open`, `feed.reset` and `cli.*`. Before this it never refreshed. The Automations banner waits for the catalog and ignores disabled automations, with a new row in `automationsPi.test.js`. A free New agent now takes the CLI's name as its default name. `scripts/qa-mobile-navigation-v2.mjs` follows the new free flow. It is not in `make ci` and was not run.

Docs: AGENTS.md, README, the landing page, getting-started, architecture and the llms intro now say "PiCode itself needs only tmux; installing CLIs uses Node.js and npm". The Install list reads Pi, Claude Code, Codex, OpenCode, Omp in getting-started and agent-clis. agent-clis also drops the claim that PiCode does not manage credentials. The landing page's automations card is scoped. mcp.md explains Connectors for CLIs other than Pi: no per-agent scope, Live only for OpenCode and Hermes, and Sign in runs the CLI's own command. philosophy names the six CLIs that have automatic messaging. The plan is marked as executed. Stale comments in `server.go` and `container.go` are fixed, and a desktop test fixture no longer uses "pi".

Debts added to `docs/handoff/open/multi-cli-ade.md`: the extension cannot send to guest agents, and the providers usage summary needs Pi's catalog. The old free form's dead code was kept, because removing it touches the schema tests.

Verified: `make ci-scoped` PASS. Visual pass 1 FAILED: on mobile the error row wrapped its label and split the link. The fix makes it one sentence with an unbroken link. Pass 2 PASSED on the mobile 390 and desktop 1440 error states and on the scrolled full list of nine rows.
visual-review: PASS (v2-system-clis-error-mobile.png, v2-system-clis-error-desktop.png, v2-system-clis-ok-desktop-bottom.png; overlayAudit ok; card 5/5)

Not verified: the loading skeleton was never captured, because agent-browser can abort or fulfil a route but cannot delay it. It is UNVERIFIED visually.
