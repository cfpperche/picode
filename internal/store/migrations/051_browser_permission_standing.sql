-- 051: the permissions table keeps two kinds of rows — a saved standing
-- (the dialog's per-kind choice, and the Ask prompt's "Always allow", which
-- the page hands back to the shell on load) and a one-off decision the shell
-- reports (Allow/Block once, or the platform's own deny). Only standings are
-- policy; the flag is what the load-time push filters on, so a decision can
-- never quietly become a permanent one. Existing every-site rows (*) were
-- written by the dialog as policy.
ALTER TABLE browser_permissions ADD COLUMN standing INTEGER NOT NULL DEFAULT 0;
UPDATE browser_permissions SET standing = 1 WHERE origin = '*';
