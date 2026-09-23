### Added
- **Abandon** on a llama.cpp operation whose result is unknown: PiCode stops following it and frees the model, without sending anything to the server.

### Fixed
- An operation left unknown on PiCode's own llama.cpp service no longer blocks starting that service forever: while the service is not running, its operations end as Interrupted.
- After PiCode restarts, an unload whose model is still loaded ends as Interrupted instead of waiting forever.
- Finishing a large download no longer holds up other model operations while PiCode checks the file.
- Leftovers of an interrupted llama.cpp install are cleaned up at start.
- A download that started is no longer reported as failed when PiCode could not record its file baseline.
