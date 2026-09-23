# 2026-09-23 — feat/wsl-mgmt-root: Distro tab (/etc/wsl.conf) and system caches, as root

Items 3 and 8 of the owner's 8-item WSL Management list; second of four branches (next: move/export with space checks; disk history). Owner chose `wsl.exe -u root` over a sudo password (their sudo asks for one) — ADR-0198.

Shipped (a0e94a95):
- Distro tab edits /etc/wsl.conf as root; system caches (apt, journal) join the scan's third stage and Clean.
- `wsl.exe -- …` goes through a shell that eats `$`; `--exec` passes argv intact (measured with a Go probe exe), so every root call uses `--exec`.

Verified live on the owner's machine (no restart): root read works; system caches measured (apt 136 MB, journal 2.1 GB); nonexistent account and root refused; write without `--yes` refused; set-then-remove round trip leaves /etc/wsl.conf byte-identical; /etc/wsl.conf.picode-orig equals the original (an /etc/wsl.conf.bak from the test writes remains, harmless). Blind spot: system-clean (apt-get clean, journal vacuum) is unit-tested and stubbed only; the Distro tab was never driven through the real shell.
Adversarial review: no injection path (Windows argv escaping checked); fixed True/False case loss, BOM and commented section headers, CRLF, .bak overwrite (.picode-orig added), symlink, localized missing-file detection, default=root/missing accounts, option/root allowlists, system-stage timeout, missing paths measure 0, journal --rotate, empty sections kept, ADR accuracy, `--yes` gate.
visual-review: PASS (root/, 12 states) after one FAIL round.
Merge: `make close` had not run when this note was written.

## Next up

- Owner tries the Distro tab and cleaning the 2.1 GB journal after `make deploy` + `make desktop-restart`.

## Debts

- The risky-settings warning is a `confirm()` in the page only; the tool itself does not know which settings are risky.
- `apt-get clean` also frees /var/cache/apt/*.bin outside the measured path, so the freed size is under-reported.
