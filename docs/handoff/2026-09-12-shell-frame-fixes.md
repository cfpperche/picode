# 2026-09-12 — feat/shell-frame-fixes: shell-owned frame + full .wslconfig (owner round 2)

Fixes from the owner's live round:

1. Main window had no drag/controls — the previous branch put them in the
   web UI, but the daemon serves the DEPLOYED bundle, not this repo's, so
   they never rendered (and never would without a deploy). The frame is now
   injected by an initialization script into every remote page (local pages
   skipped): min/max/close cluster, drag + dblclick-maximize on the brand
   row, MutationObserver re-wiring. The web/ shell-awareness was reverted —
   the served UI stays browser-shaped and deploy-independent.

2. Config tab: frontend JSON.parsed objects Tauri had already parsed
   ("SyntaxError: [object Object]" on Save) — asData() accepts both shapes.
   wslconfig_read/write are generalized to the full documented .wslconfig
   table (20 keys [wsl2] + 7 [experimental], from learn.microsoft.com
   wsl-config, read via browser this session): spec-driven form, per-kind
   validation in Rust, keys matched by name so user keys parked under an
   unusual section are edited in place, not duplicated. Backup and
   unknown-keys-preserved unchanged. Lib tests: 6.

Verified: cargo xwin build; exe installed and relaunched (PID confirmed);
ci-scoped green. Owner-visual pending (undecorated frame + expanded config).

Debts: values' hints are EN-only; enum lists track the docs, WSL may accept
more values than the selects offer (unknown-keys-preserved covers the file).

Merge: fast-forward ready.
