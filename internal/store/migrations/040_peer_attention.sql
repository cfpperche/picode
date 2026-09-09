-- ADR-0107: claiming precedes terminal writes; crashes never cause an automatic repeat.
ALTER TABLE peer_messages ADD COLUMN attention_status TEXT NOT NULL DEFAULT 'pending'
 CHECK (attention_status IN ('pending','attempted','notified','uncertain'));
CREATE INDEX peer_attention_pending ON peer_messages(attention_status,acked_at,seq);
