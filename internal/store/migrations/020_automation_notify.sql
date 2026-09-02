-- ADR-0045 amendment (2026-09-02): where to POST when a run ends
-- (Slack-compatible incoming webhook). NULL = nowhere.
ALTER TABLE automations ADD COLUMN notify_url TEXT;
