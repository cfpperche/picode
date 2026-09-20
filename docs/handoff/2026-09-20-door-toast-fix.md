# 2026-09-20 — door-toast-fix: refusal shown in the card; toast layering
Owner report: the door refusal toast painted behind the attach card and
was only found after closing it. Changes: (1) door refusals render inline
(role=alert) in the desktop attach bar and the mobile send sheet — the
message names its fix where the Send happened; (2) the sonner toaster
gets an explicit layer (z-index 1100) above in-pane stacking contexts and
page overlays (dialogs 82/83, img-lite 200).
Note: production had been running another session's unlanded build
(d1e6e90, browser-pending-roadmap) which lacked the Fatia F gate fix —
the "same error" was partly that stale binary. Deploying main restores
the three-state gate.
Verified: ci-scoped PASS; build green.
