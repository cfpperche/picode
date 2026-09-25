# 2026-09-24 — feat/lifecycle-menu-parity: lifecycle menu restored for omp/grok/hermes/muse
Owner reported (screenshots) that Omp lacked the ••• lifecycle menu on Agent
CLIs while Claude Code had it; correct is all installed CLIs with a known
plan. Live API showed FOUR CLIs hit method "unknown": omp, grok, hermes,
muse — executable resolved to PiCode's own shim in ~/.picode/bin, because
isCLIWrapper matched only the "# PiCode intercept" header; the
integration/plugin shims say "integration"/"plugin". DetectMethod saw an
unclassifiable script → For() refused → all lifecycle flags false → menu
hidden (agy escaped only via its basename rule).
Fix: isCLIWrapper now requires "#!/bin/sh" + a "# PiCode " comment — the
shape every wrapper writer emits (intercept, open-url, tmux guard, five
integration flavors). Vendor scripts without the header still resolve as
the executable.
Verified: new TestEveryWrapperFlavorKeepsTheLifecycleMenu pins each flavor
through resolveCLIExecutable → describeLifecycle (managed plan asserted);
ci-scoped green; scratch instance (qa-scratch fixproof): with the real
production shims copied in and prepended to each CLI's config PATH, all
four resolve to the real binary (omp→npm, grok/muse→vendor, hermes→git)
with canUpdate/canReinstall true — the same input shape that yields
"unknown" on the production binary. Scratch stopped. No visual-review
needed: the menu's visibility is a direct function of these flags and the
menu rendering itself is unchanged UI.
Deploy NOT done (owner's call): make land BRANCH=lifecycle-menu-parity from
the root, then make deploy when wanted.
