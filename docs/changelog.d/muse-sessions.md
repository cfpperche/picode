### Added
- **Muse Code sessions.** The Muse Code pane has a **Sessions** tab: every
  conversation the CLI has on this machine, with its folder, model, age and
  size, filtered to the workspace you are in. **Open in terminal** resumes that
  exact conversation (`muse --resume <id>`) in its own folder. Read-only —
  PiCode lists and resumes, it never rewrites Muse's session store.
- Sessions for a CLI whose history exists before it has an adapter: the pane
  now shows the tab whenever the server reports a session source, instead of
  only for CLIs with activity reporting.
