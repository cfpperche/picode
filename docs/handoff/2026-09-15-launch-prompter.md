# 2026-09-15 — launch-prompter: Muse Code and Antigravity as brief handoff targets

Fatia 1 of docs/plans/launch-muse-agy.md: Muse Code and Antigravity as brief handoff targets (owner approved, 2026-09-15).
Shipped: two new files internal/clisession/muse_io.go and agy_io.go with PromptArgs — muse passes the brief as a positional prompt (verified Muse Code 1.3.0: pass a prompt to start a session), agy uses --prompt-interactive (verified 1.2.3; --prompt would answer once and exit so it is not the shape); both ignore the pre-assigned session id. TestPromptArgs table extended with both rows.
No server or UI change: CapabilitiesOf discovers Prompter by interface assertion so /api/clis advertises prompt:true for both, and the generic brief path serves them.
Verified on a scratch with a real pi session copied into the scratch home: catalog shows muse/agy sessions caps {list:true, prompt:true}; POST /api/clis/pi/sessions/handoff/preview to=muse and to=agy return 200 with modes:[brief]; mode=native returns 409 'Muse Code has no native import; use a brief.' (same for Antigravity). The scratch isolates HOME so seeding one real session file was needed.
Visible effect: Continue in… (session rows, terminal rows, pane menu) lists Muse Code and Antigravity with brief as the only mode; their own terminals still have no Continue in… (no Reader yet — Fatia 2) and no Launch settings (no adapter yet — Fatia 3).
visual-review: N/A (no rendered surface changed; verification is API + unit)
