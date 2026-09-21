### Added

- **Git ▸ Delivery ▸ Deployment**: the second lens of the Delivery view reports
  the local PiCode instance's running revision, the integrated changes that
  revision does not contain, and the last recorded deploy attempt. Connect a
  project with the Environment selector — nothing in the lens deploys, queues,
  retries or approves anything, and an attempt that never finished reads as an
  unknown outcome.
- `picode deploy` records each attempt as a deployment receipt under the data
  directory (`var/delivery/`), so the Deployment lens can show what was
  deployed, from which revision, and what answered afterwards.

### Changed

- The Git ▸ Delivery view now has two lenses, **Integration** and **Deployment**;
  an integrated change shows whether the running revision contains it.
