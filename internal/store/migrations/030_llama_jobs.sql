CREATE TABLE llama_jobs (
    id TEXT PRIMARY KEY,
    request_key TEXT NOT NULL UNIQUE,
    endpoint TEXT NOT NULL,
    model TEXT NOT NULL,
    replace_others INTEGER NOT NULL,
    state TEXT NOT NULL,
    connection_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    payload TEXT NOT NULL
);
CREATE INDEX llama_jobs_active ON llama_jobs(endpoint, state);
