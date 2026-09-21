### Fixed

- **Two gates added yesterday were passing work they claimed to check.** The
  ADR status check could not read `- **Status:** accepted` (the colon inside
  the bold), so ten of 171 decision records were silently unchecked; an
  unreadable status line is now a failure rather than a skip, and all 171 are
  covered. The ADR-0048 mutation gate scanned only each method's own body for
  SQL, so seventeen exported mutators that delegate their write — `AddAgent`,
  `CreateTerminal`, `EnablePeer`, `ReplaceFrom` and fourteen others — were
  never required to announce anything; three of them announce nothing and are
  now listed with a reason. It also read comments, so "we deliberately do not
  AppendEvent here" would have satisfied it.
