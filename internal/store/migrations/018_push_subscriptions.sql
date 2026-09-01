-- Web Push subscriptions (ADR-0047). One row per browser that opted in;
-- endpoint is the push service's URL for that browser and is unique.
-- prefs is a small JSON object: {"actions":bool,"finished":bool}.
CREATE TABLE push_subscriptions (
  id          TEXT PRIMARY KEY,
  endpoint    TEXT NOT NULL UNIQUE,
  p256dh      TEXT NOT NULL,
  auth        TEXT NOT NULL,
  device_id   TEXT NOT NULL DEFAULT '',
  user_agent  TEXT NOT NULL DEFAULT '',
  prefs       TEXT NOT NULL DEFAULT '{"actions":true,"finished":true}',
  created_at  TEXT NOT NULL,
  last_ok_at  TEXT,
  failures    INTEGER NOT NULL DEFAULT 0
);
