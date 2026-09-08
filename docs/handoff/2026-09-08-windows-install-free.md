# 2026-09-08 — feat/windows-install-free: ADR-0098 amendment, free install path
Shipped: ADR-0098 amended (owner: "não vou investir dinheiro agora").
Azure Trusted Signing validates orgs only in USA/Canada/EU/UK (Q&A, June
2026) so a Brazilian company cannot use it; SignPath Foundation needs an
OSI license without commercial dual-licensing, which PolyForm NC + paid
license is not. Decision: the documented install line becomes
`irm https://cfpperche.github.io/picode/install.ps1 | iex` — repo-hosted,
tag-pinned, SHA256SUMS-verified, `Unblock-File`, then `picode-desktop.exe
install`; no Mark of the Web, so no SmartScreen wall. winget stays a second
door until a clean-VM `winget install` passes without a prompt (winget
stamps MotW on installers; unsigned exes have been seen to stop there).
Release-page download keeps the one-time "More info → Run anyway" sentence.
No CI signing job, no signature check in self-update. Plan phase 2
rewritten (2a `install.ps1` + static checks in `make ci`, 2b winget
experiment); decision table rows replaced. Phases 1 and 3 unchanged.
Verified: docs only; `make close` scoped gates — see close exit in the log.
Not done / debts: `install.ps1` itself is not written (phase 2a); the
Azure account created today is unused and costs nothing idle.
Merge: fast-forward ready.
