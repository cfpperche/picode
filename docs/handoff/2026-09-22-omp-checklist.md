# 2026-09-22 — feat/omp-checklist: omp's todo plan mirrors into the sidebar checklist
Shipped: a second omp extension, the checklist mirror (ADR-0055), rides the same `-e` branch as the
terminal-state extension (internal/server/term_intercept.go; cli_plan.go wires it): it observes omp's
native `todo` tool_result and session_start, maps omp's five statuses onto PiCode's three, and POSTs
the committed phases to the terminal's (or bound agent's) checklist routes — the sidebar card shows
the current step, the disclosure the full list, the way `pi-checklist` serves pi. No new tool is
registered and nothing is gated; uninstall removes both extension files.
Verified: decision-table test runs the generated template under node against a recording daemon
(omp_checklist_test.go); wrapper shape and uninstall pinned in cli_wrapper_test.go. Scoped CI green
— the usage-summary test now scrubs catalog.APIKeyEnvVars (ambient terminal env broke it everywhere).
Repeated `-e` on omp 18.2.8 probed live. Live on production (workspace picode-5fd7eb, ~/picode): omp
terminals loaded the extension from Omp's user layer and mirrored todo init/start/done into the
terminal checklist route (status mapping and sessionId correct); re-verified after the entry moved
to the canonical store (`omp config set extensions`, visible in the Packages pane under Global).
Blind spot: the live runs used the user-layer entry, not the wrapper's own -e.
visual-review: n/a (server-side injection; the checklist UI already exists)

## Next up

- Resolved in feat/omp-checklist-pkg: the `-e` injection was reverted in favor of the installable
  package, so there is no double-POST window — the plugin-store install is the whole story.

## Debts

- Todo plans over 50 tasks clip to the store cap (maxChecklistItems=50, internal/store/checklists.go);
  the TUI keeps full fidelity.
