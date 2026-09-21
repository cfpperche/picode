### Added
- **Deploys and desktop restarts can no longer race.** `make deploy`, `make desktop-restart` and a direct `picode deploy` serialize on one lock; a CLI deploy caught behind it says so and waits. The desktop swap refuses to kill the shell while the WSL keepalive is not alive (the state that loses every session to the distro's idle reclaim) and ends with a real daemon-health verdict.
