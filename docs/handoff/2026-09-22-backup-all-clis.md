# 2026-09-22 — feat/backup-all-clis

This branch pays two debts from `docs/handoff/open/multi-cli-ade.md`, which are already flipped there. First, backup covered only `~/.pi`. Second, member containers had the distro's Node with a root-owned npm prefix.

Backup: `internal/backup/clis.go` gives each CLI other than Pi a `clis/<cli>/` folder in the snapshot. The machine settings file is always copied (`clisettings.UserFiles`). The declared login file is copied only when secrets are included, with mode 0600 (`clicreds.CredentialFiles`). The file-based session directory is copied only when sessions are included (`clisession.FileSessionRoots`: Claude Code, Codex, Grok, Omp, Muse Code). SQLite stores are not copied (OpenCode, Hermes, Antigravity). Restore swaps each part back under $HOME (`swapRegular`/`swapTree`; `trees.json` lists the session trees). ADR-0014 is amended, and so is its index row. `docs-site/guide/backup.md` is rewritten for this change. The UI job has a new step, "Copy agent sessions".

Found while testing: the old backup tests did not isolate HOME. With this change they copied the machine's real CLI sessions, and the package went from 0.3 s to 39 s. `testEngine` now sets HOME/XDG to a temp dir (0.4 s). Real sizes on the owner's machine are about 4.6 GB (Codex 2.7 GB, Grok 1.1 GB, Claude 839 MB). The docs record this as a first-snapshot cost, because later snapshots hard-link unchanged files.

Container: `internal/provision/container.go` installs NodeSource Node 22 in the member root. It also writes `usr/etc/npmrc` with `prefix=${HOME}/.local`, which exists only in the container and never touches the member's host `~/.npmrc`. An older root without it is flagged and converged. The NodeSource script moved to `internal/install/nodesource.go`, because provision cannot import desktop. `MachinesDir` became a var so tests can set it.

Tests: `TestSnapshotRestoreOtherCLIs` uses the real declared paths under a temp HOME. It checks that the toggles are respected, that the credential copy is 0600 and that restore puts everything back. Also new: `TestUnderHomeRefusesOutsidePaths` and `TestRootfsStepWantsTheMemberNpmPrefix`.

Verified: `make ci-scoped` PASS. Tested live on a scratch instance with `PUT /api/backup` + `POST /api/backup/now` and sample Claude Code and Codex files. The snapshot held Claude's settings, its credential (mode 600) and its session, plus Codex's session. Visual pass 1 was UNVERIFIED: no folder was set, so the job never ran. Pass 2 PASS: the job dialog showed "Copy agent sessions" and the finished row. visual-review: PASS (v2-backup-job.png, v2-backup-done.png; card 5/5).

Not verified: the container Fix itself, which needs root, debootstrap and network on a shared server. Only its Check is tested.
