# 2026-09-08 — feat/windows-clean-install: ADR-0098 + plan, docs only
Shipped: ADR-0098 (accepted by the owner in chat, 2026-09-08) and
`docs/plans/windows-clean-install.md`. Decision: `picode-desktop.exe` stays
the only Windows deliverable and finishes the clean-machine chain itself —
two new bootstrap stages (`install-picode`: same-tag Linux asset, SHA256
verified, into `~/.local/bin`; `install-runtime`: apt tmux/git/curl/mkcert,
NodeSource Node at pi's `engines` major, `npm install -g` pi with the
ADR-0093 argv), confirmation before touching an adopted distro. Signing via
Azure Trusted Signing and a winget manifest live in CI. Imported PiCode
distro from a CI-built rootfs is phase 3, opt-in, needs its own approval
(amends ADR-0020's "adopted, never recreated"). No bundled browser, no MSI.
Why: `PicodePath` fails on any distro without picode and provision's "pi is
on PATH" only reports, so the ADR-0020 exe stops one step short for anyone
but the owner. mkcert CA import to the Windows store already exists.
Verified: docs only; `make close` (scoped gates) — see close exit in the log.
Not done / debts: no code yet — phase 1 is the next implementation branch.
Owner actions: Azure Trusted Signing account + repo secrets; clean Windows
11 VM acceptance run. Plan path fixed to `docs-site/guide/` after the rename.
Merge: fast-forward ready.
