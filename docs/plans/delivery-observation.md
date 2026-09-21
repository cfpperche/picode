# D1b — integration observation implementation

The owner continued the approved delivery plan on 2026-09-21 after D1a.
ADR-0170's D1 read/receipt boundary is now accepted. The browser and mobile Git
surfaces expose Delivery, backed by the same observer as `picode delivery show`.
No integration/deployment execution authority is added.

Adaptation: t3code's change-to-context links, Paseo's visible review state and
GitHub's explicit blockers, as documented in the delivery-governance benchmark.
Desktop keeps an in-pane detail; mobile uses its existing pushed-detail history.
Deployment stays hidden until D2 has evidence rather than shipping an inert tab.

## Decisions and evidence

| Contract rows / conditions | Implemented result | Automated evidence |
|---|---|---|
| O01 unreadable, canceled or partial scan | Issues and incomplete coverage, no all-clear | ObservationDecisionTable, UnstableAndBoundedObservation, ReceiptConfinement |
| O02 missing target | Choose target; only existing main is default | ObservationDecisionTable; blocked screenshots |
| O03/O04 clean idle vs dirty checkout | Neither means approved; unfinished changes visible | ObservationDecisionTable; deliveryReason tests |
| O05/O06/O07 ancestry relations | Integrated / not integrated / update needed, independent checks | ObservationDecisionTable, DivergenceAndUnknownHistory |
| O08 missing/corrupt evidence | Unknown; retain valid Git facts | ObservationDecisionTable, ReceiptConfinement |
| O09/O10/O11 exact scoped evidence, covered-only reuse, changed inputs | Passed scope / prior scope / needs recheck; never full main CI by inference | CoveredContentReuse; Go ReuseParity plus JS decideReuse parity cases |
| O12 integration succeeded but full checks failed | Integrated + failed | ObservationDecisionTable; scratch populated state |
| O13 current occupancy | Associated names, no authorship or native session claim | DeliveryObservationOwnerAndRoot; scratch registered/observed rows |
| O14/O15 deleted branch or missing object | Retained declaration/land source; missing history unknown | ObservationDecisionTable, DivergenceAndUnknownHistory; producer deleted-ref test |
| O22 external changes without events | Visible 15-second refresh; hidden pauses; reconnect/focus refresh | shared client delivery tests |
| O23 moving refs | Retry once then unknown | UnstableAndBoundedObservation |
| O24 late request / moved root | Disposal ignores old response; root precondition; Follow folder | shared client delivery tests, DeliveryObservationOwnerAndRoot |
| Started attempt without finish | Unknown outcome | DivergenceAndUnknownHistory |
| Bad JSON/version/repository/time/size/permissions/symlink | Reject evidence, expose partial coverage | ReceiptConfinement; producer symlink test |
| Receipt write fails | Preserve CI exit status | producer CI exit-code test |
| Target differs from declaration | Review request is for another target | ReviewTargetAndNoRepositoryHook |
| Repository fsmonitor configured | Observer does not execute it | ReviewTargetAndNoRepositoryHook |

Go names above use `Test` prefixes in `internal/delivery` and `internal/server`.
Producers supplement existing stamps and logs; historical evidence is not invented.
Started/finished records use random IDs and atomic replacement. No retention runs.
The registry's declarations remain in the store with its own event contract.

## Bounds and acceptance limits

Snapshots compare refs before and after, not a transaction over every local file.
A working file or receipt can change immediately after its observation. Evidence
is per-user local data, not a security boundary against an actor able to edit Git
metadata. Reads do not execute repository scripts; status disables fsmonitor.

The pilot reads up to 1,000 receipt files and changes and 32 checkout statuses.
A limit marks coverage incomplete. Older evidence is not deleted; browsing beyond
that bounded receipt window and on-demand checkout status expansion remain open
follow-ups. Full external-provider checks, semantic squash/cherry-pick inclusion,
D2 publication, physical mobile/Windows and authenticated vendor acceptance are
not established by these local fixtures.

Visual evidence lives in `var/screenshots/delivery-observation/` on the root
checkout after closure. Scratch uses separate data, Git fixtures and process;
error/moved-root screens use explicit browser fault injection, not production.
The session handoff records the final screenshot verdict and gate results.
