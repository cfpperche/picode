# Project plan: Windows installer (PiCode on Windows + WSL)

Status: approved by the owner 2026-09-16; M1 in progress.
Executes ADR-0098 (accepted 2026-09-08, amended) in the post-ADR-0142 shell
era. No new ADR: no protocol, persistence, security-model or process boundary
is crossed.

## Objective

One documented path from a clean Windows 10/11 machine to working PiCode:
detect WSL and distros, guide through whatever is missing (at most one
reboot, self-resuming at logon), and end with the shell resident in the tray.
Adopt existing distros untouched; never reinstall, replace or unregister
anything the user owns.

## Engineer decisions (the owner may override any)

1. Vehicle: the `install.ps1` one-liner plus exe-driven stages — ADR-0098 as
   written. No MSI, no install wizard in v1.
2. Tag policy: pin per release; the release train bumps the pin.
   Reproducible beats floating.
3. E2E gate: a fresh-local-account pass ships v1; a full clean-machine pass
   (VM or reinstall) is required before winget promotion.
4. Two exes, one fetch: the script downloads the tool and the shell for the
   pinned tag, both SHA256-verified; the Linux binary is fetched by the
   `install-picode` stage once the distro user exists.
5. winget stays a second door until a prompt-free clean install is observed.
6. Per-account installs: `--user <name>` aims the binary and pi at one
   account (pi in its own npm prefix, linked into `~/.local/bin` — no
   profile edit, since no `$` survives the wsl.exe boundary); without the
   flag pi stays a system-wide root install. Unknown accounts fail fast.
7. Failures stay visible: the launcher waits for the elevated child and
   reports its exit code; the last error lands in
   `%ProgramData%\PiCode Desktop\install.log`; the window pauses for Enter
   on failure.

## Milestones and gates

### M1 — Chain complete (on main, tested)

Scope: install finale retargeted to the shell (task verified at the shell,
shell launched); `install-picode` stage; `install-runtime` stage
(Ubuntu-only, check → fix → verify per package); `--user` per-account
installs; parent-waits-for-child elevation with install log and failure
pause; table tests; desktop architecture doc updated.

Status: **code landed 2026-09-19** (branches `feat/win-install-m1`, adopted
after four idle days: the WIP that closes M1 is committed, and the plan
itself is in git). The gate below is the part still open, and it is the
owner's: nothing here can run those scenarios from Linux.

Gate: `make ci` green on main; VM scenarios green (clean-machine path and
adopted-Ubuntu `--user` path, idempotent re-runs) on `picode-test`
(Hyper-V, checkpoint `clean`); `startup-check` clean.

Live validation runs on the VM only. The owner box's first dogfood run
died before any durable write (forensics: shell untouched, task XML
byte-identical, no dpkg activity, `/usr/bin/pi` never created) and further
runs there are frozen — the box is the production machine.

Owner acts: UAC approvals during the VM scenarios (the engineer drives via
PowerShell Direct).

### M2 — Distribution live

Scope: `install.ps1` published on the site (`docs-site/public/install.ps1`);
Windows guide rewritten around the one-liner; release-train pin-bump helper;
next release carries it.

Gate: end-to-end install from the paste with WSL already present (this
exercises script → stages → shell, not the reboot path) — on the VM from a
WSL-ready checkpoint, or on a fresh local account of the owner box.

Owner acts: UAC approvals during the run (or run the paste on the fresh
account); review the guide page.

### M3 — Proven clean + winget

Scope: full clean-machine E2E (no WSL → paste → reboot → resume → tray, no
console window, PiCode URL answers); winget manifest file + submission PR
(owner identity); promotion decision for the documented line.

Gate: observed prompt-free clean install; manifest merged.

Owner acts: joint observation on the `picode-test` VM (provided for M1);
file (or co-file) the winget-pkgs PR; approve promotion of the documented
line.

## Validation (every milestone)

- Focused: `go test ./internal/desktop/ ./internal/install/
  ./internal/provision/`; `make ci-scoped` per slice; `make ci` pre-merge.
  Existing gates, no new harness.
- Regression: preservation-contract checks (equal-version binary untouched,
  existing pi untouched, tmux intact, no unregister) via stage tests plus
  the dogfood re-run.
- Release: `release-notes.mjs --check`, WhatsNew icons test, artifact
  SHA256 + binary smoke per `docs/release-process.md`.

## Risks

- Reboot-resume fragility → stages derive from observed state, so any
  interruption resumes; the RunOnce value and the task are named constants,
  removable by hand.
- apt/NodeSource/npm drift → fail closed naming the missing piece;
  per-package check → fix → verify, never partial-unknown state.
- SmartScreen on a new path → visible immediately at the clean-machine
  gate, not in the field.
- Dirty-main generated docs (bit two releases in a row) → regenerate
  (`make openapi`, `make docs-shots`) before every gate.
- winget review latency or an unsigned-exe stop → it stays a second door;
  the script is the documented path regardless.

## Non-goals

MSI/MSIX/Store packaging, bundled browser, paid signing, the phase-3
dedicated distro, a GUI install wizard (revisit after M3).

## What I need from the owner (gates only)

- M2: the fresh-account install run + guide review.
- M3: the clean machine or VM + joint observation; the winget-pkgs PR
  identity; release/deploy approvals on the normal train (deploy remains
  the owner's call, ADR-0105).
