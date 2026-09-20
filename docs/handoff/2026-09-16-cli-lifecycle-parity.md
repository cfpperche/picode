# 2026-09-16 — feat/cli-lifecycle-parity: muse/agy Update button and full lifecycle menu
Owner reported from screenshots that muse/agy pages lack the Update button
and full lifecycle menu vs other CLIs. Root cause: both clilifecycle plans
carried only latestFrom channel (check-only), so canUpdate/canReinstall were
false and uninstall was none.
Fix: agy runs its own `agy update` (verified in `--help`; the binary carries
an "Update failed, please install from website" string, so the command is
the vendor updater). Muse ships no update argv, so the plan carries env
`MUSE_LAUNCHER_INSTALL=1` on a bare run (verified against the installed
launcher script: INSTALL=1 runs update_binary and exits 0 before arg
parsing; also confirmed `MUSE_NO_AUTO_UPDATE` does not block that path).
New Plan.UpdateEnv + CanUpdate/CanReinstall predicates (env counts as
offered); server overlays the env with drop-first semantics via existing
dropEnv. Both uninstalls are guided with vendor docs links (neither vendor
ships an uninstaller).
Verified: decision tables extended, resolveLifecycleExec overlay unit test,
catalog flags on scratch with real vendor installs, desktop screenshot read
(blue Update button + open menu with Check for updates/Reinstall + guided
uninstall line). Muse correctly shows no Update button (1.3.0 is latest on
its channel). Real update jobs were NOT run (would mutate owner binaries);
first owner click proves the runtime path, output lands in the job card.
Pre-existing: owner production shows "Activity reporting files need repair"
on agy (seed v2 enabled config without installing files); one click on
Repair integration fixes it. Deploy NOT done (owner's call).
visual-review: PASS (var/screenshots/agy-update-button.png read; card 5/5)
