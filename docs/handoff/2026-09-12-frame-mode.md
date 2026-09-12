# 2026-09-12 — feat/frame-mode: the served UI claims the window frame (ADR-0121)

The handshake from the owner-approved plan: the shell sets
__PICODE_SHELL__ on every page and injects a fallback frame (drag wiring
+ controls cluster); the served UI, in shell mode, claims the frame via
data-picode-frame on the root, draws WindowControls in a reserved
top-right slot, and pads .main-tabs/.dash-head clear of it. The injected
script's observer removes the fallback the moment the claim appears.
Neither side depends on the other's deploy cadence: today the fallback
serves; after the owner's next make deploy the integrated frame takes
over with no further shell change.

web: shellFrame.js (+ node test, 297 pass), WindowControls.jsx,
app.css slot rules, benchmarks.md convention (new views must honour the
slot). shell: script flag + claim check + cluster retirement. ADR-0121
accepted (owner approved the plan in-session) with the alternatives and
why each lost. Capability remote trust was ADR'd by reference, not moved.

Verified: make web, cargo xwin build, npm test 297/297, ci-scoped green;
shell exe installed + relaunched (PID confirmed). The integrated frame
itself is owner-visual AFTER their deploy — until then the window runs
the fallback exactly as before.

Debts: page-reporting key stayed in the config table though current docs
dropped it (WSL still honours it); devtools-on-release still on.

Merge: fast-forward ready (after merge main into branch below).
