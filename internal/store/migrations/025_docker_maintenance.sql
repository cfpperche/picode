CREATE TABLE docker_locks (
    endpoint TEXT NOT NULL,
    kind TEXT NOT NULL,
    target TEXT NOT NULL,
    owner TEXT NOT NULL,
    PRIMARY KEY (endpoint, kind, target)
);
INSERT INTO docker_locks SELECT endpoint, 'container', container_id, id
    FROM docker_operations WHERE state = 'running';

CREATE TABLE docker_plans (
    id TEXT PRIMARY KEY,
    payload TEXT NOT NULL
);
CREATE TABLE docker_jobs (
    id TEXT PRIMARY KEY,
    request_key TEXT NOT NULL UNIQUE,
    plan_id TEXT NOT NULL UNIQUE,
    state TEXT NOT NULL,
    created_at TEXT NOT NULL,
    payload TEXT NOT NULL
);
CREATE INDEX docker_jobs_recent ON docker_jobs(created_at DESC);
CREATE TABLE docker_monitors (
    endpoint TEXT NOT NULL,
    project TEXT NOT NULL,
    revision INTEGER NOT NULL,
    payload TEXT NOT NULL,
    PRIMARY KEY (endpoint, project)
);
CREATE TABLE docker_incidents (
    id TEXT PRIMARY KEY,
    endpoint TEXT NOT NULL,
    project TEXT NOT NULL,
    signal TEXT NOT NULL,
    state TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    payload TEXT NOT NULL
);
CREATE UNIQUE INDEX docker_incident_active ON docker_incidents(endpoint, project, signal)
    WHERE state IN ('pending', 'open');
