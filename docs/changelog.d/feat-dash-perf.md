### Fixed
- **Dashboard loads faster.** The six CLI meters now aggregate in parallel (cold 7d window drops from ~8s to ~5s on this machine), the server pre-warms the stats cache after boot, and the dashboard paints the last good numbers instantly while refreshing underneath instead of a full skeleton.
