-- 052: the servers the human asked the panel not to show (Servers panel v2).
-- One row per hidden listener, keyed by the identity of the process holding
-- the socket — port + pid + start token — so a new process on the same port is
-- a new row in the panel, never a silent hole: the panel's Hide can only ever
-- mean "this one".
CREATE TABLE dev_server_hides (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  port INTEGER NOT NULL,
  pid INTEGER NOT NULL,
  start_key TEXT NOT NULL,
  owner TEXT NOT NULL DEFAULT '',
  tool TEXT NOT NULL DEFAULT '',
  hidden_at TEXT NOT NULL
);
CREATE UNIQUE INDEX idx_dev_server_hides_key ON dev_server_hides(port, pid, start_key);
