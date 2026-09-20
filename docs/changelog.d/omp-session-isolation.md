### Added

- Scope Omp agent `/resume` sessions to a durable private directory per workspace agent.

### Changed

- Other Agent CLIs keep their native session-storage behavior. PiCode does not emulate per-agent `/resume` isolation for runtimes without a dedicated session-directory boundary.
