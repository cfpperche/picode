# 2026-09-21 — guide-second-account: the second-account walkthrough, photographed

Shipped: docs-site/guide/providers.md gains "Adding a second account" —
numbered steps, the two traps named (first-run setup before the prompt; the
browser already holding the current account), the naming step, and the
per-vendor doors. Two new pipeline captures seed it: the fixture
(cmd/picode-docs-fixture) now writes a synthetic pi auth.json, a Claude Code
login and two vault rows with cached usage reports, and scripts/docs-shots.mjs
captures the Providers pane in both states (app-providers.png — native strip
plus naming note; app-providers-pi.png — pi's grid with Usage bars and a money
window). No secrets: every token is invented and useless.

Verified: make docs-shots (7 surfaces, gate MARKER_OK first round on both new
ones), docs-check ok, vale clean, test-js green.

visual-review: PASS (both captures read in-session against the walkthrough's
own steps: native strip + naming note visible; Usage bars, money window, and
the unknown+Check rows all present)
Not done: the transient sign-in strip (Open terminal · Check now · Dismiss) is
described but not photographed — capturing it needs the runner to click Sign
in, which spawns a real CLI process in the fixture tmux; left out on purpose.

## Next up

- The capture runner could seed a pre-created sign-in terminal so the strip
  state photographs without spawning a CLI.
