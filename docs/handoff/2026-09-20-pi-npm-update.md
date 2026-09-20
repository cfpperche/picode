# 2026-09-20 — pi-npm-update: Agent CLIs Update uses npm for global Pi

Shipped: npm-managed Pi update/reinstall run `npm install -g` (`npmUpdateOnly`,
same shape as claude-code/omp). `pi update` refuses when the package dir is
not W_OK (a sudo npm install into an nvm prefix) and prints the npm command
instead of running it. Architecture `cli-terminal-launch.md`; changelog fragment.
Verified: `make close` ci-scoped PASS. Live on this machine, npm as the
service user replaced root-owned 0.85.1 with 0.86.1 (parent-dir rename);
the retired copy needed sudo rm. Not run: System `POST /api/system/pi-update`
(`pi update --self`).
visual-review: n/a (no JSX/CSS)
Merge: fast-forward ready at close-summary against main b456b3bc.
