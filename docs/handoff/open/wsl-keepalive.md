# WSL keepalive

## Debts

- [ ] Attribute the two instance terminations that did not coincide with a
  desktop-restart (2026-09-18 16:44 and 17:06 local): enable WSL's
  Operational event log on Windows (Task Scheduler log was disabled) and
  check whether `sparseVhd=true` / `autoMemoryReclaim` in `.wslconfig`
  triggers WSL-side reclaims that a keepalive cannot prevent. If sparse
  reclaim is confirmed, the owner decides disk-vs-stability (C: had 27 GB
  free).
- [ ] The Rust `TaskEnsure` register/run paths do real Windows IO and are
  compile-checked only (`cargo xwin check`); the live deployment
  (registered 2026-09-18, Running) is the test. A Windows CI runner would
  let the unit suite cover them.

## Measured (2026-09-18)

- The 18:05:05 termination is fully attributed: `make desktop-restart`
  killed the shell → its job-object keepalive died → WSL reclaimed the
  distro 31s into the new shell's boot. Deploys themselves were clean all
  day (forensics: "all N session(s) still alive", ~15 times).
- `schtasks /create /sc onlogon` needs admin; `/sc once /sd` is
  locale-brittle — the register path uses PowerShell cmdlets
  (non-elevated, proven live).
