-- Rollback migration 041: back to availability expressed only as an allowlist.
--
-- Dropping the flag puts every add-on back on sale, the buy action included:
-- an item that was withheld becomes purchasable again the moment this runs.

ALTER TABLE entitlement_catalog DROP COLUMN IF EXISTS purchasable;
