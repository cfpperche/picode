### Changed

- A llama.cpp download that PiCode lost track of and the server no longer has is now marked **Interrupted** on its own, so the model can be downloaded again without choosing **Abandon** first.
- Finished llama.cpp jobs older than 30 days are cleared at startup (the newest 500 always stay).

### Fixed

- If PiCode's llama.cpp supervisor process is killed, the models its server had loaded are now stopped instead of staying in memory.
