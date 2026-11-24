-- Migration 071: drop impersonation_consents
--
-- Owner and Staff users impersonate without the target's consent: every
-- session is audited in impersonation_audit instead. The consent records have
-- no use left.

DROP TABLE IF EXISTS impersonation_consents;
