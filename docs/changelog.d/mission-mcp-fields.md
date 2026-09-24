### Fixed
- Mission CLI/MCP writes now name a missing `generation`, `expectedVersion` or `requestId` before contacting the daemon. Read actions remain available without those fields, and stale-generation refusals are unchanged.
