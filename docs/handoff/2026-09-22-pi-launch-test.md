# 2026-09-22 — feat/pi-launch-test: the third machine-dependent test, and the port

Found in the CI run for the pushed `main`: `TestAddAgentWithLaunchOverrides` (ADR-0184 slice 1) expects 201 from `POST /api/agents` for a Pi agent with overrides, and a runner with no `pi` answers 400 *"Pi was not found. Check its executable or PATH."* Third test today with this dependency (`opencode`, `omp`, now `pi`) — the debt recorded this evening in `docs/handoff/open/process.md` bit within the hour.
Fix: Pi gets a configured executable through `PUT /api/clis/pi`, the door this file already uses for codex and grok. It also sharpens the reserved-flag refusal above it: that 400 can no longer be about a missing binary.
Proven with a PATH holding no agent CLI (3 runs green) and with the normal PATH; `make ci-scoped` PASS (fmt,vet,hooks,go[4]; 1 path vs main).
No fragment: nothing user-visible (a test).
Also observed, same evening: `make ci` here failed with `409 callback port busy (127.0.0.1:53692)` — the production daemon (pid 4133962) held the fixed OAuth callback port while a signin flow was open, so the *test* could not bind and neither could `make ci`. It cleared by itself minutes later (no listeners on 5369x), nothing was touched, and the candidates are recorded in `docs/handoff/open/agent-cli-credentials.md`.
Blind spot: the runners are the only place the original 400 shows, so the proof is a pruned PATH plus the CI run this push triggers.
