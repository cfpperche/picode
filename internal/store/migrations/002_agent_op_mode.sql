-- Operation mode: which tool set pi starts with.
-- NULL / empty = full (default). "readonly" = --tools read,grep,find,ls.
-- Future extension modes reuse this column.
ALTER TABLE agents ADD COLUMN op_mode TEXT;
