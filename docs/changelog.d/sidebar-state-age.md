### Changed

- **Sidebar status pills show how long the agent has been in its state** — `Ready · now`, `Ready · 5m`, `Open · 2h`, `Stopped · 3d` — not only while `Working`. The age is the moment the state actually began: a CLI agent's own hook report for terminal rows, and a managed agent's start/stop and turn settle (a new `agent.settled` update, applied live without a refresh) for the rest. Where no truthful timestamp exists the pill shows the label alone; hover carries the full time. Mobile status chips no longer clip the age text.
