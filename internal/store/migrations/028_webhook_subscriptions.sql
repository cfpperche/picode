-- Outbound webhooks (ADR-0071): one row per subscribed URL. The deliverer
-- tails the durable events log and POSTs each matching event signed with
-- secret (HMAC-SHA256). types is a JSON array of event-type prefixes;
-- an empty array means every durable event. cursor is the id of the last
-- durably delivered event (at-least-once, in order per subscription).
-- secret is stored raw — the same trust level as push_subscriptions.auth
-- (018) — because deliveries are signed, not compared.
CREATE TABLE webhook_subscriptions (
  id              TEXT PRIMARY KEY,
  url             TEXT NOT NULL,
  secret          TEXT NOT NULL,
  types           TEXT NOT NULL DEFAULT '[]',
  enabled         INTEGER NOT NULL DEFAULT 1,
  cursor          INTEGER NOT NULL DEFAULT 0,
  last_status     TEXT NOT NULL DEFAULT '',
  last_error      TEXT NOT NULL DEFAULT '',
  last_attempt_at TEXT NOT NULL DEFAULT '',
  created_at      TEXT NOT NULL
);
