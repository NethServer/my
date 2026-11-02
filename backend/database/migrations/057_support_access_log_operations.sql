-- Migration 057: Record operator actions in the support access log
-- Extending, closing, adding or removing a service and the proxy logging into
-- the NS8 cluster for the operator, and the operator filling a login form
-- with the session credentials or closing a web interface, join the accesses
-- in the session history.

ALTER TABLE support_access_logs DROP CONSTRAINT IF EXISTS support_access_logs_access_type_check;
ALTER TABLE support_access_logs ADD CONSTRAINT support_access_logs_access_type_check
    CHECK (access_type IN ('view', 'ssh', 'web_terminal', 'ui_proxy',
                           'session_extend', 'session_close', 'service_add', 'service_remove', 'cluster_login', 'autofill', 'ui_close'));

COMMENT ON COLUMN support_access_logs.access_type IS 'Type of access: view, ssh, web_terminal, ui_proxy, session_extend, session_close, service_add, service_remove, cluster_login, autofill, ui_close';
