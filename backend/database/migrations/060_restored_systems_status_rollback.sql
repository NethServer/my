-- Rollback 060: nothing to revert.
--
-- The systems moved to 'inactive' are not recorded, and putting them back
-- to 'unknown' would only restore the wrong status: the heartbeat monitor
-- keeps them correct from here on either way.

SELECT 1;
