### Fixed
- **System's Agent CLIs section no longer flashes "none installed"** while the list loads: it shows placeholder rows, and says so if the list cannot be read. On the phone, the list now refreshes after you install or remove a CLI, so System and the Automations banner stay current without a reload.
- The Automations "Pi is not installed" banner ignores disabled automations.
- A free **New agent** takes the CLI's name when the Name field is left empty, as the placeholder suggests.
- The docs say what PiCode needs precisely: only tmux for PiCode itself, Node.js and npm to install CLIs from Agent CLIs, **Install** for Pi, Claude Code, Codex, OpenCode and Omp, and how Connectors differ on the CLIs other than Pi.
