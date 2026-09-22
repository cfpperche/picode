# 2026-09-22 — feat/providers-hide-empty: the providers table lists what the user has

Shipped: `CliCredentials.jsx` (browser and mobile) filters the roster before the table. A provider shows only when it has an account, a custom definition, a CLI login waiting to be imported, or an import error. The rest of the roster stays in the Add dialog's select: pi's whole catalog, or a guest CLI's declaration. With nothing to show, the pane says "No provider credentials for <CLI> yet." under the bar's Add. The API contract is unchanged (ADR-0169's roster still carries every provider). `docs/architecture/cli-providers.md` updated.

Why: the owner saw about 20 "<name> · No accounts yet." lines on Pi's Providers tab. Hermes, OpenCode and Omp had the same lines on a shorter list.

Verified: `make ci-scoped` PASS (fmt, vet, hooks, test-js, build). Scratch instance at 1440px and 390px: Pi and all eight guests show 0 "No accounts yet" lines. The empty state was checked with a stubbed `/api/credentials?cli=codex`, because the scratch had no CLI with zero accounts. The Add dialogs still list empty providers, and the overlay audit is ok on all three. Blind spot: the native-login-to-import row was not seen live.
visual-review: PASS (provhide-pi-desktop.png, provhide-codex-empty.png, provhide-pi-add.png; overlayAudit ok; card 5/5)

Seen, not caused here: in the key-style Add dialog, the "Saved to this machine's vault." subtitle sits flush on the "Provider" label.

Merge: fast-forward ready.
