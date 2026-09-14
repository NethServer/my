-- Rollback migration 044: the dashboard forgets the legacy estate.
--
-- Dropping the table makes every legacy counter read zero: the API keeps
-- answering, the totals simply stop mentioning the old portal. Nothing is lost
-- that cannot be rebuilt — the numbers live on the legacy MySQL and the next
-- proxy_sync run repopulates the table once it exists again.

DROP TABLE IF EXISTS legacy_systems_by_org;
