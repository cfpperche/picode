### Changed
- **`picode provision` no longer blocks on `pi`.** The doctor reports which agent CLIs it found on PATH and converges with none of them installed (ADR-0179). The member container on a shared server no longer installs pi; it keeps node and npm so members install the CLIs they use from Agent CLIs.
- **PiCode Desktop (Windows) installs tmux, git, curl, Node.js 22 and npm — no agent CLI.** A machine without pi now finishes the install, `--user` aims only the picode binary, and a Node.js already present is left alone.
