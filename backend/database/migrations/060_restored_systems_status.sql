-- Migration 060: restored systems stuck as pending
--
-- Restoring a soft-deleted system used to reset its status to 'unknown'
-- even when it had reported before. The heartbeat monitor in collect only
-- promotes 'unknown' on a fresh heartbeat, so a restored system that stayed
-- silent sat as pending forever. A restore now sets 'inactive' when the
-- system has a heartbeat row; this data-only migration aligns the systems
-- restored before that change. A live one is recovered to 'active' by the
-- next monitor cycle.
--
-- An 'unknown' system with a heartbeat row can only come from a restore:
-- the heartbeat worker promotes 'unknown' to 'active' on every heartbeat.

UPDATE systems s
SET status = 'inactive'
WHERE s.status = 'unknown'
  AND s.deleted_at IS NULL
  AND s.unregistered_at IS NULL
  AND EXISTS (SELECT 1 FROM system_heartbeats h WHERE h.system_id = s.id);
