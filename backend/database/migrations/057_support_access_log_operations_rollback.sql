-- Rollback Migration 057: Restore the original support access types

DELETE FROM support_access_logs
 WHERE access_type IN ('session_extend', 'session_close', 'service_add', 'service_remove', 'cluster_login', 'autofill', 'ui_close');
ALTER TABLE support_access_logs DROP CONSTRAINT IF EXISTS support_access_logs_access_type_check;
ALTER TABLE support_access_logs ADD CONSTRAINT support_access_logs_access_type_check
    CHECK (access_type IN ('view', 'ssh', 'web_terminal', 'ui_proxy'));

COMMENT ON COLUMN support_access_logs.access_type IS 'Type of access: view, ssh, web_terminal, ui_proxy';
