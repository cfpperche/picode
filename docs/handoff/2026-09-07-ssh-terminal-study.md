# 2026-09-07 — ssh-terminal-study: SSH access to agent terminals (study only)

Shipped: `docs/benchmarks/2026-09-07-ssh-terminal.md` + README index row.
Receipts from live docs: Coder `coder ssh`/`config-ssh`, Codespaces
`gh cs ssh` (image must ship sshd), Ona/Gitpod Classic (SSH-first), VS Code
Remote-SSH, Tailscale SSH (identity/ACL/recording), Terminal-Bench and
Daytona as the no-SSH contrast. Repo facts: ADR-0002 already puts every
terminal in tmux, so `tmux attach` over the user's own SSH works today,
undocumented; auth stays host-side (ADR-0049 untouched); Tailscale SSH
matches ADR-0050/0051 deployments.

Verified: receipts fetched from live docs pages (browser, 2026-09-07);
`make ci-scoped` PASS in the worktree; no code changed.

visual-review: n/a (no UI).

Not done / debts: recommendation is owner call — document (www/ page +
later optional copy-attach affordance), refuse bundled sshd/gateway.
Open: tmux socket isolation on shared boxes (0051), `attach -r` safety,
flight-recorder blind spot to SSH clients. No ADR filed; no CHANGELOG
(study-only, precedent 6da0473a).

Merge: fast-forward ready.
