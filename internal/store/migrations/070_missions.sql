-- ADR-0199/0200: mission history survives removal of agents and workspaces.
CREATE TABLE missions (
    seq INTEGER PRIMARY KEY AUTOINCREMENT,
    id TEXT NOT NULL UNIQUE,
    workspace_id TEXT NOT NULL,
    body TEXT NOT NULL
);
CREATE INDEX missions_workspace ON missions(workspace_id, seq);
CREATE TABLE mission_reservations (
    mission_id TEXT NOT NULL PRIMARY KEY,
    agent_id TEXT NOT NULL UNIQUE
);
CREATE TABLE mission_requests (
    actor TEXT NOT NULL,
    request_id TEXT NOT NULL,
    payload TEXT NOT NULL,
    result TEXT NOT NULL,
    PRIMARY KEY(actor, request_id)
);
CREATE TABLE mission_history (
    seq INTEGER PRIMARY KEY AUTOINCREMENT,
    mission_id TEXT NOT NULL,
    version INTEGER NOT NULL,
    action TEXT NOT NULL,
    actor TEXT NOT NULL,
    at TEXT NOT NULL,
    body TEXT NOT NULL,
    UNIQUE(mission_id, version)
);
CREATE INDEX mission_history_mission ON mission_history(mission_id, seq);

CREATE TABLE mission_questions (
    inbox_id TEXT PRIMARY KEY,
    mission_id TEXT NOT NULL,
    generation INTEGER NOT NULL,
    scope_version INTEGER NOT NULL,
    session TEXT NOT NULL
);
