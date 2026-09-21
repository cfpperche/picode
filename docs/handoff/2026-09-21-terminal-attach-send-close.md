# 2026-09-21 — feat/terminal-attach-send-close: verified TUI submit, bar closes on send
Shipped: `deliverViaPaste` polls the live pane until the composer reads
empty (per-CLI reader; omp maps to pi), retries Enter once against the
live pane, and answers 502 `staged` instead of a blind 200 — the bar
keeps the text staged with the reason. `TermAttachBar.send` closes the
bar and refocuses the pane on every 200; failures keep it open.
Verified: `make ci-scoped` PASS (incl. server suite); new
`TestTuiReaderFor`; `make web` ok; scratch live: Send on a doorless pane
→ 409 keeps the bar open with staged chip + in-card error (screenshot
read), overlayAudit ok.
Blind spots: success-close and staged-retry need a live CLI door (no
authenticated CLI in scratch); keyed on unit + static review.
visual-review: PASS (attach-send-error-keeps-open read; card 5/5; audit ok)
Not done: none.
Merge: fast-forward ready. Deploy: owner's order (not run).
