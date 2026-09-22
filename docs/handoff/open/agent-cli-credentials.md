# Agent CLI credentials — one vault, then guided use
Plan: docs/plans/agent-cli-credentials.md

## Next

- Answer the six owner questions under `## Open questions (owner)` in `docs/plans/agent-cli-credentials.md` (encryption, concurrency, pi folds in, harvest, import of a rotating login, account per provider vs per CLI) before step 1 starts.
- Seed the two ADRs the plan names in the branch that implements step 1 — `credentials-vault` (persistence + security model) and `credential-injection` (process + security model).

## Debts

- [ ] **The OAuth callback port is fixed, so a live daemon blocks every other
  PiCode on the machine.** Measured 2026-09-22: the production daemon
  (`/home/goat/.local/bin/picode`, pid 4133962) held `127.0.0.1:53692` with an
  open signin flow, and `TestOmpSigninStartsBrowserOauth` then failed 5/5 with
  409 *"callback port busy"* — `make ci` cannot pass on that machine while the
  flow is open, and a second instance's own signin would 409 the same way. The
  port is fixed because the provider's redirect URI is registered for it, so the
  candidates are: release the listener when a flow is abandoned (a timeout), say
  *who* holds it in the error (the daemon's pid is knowable), and let the test
  skip with that reason instead of failing the gate. Not touched: the holder was
  the owner's live daemon with terminals attached.
- [ ] Muse Code and Antigravity credential paths were probed on this machine and neither vendor publishes documentation to confirm them — re-probe at implementation (`docs/benchmarks/2026-09-20-agent-cli-credentials.md`).
- [ ] OpenCode `OPENCODE_CONFIG*` semantics (file vs directory, precedence over the vendor's own `auth.json`) are unverified against a real binary (`docs/plans/agent-cli-credentials.md`).
- [ ] Claude Code publishes no non-interactive status command in the study (`/status` is in-REPL only) — re-check before wiring vendor verify into its pane (`docs/plans/agent-cli-credentials.md` §2.7).
- [ ] The study's routing of `internal/usage` and roster code for the nine CLIs changes nothing until step 1 lands: usage still reads the active slot per provider, not per CLI (`docs/benchmarks/2026-09-20-agent-cli-credentials.md`).
