### Added
- **Muse Code and Antigravity sessions can be continued elsewhere.**
  Both are now handoff sources: `Continue in…` appears on their session
  rows, offering every CLI that can receive the conversation. Muse reads
  through its own `export --session` (official transcript); Antigravity
  reads the CLI's per-conversation transcript, deliberately not the
  SQLite protobuf (unversioned field numbers). Reasoning never travels;
  oversized sessions fall back to the brief, which both already support.
