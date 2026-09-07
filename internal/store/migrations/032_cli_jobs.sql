CREATE TABLE IF NOT EXISTS cli_jobs (
	id TEXT PRIMARY KEY,
	cli TEXT NOT NULL,
	action TEXT NOT NULL,
	request_key TEXT NOT NULL UNIQUE,
	state TEXT NOT NULL,
	created_at TEXT NOT NULL,
	payload TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_cli_jobs_created ON cli_jobs(created_at DESC);
