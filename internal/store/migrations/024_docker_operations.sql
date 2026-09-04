CREATE TABLE docker_operations (
  id TEXT PRIMARY KEY,
  request_key TEXT NOT NULL UNIQUE,
  endpoint TEXT NOT NULL,
  container_id TEXT NOT NULL,
  container_name TEXT NOT NULL,
  action TEXT NOT NULL CHECK (action IN ('start', 'stop', 'restart')),
  actor TEXT NOT NULL,
  agent_id TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL CHECK (state IN ('running', 'succeeded', 'failed', 'unknown')),
  message TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  finished_at TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX docker_operation_active ON docker_operations(endpoint, container_id) WHERE state = 'running';
CREATE INDEX docker_operation_recent ON docker_operations(created_at DESC);
