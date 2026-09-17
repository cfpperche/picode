### Changed
- **Muse Code and Antigravity launch through the PATH wrapper like every
  other CLI.** Same presence lease, same native runtime registration, no
  more launch carve-outs. Reporting is unchanged: Antigravity through
  its title reporter (now with a runtime behind it), Muse Code honestly
  Open (its hooks carry no terminal identity). Maintenance subcommands
  (`muse exec`, `agy update`, …) skip the lease.

### Fixed
- Plan file lists no longer repeat the wrapper entry.
