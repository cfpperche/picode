### Fixed

- `scripts/qa-scratch.sh stop` ends the daemon **it** started (the pid the daemon itself wrote to `data/server.json`, with the remembered pid as fallback) and verifies the port afterwards, instead of killing whoever holds the recorded port: a stale port file used to make `stop` kill a neighbouring scratch's daemon and leave its own alive. If a different process holds that port, the stop now leaves it alone and says so; terminals are still removed through the API first, so no tmux session is orphaned.
- `start` refuses a port someone else holds instead of killing its holder, records the daemon's pid, and `status` prints port, pid, liveness and health.
- A tmux socket that cannot be created is an error, not silence: `NewWithSocket` creates the socket's directory, and tmux's `error creating …`/`error connecting to …` text (printed with exit status 0 by `new-session`) is promoted to an error. Before, a session on an unusable socket path "succeeded" with nothing behind it.

### Changed

- The tmux watcher integration test and the kill-server test address a server they own with `-S` instead of rebinding the process-wide `TMUX_TMPDIR`: a test that ends a server can no longer strand other tests — or a background goroutine — in the same binary on the server it dismantles. The kill test keeps asserting the directory guard; the watcher test asserts the suite's namespace is untouched.
