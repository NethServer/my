-- Migration 072: add systems.ui_url
--
-- The address of the system's web interface, set by an admin when the default
-- one derived from the FQDN (https://<fqdn>/cluster-admin for NethServer,
-- https://<fqdn>:9090 for NethSecurity) does not open it. NULL means the
-- default, so it keeps following the FQDN of later inventories.

ALTER TABLE systems ADD COLUMN IF NOT EXISTS ui_url VARCHAR(2048);

COMMENT ON COLUMN systems.ui_url IS 'Custom address of the system web interface, set by an admin. NULL = the default derived from fqdn and type';
