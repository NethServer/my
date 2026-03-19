-- Rollback migration 017: Support sessions and access logs

DROP TABLE IF EXISTS support_access_logs;
DROP TABLE IF EXISTS support_sessions;

DELETE FROM schema_migrations WHERE migration_number = 17;
