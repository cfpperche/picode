# Delivery flow

Plan: docs/plans/delivery-flow.md
Research: docs/benchmarks/2026-09-21-delivery-governance.md
D0 evidence: docs/plans/delivery-flow-evidence.md
D0 surface contract: docs/plans/delivery-flow-design.md

## Next

- Measure the D1+D2 observation pilot (D5): terminal lookups and elapsed human time on the same supervision tasks, and every discrepancy between the lanes and their sources. D3/D4 execution authority needs its own owner decision.
- Zero-unpublished wording: with everything published the lane reads "Integrated, not published: 0 changes" — say "All integrated changes are in the running version." instead (`web/shared/domain/delivery.js`, `environmentLane`).

## Debts

- [ ] Native authenticated sessions for the nine vendors have not exercised the delivery command/MCP end to end; common catalog-principal and CLI-to-daemon fixtures are covered. The existing tool picker has no delivery option; explicit MCP selection uses client configuration.
- [ ] Historical land/deploy queue wait has no authoritative start timestamp; D0 measured seven source lookups only. Capture owner-task time in the D1+D2 pilot and operation wait after D3/D4.
- [ ] D1b reports incomplete coverage after 1,000 receipt files/changes or 32 checkout statuses; add paginated older evidence and on-demand omitted checkout reads before scaling beyond the measured local pilot. No evidence is pruned.
- [ ] Native Windows ACL inheritance and physical mobile behavior were not exercised by the Linux scratch observation tests.
- [x] D2's deployment producer (`internal/install`, around `picode deploy`) is verified against a faked `systemctl`/`Run` boundary; no real deploy has written a receipt yet, so the first live one is the owner's deploy. **Paid 2026-09-21**: two forced deploys wrote receipts — the second from a clean tree records `outcome: passed`, `clean: true`, `requested == built == revisionBefore == observedRevision`, `bootBefore`/`bootAfter` from the two boots, and the daemon answers `/api/version` with that exact revision.
- [ ] D2 observes the local instance only: a remote or external environment remains unknown by design, and `picode delivery show` reads no environment, so its `publication` field stays `unknown`.
- [ ] A deploy from an unclean checkout records `clean: false`, and the Deployment lane then reads `artifact: dirty` with "included changes are unconfirmed" (`make docs-shots` failing mid-run is enough — it leaves tracked captures modified). Note it in the public guide, and fix `docs-shots` (`app-mobile-inbox` waitText never appears; `/tmp/picode-deploy-shots.log`) so a deploy does not silently downgrade its own provenance.
