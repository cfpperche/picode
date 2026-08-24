-- Which pi JSONL this agent should open (--session). NULL = new session on start.
ALTER TABLE agents ADD COLUMN session_path TEXT;
