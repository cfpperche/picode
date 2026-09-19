-- ADR-0160: an agent names its catalog runtime. Existing rows are Pi.
-- Guest CLIs become agents in later slices; Runtime.Start stays Pi-only.
ALTER TABLE agents ADD COLUMN cli TEXT NOT NULL DEFAULT 'pi';
