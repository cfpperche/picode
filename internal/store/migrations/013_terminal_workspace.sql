-- Terminals belong to a workspace (ADR-0026). Existing terminals stay free.
-- No FK on purpose: SQLite refuses ADD COLUMN with a REFERENCES clause and a
-- non-NULL default while foreign keys are on, and recreating the table buys
-- nothing — workspace existence is validated on create and the cascade on
-- remove is app-driven, exactly like terminal_settings ownership already is.
ALTER TABLE terminals ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'ws_free';
