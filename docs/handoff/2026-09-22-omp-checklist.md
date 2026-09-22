# 2026-09-22 — feat/omp-checklist: omp's todo plan mirrors into the sidebar checklist
Shipped: a second omp extension, the checklist mirror (ADR-0055), rides the same `-e` branch as the
terminal-state extension (internal/server/term_intercept.go; cli_plan.go wires it): it observes omp's
native `todo` tool_result and session_start, maps omp's five statuses onto PiCode's three, and POSTs
the committed phases to the terminal's (or bound agent's) checklist routes — the sidebar card shows
the current step, the disclosure the full list, the way `pi-checklist` serves pi. No new tool is
registered and nothing is gated; uninstall removes both extension files.
Verified: decision-table test runs the generated template under node against a recording daemon
(omp_checklist_test.go); wrapper shape and uninstall pinned in cli_wrapper_test.go. Scoped CI green
except TestUsageSummaryIsCacheOnlyAndSaysUnknown, which fails identically on main without this diff
(login-state dependent). Live on production: omp terminal checklist-mirror-2ef6f2 (workspace
picode-5fd7eb, ~/picode) loaded the extension from user-scope ~/.omp/agent/settings.json and mirrored
todo init/start/done into GET /api/terminals/checklist-mirror-2ef6f2/checklist (status mapping and
sessionId correct). Blind spot: validated via user-scope settings, not the `-e` injection itself.
visual-review: n/a (server-side injection; the checklist UI already exists)
Merge: fast-forward ready.

## Next up

- After this lands AND deploys, remove the omp-checklist validation entry from ~/.omp/agent/settings.json — the `-e` copy would then double-POST.

## Debts

- Todo plans over 50 tasks clip to the store cap (maxChecklistItems=50, internal/store/checklists.go);
  the TUI keeps full fidelity.
