# Continue in: live CLI validation, 2026-09-20

The menu appeared for Claude Code, Codex, Grok, Hermes, OpenCode and Omp.
Pi replied successfully but did not expose Continue in until an explicit session
listing bound its session. Muse replied successfully but remained unbound even
after stopping. Antigravity remained at onboarding. This is a validation report,
not a fix or an end-to-end cross-CLI handoff acceptance.

## Method

Built main `1889244a` in an isolated worktree and ran scratch
`http://localhost:8471` against task-owned `/tmp/picode-continue-demo.aMo367`.
Installed CLIs used an isolated home directory with copied authentication and
configuration. The prompt was: `Reply with the single word READY. Do not use tools
or change any files.` The observation loop took 90 samples, approximately one
second apart plus API time, without listing Pi sessions during observation.
Measurements concern session binding and menu availability, not model latency.

## Results

| CLI | Conversation result | Session binding and menu |
|---|---|---|
| Pi 0.85.1, fresh xAI Grok 4.6 run | READY | No pin during observation; Continue appeared only after explicit session listing. |
| Claude Code 2.1.278 | READY | Pin already existed before prompt; Continue visible. |
| Codex 0.155.1 | READY | Pin observed 2.590 seconds after actual Enter; Continue visible. |
| Grok 1.0.34 | READY | Pin already existed before prompt; Continue visible. |
| Hermes | READY | Pin observed 1.984 seconds after actual Enter; Continue visible. |
| OpenCode 1.18.31 | Provider 403; no successful reply | Pin observed 1.575 seconds after actual Enter; Continue visible despite failed model request. |
| Omp 18.2.4, fresh run | READY | Pin already existed before prompt; Continue visible. |
| Muse Code 1.3.0 | READY; turn completed in 1m08s | No pin before or after stop; Continue absent. |
| Antigravity | Blocked at Terms of Service and Data Use onboarding | Consent not accepted; Continue absent; conversation not validated. |

Pi's fresh user message was timestamped `12:25:31.758Z` and assistant reply
`12:25:32.810Z`. Continue remained absent in the `12:27:34` before-lookup capture.
The deliberate request to
`GET /api/workspaces/continue-demo-014b8e/sessions?agent=demo-pi-fresh-3b880c`
was followed by `agent.updated` at `12:27:34.832963996Z` and immediate menu
availability. The 122.023-second reply-to-pin interval includes deliberate
waiting and the lookup; it is not an automatic binding latency measurement.

Muse had no pin after the stop API call at `12:26:38.257Z`. Its isolated native
`session-index.db` contained zero rows in `sessions`, and
`GET /api/clis/muse/sessions` returned an empty list, although binary
`.msp-view-v1` artifacts existed. This proves the observed indexing gap, not
which upstream component is responsible.

OpenCode's configured Cloudflare model was forbidden on the account's plan.
Its visible menu therefore proves discoverability, not a successful conversation.
The initial Pi Anthropic token-refresh failure and restarted run were excluded;
the fresh xAI run above is the definitive Pi case. Omp's initial wizard/restart
attempt was invalidated by a database copy and excluded; the fresh auth-snapshot
run is the reported measurement.

## Visual evidence and limits

Actual PNGs were read in a subagent: all nine menus, Antigravity onboarding,
Pi pending/before/after session listing, Pi's hovered target submenu, Muse after
stop, and the captured conversations. Continue is absent in the fresh Pi and
Muse failure captures. No code or UI behavior was changed.

All nine menu audit records report `auditOk: true`; the Pi target submenu audit
also passed. Captured overlays fit the viewport, text is readable, triggers
remain visible, and the hovered Continue row has visible highlighting.

visual-card: (1) overlays inside viewport: yes; (2) items readable: yes;
(3) triggers usable: yes; (4) clipping/double-scroll/dead hover: no observed;
(5) next click obvious: yes where Continue is available, unavailable in the
documented missing-menu cases. visual-review: PASS for captured geometry and
readability; functional availability: FAIL for Pi automatic binding and Muse.

The automatic response regex missed terminal glyphs for Codex, Muse and Grok.
A null `responseSeconds` is not a failed reply: pane evidence confirmed READY;
Codex and Grok also have conversation PNGs. Cross-CLI handoff was not executed.

Evidence is retained under
`/home/goat/picode/var/screenshots/continue-demo-validation/`, including copied
screenshots and `continue-*.jsonl` measurements. No product fix or deployment
is claimed. Follow-up debts belong to `docs/handoff/open/sessions.md`.
