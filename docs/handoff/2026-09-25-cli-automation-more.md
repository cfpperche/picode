# 2026-09-25 — cli-automation-more: automation `start` runs on OpenCode and Omp; honest cost where unmeasured

Follow-up to ADR-0217 (still proposed, updated in place). The owner asked why only four CLIs ran `start`: OpenCode had a reader and hooks all along (the first count came from the wrong list); Omp needed a reader.
Shipped: OpenCode and Omp join `store.UnattendedCLIs` / JS `START_CLIS`. Readers measured live: Omp 18.2.11's input is the screen's last row `╰─ ` + text (cursor 3 + width) under its ` > ` status bar; OpenCode 1.18.32's new-session screen puts text at column 6 between bar rows, grey placeholder (38;2;128;128;128), bright typed text. `peerOmpInput` / `peerOpenCodeHomeInput` + draft cases in `peer_attention.go`; fixtures `internal/server/testdata/composer-{empty,draft}-{omp,opencode}.json` are real snapshots; `omp` added to `doorReaderCLI`.
Found: `climetrics.MeterSessionFile` prices only pi, claude-code, codex, omp, muse — OpenCode, Grok and Hermes runs had no cost and the cap could not stop them (OpenCode showed $0.00). Now the runs table shows "—" with a tooltip, and the editor hint and the detail's "Max cost per run" say the limit is not enforced; JS `METERED_CLIS` is pinned to the new `climetrics.Metered` by `TestMeteredCLIsMatchTheEditor`.
Still refused: Muse (hooks not attributable, no end-of-turn signal) and Antigravity (composer unmeasured).
Verified: server/store/JS tests; live on a scratch with copied logins: OpenCode 2/2 done, Omp 2/2 done (~$0.0016 each); terminals closed after, copied Omp/OpenCode logins deleted from the scratch HOME.
QA mishap: a second `qa-scratch start` without a stop left the first daemon running without its port file; ended by its exact pid. The tmux guard correctly refused to kill a session this terminal did not create.
Blind spots: cost cap on OpenCode/Grok/Hermes is advisory only (no meter); fixtures cover the versions measured, not later TUI redesigns.
visual-review: PASS (one non-blocking finding fixed: the uncapped-cost fact on the detail).
Not deployed (owner's call). Merge: through `make land` after merging `main` and rerunning `make close` (ADR-0124).
