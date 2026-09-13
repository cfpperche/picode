### Fixed
- **A file, folder, git or app tab no longer asks the API for an agent's role.**
  Selecting one of those tabs fetched `/role-state` and `/slash` with a tab id
  the server does not know as an agent — two 404s per selection, a chat
  composer asking a repository its role. Both fetches now ask `isAgentTab`
  (a bare id, not a tagged surface), so they run where an agent is on screen
  and nowhere else; the composer's role chip and slash list are unchanged on
  an agent tab.
