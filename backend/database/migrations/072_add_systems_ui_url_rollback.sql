-- Rollback 072: drop systems.ui_url
--
-- The custom addresses are lost: every system goes back to its default one.

ALTER TABLE systems DROP COLUMN IF EXISTS ui_url;
