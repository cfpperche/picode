### Fixed

- Pi agents in terminal mode now reuse the shared CLI launcher and activity integration, including working, needs-you and idle states across desktop and mobile. Existing open Pi terminals remain usable until explicitly restarted.
- Pi terminal and chat transitions now share lifecycle guards, preserve agent session ownership and refuse replacement while a previous process is still closing. Agent launch settings remain available alongside Pi-specific configuration.
