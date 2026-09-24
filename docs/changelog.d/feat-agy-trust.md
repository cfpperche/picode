### Fixed

- **Instructions: Antigravity reads its files after all.** The earlier entry
  that its cells now say "unknown" was wrong: Antigravity reads `GEMINI.md`
  and `AGENTS.md` from the start of a session, but only in a folder you
  trusted in it — that exact folder, not the folders inside it. The page
  now shows "reads" or "not trusted" accordingly, and says when a folder
  needs trusting.
