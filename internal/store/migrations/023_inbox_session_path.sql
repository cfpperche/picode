-- ADR-0059: an ask_human item records the exact Pi session that filed it.
ALTER TABLE inbox_items ADD COLUMN session_path TEXT NOT NULL DEFAULT '';
