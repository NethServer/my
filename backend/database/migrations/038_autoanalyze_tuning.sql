-- Migration 038: Keep planner statistics fresh on the tables that drive the
-- org list/stats query plans.
--
-- systems and applications are rewritten continuously by collect (heartbeats,
-- inventory, certification_level). At the default 10% autoanalyze scale factor
-- their statistics drift far enough between runs that the planner flips the
-- reseller/distributor/customer count subqueries to sequential scans. Lower the
-- scale factor to 2% so autoanalyze runs more often on these hot tables.
--
-- distributors has only a handful of rows -- below the default autoanalyze
-- threshold (50) -- so it was never auto-analyzed at all, leaving the planner
-- with no statistics and choosing nested loops that fan out to thousands of
-- rows. Trigger after a few modifications regardless of table size.

ALTER TABLE systems      SET (autovacuum_analyze_scale_factor = 0.02);
ALTER TABLE applications SET (autovacuum_analyze_scale_factor = 0.02);
ALTER TABLE distributors SET (autovacuum_analyze_scale_factor = 0,
                              autovacuum_analyze_threshold = 5);
