### Fixed
- **Agent CLIs: an Omp terminal no longer stays Working after it replies.**
  omp never fires the events pi uses to settle a run, so the activity
  reporter waited forever at Working. omp now uses its own event set —
  Ready comes back when the reply finishes — and, measured against its
  approval dialog, it never claims Needs you: approvals happen in the omp
  terminal itself.
